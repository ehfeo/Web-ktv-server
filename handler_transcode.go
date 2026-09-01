package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// mediaInfoCache 媒体信息缓存，避免每次点歌都调用ffprobe导致磁盘100%
var mediaInfoCache struct {
	sync.RWMutex
	cache map[string]*mediaInfoCacheEntry
}

type mediaInfoCacheEntry struct {
	videoCodec               string
	allAudioIsAAC            bool
	allAudioIsMP3            bool
	allAudioBrowserSupported bool
	videoBitrate             string
	audioBitrate             string
	audioCodecStr            string
	modTime                  time.Time // 文件修改时间，用于失效检测
}

const mediaInfoCacheMaxEntries = 2000

func getFFmpegPath() string {
	currentDir, _ := os.Getwd()
	ffmpegPath := filepath.Join(currentDir, "ffmpeg.exe")
	if _, err := os.Stat(ffmpegPath); err == nil {
		return ffmpegPath
	}

	if path, err := exec.LookPath("ffmpeg"); err == nil {
		return path
	}

	return "ffmpeg"
}

var transcodeProgress struct {
	sync.Mutex
	progress      int
	status        string
	message       string
	log           string
	command       string
	mediaInfo     string
	outputPath    string
	trackWarning  *MediaTrackWarning
}

var lastCompletedTranscode struct {
	sync.Mutex
	requestKey string
	outputPath string
	status     string
}

type TranscodeTask struct {
	FilePath    string `json:"filePath"`
	FileName    string `json:"fileName"`
	RequestKey  string `json:"requestKey"`
	VideoCodec  string `json:"videoCodec"`
	AudioCodec  string `json:"audioCodec"`
}

var globalTranscodeQueue struct {
	sync.Mutex
	queue      []TranscodeTask
	isRunning  bool
	processIdx int
}

// diskSleepStatus 用于在磁盘休眠唤醒期间向前端推送提示状态
// 当 getMediaInfo 检测到磁盘休眠（预热读取超过 500ms 仍未返回）时设置 waking=true
// 预热完成后清除 waking
var diskSleepStatus struct {
	sync.RWMutex
	waking  bool      // 是否正在等待磁盘唤醒
	file    string    // 当前正在等待唤醒的文件
	since   time.Time // 检测到休眠的时刻
}

// setDiskSleeping 设置磁盘休眠状态。sleeping=true 表示进入等待唤醒状态。
func setDiskSleeping(sleeping bool, filePath string) {
	diskSleepStatus.Lock()
	defer diskSleepStatus.Unlock()
	if sleeping {
		diskSleepStatus.waking = true
		diskSleepStatus.file = filePath
		diskSleepStatus.since = time.Now()
		log.Printf("[磁盘休眠] 检测到磁盘休眠，正在等待唤醒: file=%s", filepath.Base(filePath))
	} else {
		if diskSleepStatus.waking {
			log.Printf("[磁盘休眠] 唤醒完成: file=%s 等待耗时=%dms",
				filepath.Base(diskSleepStatus.file), time.Since(diskSleepStatus.since).Milliseconds())
		}
		diskSleepStatus.waking = false
		diskSleepStatus.file = ""
	}
}

// DiskSleepStatusHandler 返回当前磁盘休眠状态，供前端轮询
func DiskSleepStatusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	diskSleepStatus.RLock()
	defer diskSleepStatus.RUnlock()
	resp := map[string]interface{}{
		"waking":  diskSleepStatus.waking,
		"message": "",
		"file":    "",
		"elapsed": 0,
	}
	if diskSleepStatus.waking {
		resp["message"] = "硬盘已休眠，正在等待硬盘唤醒响应..."
		resp["file"] = filepath.Base(diskSleepStatus.file)
		resp["elapsed"] = time.Since(diskSleepStatus.since).Milliseconds()
	}
	json.NewEncoder(w).Encode(resp)
}

func init() {
	globalTranscodeQueue.queue = make([]TranscodeTask, 0)
	globalTranscodeQueue.isRunning = false
	globalTranscodeQueue.processIdx = -1
	mediaInfoCache.cache = make(map[string]*mediaInfoCacheEntry)
	mediaTracksCache.cache = make(map[string]*mediaTracksCacheEntry)
	findMediaFileCache.cache = make(map[string]string)
}

func CheckAndAddTranscodeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	if r.Method != http.MethodPost {
		http.Error(w, "不支持的方法", 405)
		return
	}

	var req struct {
		FileName   string `json:"fileName"`
		RequestKey string `json:"requestKey"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "解析请求失败", 400)
		return
	}

	if req.RequestKey == "" {
		req.RequestKey = req.FileName
	}

	log.Printf("[CheckAndAddTranscode] 开始检查: file=%s", filepath.Base(req.FileName))
	foundPath := findMediaFile(req.FileName)
	if foundPath == "" {
		http.Error(w, "文件未找到", 404)
		return
	}

	videoCodec, _, _, allAudioBrowserSupported, _, _, audioCodecStr := getMediaInfo(foundPath)
	videoIsH264 := strings.EqualFold(videoCodec, "h264")
	videoCodecUpper := strings.ToUpper(videoCodec)

	ext := strings.ToLower(filepath.Ext(foundPath))
	audioExtensions := []string{".mp3", ".wav", ".flac", ".aac", ".m4a", ".m4r", ".alac", ".ogg", ".oga", ".opus", ".wma", ".ape", ".aiff", ".aif", ".amr", ".dts", ".dff", ".dsf", ".sacd", ".tak", ".tta", ".wv", ".mka"}
	isAudioFile := false
	for _, ae := range audioExtensions {
		if ext == ae {
			isAudioFile = true
			break
		}
	}

	if isAudioFile || (videoIsH264 && allAudioBrowserSupported) {
		result := struct {
			NeedsTranscode bool   `json:"needsTranscode"`
			Status         string `json:"status"`
			RequestKey     string `json:"requestKey"`
		}{
			NeedsTranscode: false,
			Status:         "ready",
			RequestKey:     req.RequestKey,
		}
		json.NewEncoder(w).Encode(result)
		return
	}

	var codecInfo string
	if !videoIsH264 && audioCodecStr != "" {
		codecInfo = "V:" + videoCodecUpper + ",A:" + audioCodecStr
	} else if !videoIsH264 {
		codecInfo = "V:" + videoCodecUpper
	} else if audioCodecStr != "" {
		codecInfo = "A:" + audioCodecStr
	}

	globalTranscodeQueue.Lock()

	for _, task := range globalTranscodeQueue.queue {
		if task.RequestKey == req.RequestKey {
			position := 0
			for i, t := range globalTranscodeQueue.queue {
				if t.RequestKey == req.RequestKey {
					position = i + 1
					break
				}
			}

			isCurrent := globalTranscodeQueue.processIdx >= 0 &&
				globalTranscodeQueue.processIdx < len(globalTranscodeQueue.queue) &&
				globalTranscodeQueue.queue[globalTranscodeQueue.processIdx].RequestKey == req.RequestKey

			status := "waiting"
			if isCurrent {
				status = "transcoding"
			}

			var taskCodecInfo string
			if task.VideoCodec != "" && task.AudioCodec != "" {
				taskCodecInfo = "V:" + task.VideoCodec + ",A:" + task.AudioCodec
			} else if task.VideoCodec != "" {
				taskCodecInfo = "V:" + task.VideoCodec
			} else if task.AudioCodec != "" {
				taskCodecInfo = "A:" + task.AudioCodec
			}
			result := struct {
				NeedsTranscode bool   `json:"needsTranscode"`
				Status         string `json:"status"`
				QueuePosition  int    `json:"queuePosition"`
				RequestKey     string `json:"requestKey"`
				CodecInfo      string `json:"codecInfo"`
			}{
				NeedsTranscode: true,
				Status:         status,
				QueuePosition:  position,
				RequestKey:     req.RequestKey,
				CodecInfo:      taskCodecInfo,
			}
			globalTranscodeQueue.Unlock()
			json.NewEncoder(w).Encode(result)
			return
		}
	}

	globalTranscodeQueue.queue = append(globalTranscodeQueue.queue, TranscodeTask{
		FilePath:   foundPath,
		FileName:   filepath.Base(foundPath),
		RequestKey: req.RequestKey,
		VideoCodec: videoCodecUpper,
		AudioCodec: audioCodecStr,
	})

	if !globalTranscodeQueue.isRunning {
		globalTranscodeQueue.isRunning = true
		go processGlobalTranscodeQueue()
	}
	globalTranscodeQueue.Unlock()

	result := struct {
		NeedsTranscode bool   `json:"needsTranscode"`
		Status         string `json:"status"`
		QueuePosition  int    `json:"queuePosition"`
		RequestKey     string `json:"requestKey"`
		CodecInfo      string `json:"codecInfo"`
	}{
		NeedsTranscode: true,
		Status:         "waiting",
		QueuePosition:  len(globalTranscodeQueue.queue),
		RequestKey:     req.RequestKey,
		CodecInfo:      codecInfo,
	}
	json.NewEncoder(w).Encode(result)
}

func TranscodeStatusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	requestKey := r.URL.Query().Get("requestKey")
	if requestKey == "" {
		http.Error(w, "缺少requestKey参数", 400)
		return
	}

	globalTranscodeQueue.Lock()
	defer globalTranscodeQueue.Unlock()

	position := 0
	status := "not_found"
	progress := 0
	codecInfo := ""

	for i, task := range globalTranscodeQueue.queue {
		if task.RequestKey == requestKey {
			position = i + 1
			status = "waiting"
			if task.VideoCodec != "" && task.AudioCodec != "" {
				codecInfo = "V:" + task.VideoCodec + ",A:" + task.AudioCodec
			} else if task.VideoCodec != "" {
				codecInfo = "V:" + task.VideoCodec
			} else if task.AudioCodec != "" {
				codecInfo = "A:" + task.AudioCodec
			}
			break
		}
	}

	isCurrent := false
	if globalTranscodeQueue.processIdx >= 0 && globalTranscodeQueue.processIdx < len(globalTranscodeQueue.queue) {
		isCurrent = globalTranscodeQueue.queue[globalTranscodeQueue.processIdx].RequestKey == requestKey
	}

	outputPath := ""
	if isCurrent {
		status = "transcoding"
		transcodeProgress.Lock()
		progress = transcodeProgress.progress
		if transcodeProgress.status == "completed" {
			status = "completed"
			outputPath = transcodeProgress.outputPath
		}
		transcodeProgress.Unlock()
		position = 0
		task := globalTranscodeQueue.queue[globalTranscodeQueue.processIdx]
		if task.VideoCodec != "" && task.AudioCodec != "" {
			codecInfo = "V:" + task.VideoCodec + ",A:" + task.AudioCodec
		} else if task.VideoCodec != "" {
			codecInfo = "V:" + task.VideoCodec
		} else if task.AudioCodec != "" {
			codecInfo = "A:" + task.AudioCodec
		}
	}

	if status == "not_found" {
		// 检查是否是最近完成的转码任务
		lastCompletedTranscode.Lock()
		if lastCompletedTranscode.requestKey == requestKey && lastCompletedTranscode.status == "completed" {
			status = "completed"
			outputPath = lastCompletedTranscode.outputPath
			progress = 100
			// 命中已完成缓存，无需额外日志
		} else {
			status = "ready"
			progress = 100
			position = 0
		}
		lastCompletedTranscode.Unlock()
	}

	result := struct {
		Status        string `json:"status"`
		Progress      int    `json:"progress"`
		QueuePosition int    `json:"queuePosition"`
		RequestKey    string `json:"requestKey"`
		CodecInfo     string `json:"codecInfo"`
		OutputPath    string `json:"outputPath"`
	}{
		Status:        status,
		Progress:      progress,
		QueuePosition: position,
		RequestKey:    requestKey,
		CodecInfo:     codecInfo,
		OutputPath:    outputPath,
	}
	json.NewEncoder(w).Encode(result)
}

func TranscodeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	if r.Method == http.MethodPost {
		var req struct {
			FileName string `json:"fileName"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "解析请求失败", 400)
			return
		}

		foundPath := findMediaFile(req.FileName)
		if foundPath == "" {
			http.Error(w, "文件未找到", 404)
			return
		}

		globalTranscodeQueue.Lock()
		queueIndex := len(globalTranscodeQueue.queue)
		globalTranscodeQueue.queue = append(globalTranscodeQueue.queue, TranscodeTask{
			FilePath:   foundPath,
			FileName:   filepath.Base(foundPath),
			RequestKey: req.FileName,
		})
		if !globalTranscodeQueue.isRunning {
			globalTranscodeQueue.isRunning = true
			go processGlobalTranscodeQueue()
		}
		globalTranscodeQueue.Unlock()

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf(`{"success": true, "queueIndex": %d, "queueLength": %d}`, queueIndex, len(globalTranscodeQueue.queue))))
		return
	}

	http.Error(w, "不支持的方法", 405)
}

func processGlobalTranscodeQueue() {
	for {
		globalTranscodeQueue.Lock()
		if len(globalTranscodeQueue.queue) == 0 {
			globalTranscodeQueue.isRunning = false
			globalTranscodeQueue.processIdx = -1
			globalTranscodeQueue.Unlock()
			return
		}

		task := globalTranscodeQueue.queue[0]
		globalTranscodeQueue.processIdx = 0
		queueLength := len(globalTranscodeQueue.queue)
		globalTranscodeQueue.Unlock()

		fmt.Printf("[转码] 开始处理: %s (队列剩余%d)\n", task.FileName, queueLength)

		transcodeProgress.Lock()
		transcodeProgress.progress = 0
		transcodeProgress.status = "running"
		transcodeProgress.message = ""
		transcodeProgress.outputPath = ""
		transcodeProgress.Unlock()

		runTranscode(task.FilePath, task.RequestKey)

		globalTranscodeQueue.Lock()
		if len(globalTranscodeQueue.queue) > 0 {
			globalTranscodeQueue.queue = globalTranscodeQueue.queue[1:]
		}
		globalTranscodeQueue.processIdx = -1
		globalTranscodeQueue.Unlock()
	}
}

func TranscodeProgressHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	transcodeProgress.Lock()
	defer transcodeProgress.Unlock()

	globalTranscodeQueue.Lock()
	queueLength := len(globalTranscodeQueue.queue)
	isRunning := globalTranscodeQueue.isRunning
	globalTranscodeQueue.Unlock()

	result := struct {
		Progress    int    `json:"progress"`
		Status      string `json:"status"`
		Message       string             `json:"message"`
		Log           string             `json:"log"`
		Command       string             `json:"command"`
		MediaInfo     string             `json:"mediaInfo"`
		QueueLength   int                `json:"queueLength"`
		IsRunning     bool               `json:"isRunning"`
		OutputPath    string             `json:"outputPath"`
		TrackWarning  *MediaTrackWarning `json:"trackWarning,omitempty"`
	}{
		Progress:      transcodeProgress.progress,
		Status:        transcodeProgress.status,
		Message:       transcodeProgress.message,
		Log:           transcodeProgress.log,
		Command:       transcodeProgress.command,
		MediaInfo:     transcodeProgress.mediaInfo,
		QueueLength:   queueLength,
		IsRunning:     isRunning,
		OutputPath:    transcodeProgress.outputPath,
		TrackWarning:  transcodeProgress.trackWarning,
	}

	json.NewEncoder(w).Encode(result)
}

// CheckTracksHandler 检查文件轨道完整性
func CheckTracksHandler(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, "缺少name参数", 400)
		return
	}

	log.Printf("[CheckTracks] 开始检查轨道: file=%s", filepath.Base(name))
	foundPath := findMediaFile(name)
	if foundPath == "" {
		http.Error(w, "文件未找到", 404)
		return
	}

	warning := checkMediaTracks(foundPath)
	w.Header().Set("Content-Type", "application/json")
	// 始终返回结构体，确保 audioTrackCount 字段正确传给前端
	// （warning.Message 为空表示文件正常，前端据此决定是否显示警告条）
	json.NewEncoder(w).Encode(warning)
}

var findMediaFileCache struct {
	sync.RWMutex
	cache map[string]string
}

func findMediaFile(name string) string {
	findStart := time.Now()
	// 空name直接返回，避免无效Walk
	if name == "" {
		return ""
	}

	// 非媒体文件（如.lrc歌词）静默返回空，不输出WARN/ERROR日志
	ext := strings.ToLower(filepath.Ext(name))
	if ext != "" && !isMediaExtension(ext) {
		return ""
	}

	// 1. 优先从内存曲库映射表查找（O(1)）
	lookupStart := time.Now()
	if absPath, ok := lookupAbsPath(name); ok {
		lookupMs := time.Since(lookupStart).Milliseconds()
		statStart := time.Now()
		_, statErr := os.Stat(absPath)
		statMs := time.Since(statStart).Milliseconds()
		if statErr == nil {
			log.Printf("[findMediaFile] 步骤1 内存映射命中 lookup=%dms stat=%dms 总耗时=%dms name=%s",
				lookupMs, statMs, time.Since(findStart).Milliseconds(), name)
			return absPath
		}
		log.Printf("[findMediaFile] 内存映射命中但文件不存在: name=%s absPath=%s stat=%dms err=%v", name, absPath, statMs, statErr)
	} else {
		log.Printf("[findMediaFile] 步骤1 内存映射未命中 lookup=%dms name=%s", time.Since(lookupStart).Milliseconds(), name)
	}

	// 2. 查findMediaFileCache
	cacheStart := time.Now()
	findMediaFileCache.RLock()
	if p, ok := findMediaFileCache.cache[name]; ok {
		findMediaFileCache.RUnlock()
		statStart := time.Now()
		_, statErr := os.Stat(p)
		statMs := time.Since(statStart).Milliseconds()
		if statErr == nil {
			log.Printf("[findMediaFile] 步骤2 缓存命中 lookup=%dms stat=%dms 总耗时=%dms name=%s",
				time.Since(cacheStart).Milliseconds(), statMs, time.Since(findStart).Milliseconds(), name)
			return p
		}
		log.Printf("[findMediaFile] 缓存命中但文件不存在: name=%s path=%s stat=%dms", name, p, statMs)
	} else {
		findMediaFileCache.RUnlock()
		log.Printf("[findMediaFile] 步骤2 缓存未命中 lookup=%dms name=%s", time.Since(cacheStart).Milliseconds(), name)
	}

	// 3. 裸文件名查找（无目录前缀时，通过basename反向映射查找）
	if !strings.Contains(name, "/") {
		basenameStart := time.Now()
		if absPath, ok := lookupAbsPathByBasename(name); ok {
			log.Printf("[findMediaFile] 步骤3 basename映射命中 总耗时=%dms name=%s", time.Since(basenameStart).Milliseconds(), name)
			return absPath
		}
		log.Printf("[findMediaFile] 步骤3 basename映射未命中 耗时=%dms name=%s", time.Since(basenameStart).Milliseconds(), name)
	}

	// 4. fallback到原始查找逻辑（可能触发Walk磁盘扫描）
	log.Printf("[findMediaFile][WARN] 内存映射、缓存、basename均未命中，触发fallback查找: name=%s", name)
	fallbackStart := time.Now()
	result := findMediaFileUncached(name)
	log.Printf("[findMediaFile] 步骤4 fallback完成 耗时=%dms 总耗时=%dms result=%s name=%s",
		time.Since(fallbackStart).Milliseconds(), time.Since(findStart).Milliseconds(),
		func() string { if result != "" { return "FOUND" }; return "NOT_FOUND" }(), name)
	if result != "" {
		findMediaFileCache.Lock()
		findMediaFileCache.cache[name] = result
		if len(findMediaFileCache.cache) > 5000 {
			findMediaFileCache.cache = make(map[string]string)
		}
		findMediaFileCache.Unlock()
	} else {
		log.Printf("[findMediaFile][ERROR] fallback也未找到文件: name=%s", name)
	}
	return result
}

func findMediaFileUncached(name string) string {
	if name == "" {
		return ""
	}
	var foundPath string
	var fileNameOnly string = name

	// 非媒体文件（如.lrc歌词）不执行Walk扫描
	ext := strings.ToLower(filepath.Ext(name))
	if ext != "" && !isMediaExtension(ext) {
		return ""
	}
	// 空扩展名也不执行Walk（可能是异常请求）
	if ext == "" {
		return ""
	}

	parts := strings.SplitN(name, "/", 2)
	if len(parts) == 2 {
		dirPrefix := parts[0]
		remainingPath := parts[1]
		fileNameOnly = filepath.Base(remainingPath)

		// 第一轮：先对所有匹配的目录做os.Stat直接查找（O(1)，不Walk）
		var matchedDirs []string
		for _, dir := range mediaDirs {
			if filepath.Base(dir) == dirPrefix {
				matchedDirs = append(matchedDirs, dir)
				fullPath := filepath.Join(dir, remainingPath)
				if _, err := os.Stat(fullPath); err == nil {
					return fullPath
				}
			}
		}

		// 第二轮：所有os.Stat都失败，才Walk（最后手段）
		for _, dir := range matchedDirs {
			log.Printf("[findMediaFileUncached][DISK-SCAN] Walk扫描目录: dir=%s target=%s", dir, fileNameOnly)
			err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return nil
				}
				if !info.IsDir() && info.Name() == fileNameOnly {
					foundPath = path
					return filepath.SkipAll
				}
				return nil
			})

			if err == nil && foundPath != "" {
			return foundPath
		}
	}
	}

	// 第一轮：先对所有目录做os.Stat直接查找
	for _, dir := range mediaDirs {
		fullPath := filepath.Join(dir, name)
		if _, err := os.Stat(fullPath); err == nil {
			return fullPath
		}
	}

	// 第二轮：所有os.Stat都失败，才Walk（最后手段）
	for _, dir := range mediaDirs {
		log.Printf("[findMediaFileUncached][DISK-SCAN] Walk扫描目录(无前缀): dir=%s target=%s", dir, fileNameOnly)
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if !info.IsDir() && info.Name() == fileNameOnly {
				foundPath = path
				return filepath.SkipAll
			}
			return nil
		})

		if err == nil && foundPath != "" {
			return foundPath
		}
	}

	return ""
}

// isMediaExtension 判断是否为媒体文件扩展名
func isMediaExtension(ext string) bool {
	return isMediaExt2(ext)
}

// isMediaExt2 根据扩展名判断是否为支持的媒体文件
func isMediaExt2(ext string) bool {
	ext = strings.ToLower(ext)
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	switch ext {
	case ".mp3", ".wav", ".flac", ".aac", ".m4a", ".m4r", ".alac", ".ogg", ".oga", ".opus", ".wma", ".ape", ".aiff", ".aif", ".amr", ".dts", ".dff", ".dsf", ".sacd", ".tak", ".tta", ".wv", ".mka",
		".mp4", ".mkv", ".avi", ".mov", ".wmv", ".rm", ".rmvb", ".ts", ".webm",
		".mpg", ".mpeg", ".flv", ".m2ts", ".mts", ".m2t", ".3gp", ".3g2", ".m4v", ".vob", ".ogv", ".asf", ".divx", ".f4v", ".mxf", ".wtv":
		return true
	}
	return false
}

func runTranscode(filePath string, requestKey string) {
	defer func() {
		if r := recover(); r != nil {
			transcodeProgress.Lock()
			transcodeProgress.status = "error"
			transcodeProgress.message = fmt.Sprintf("%v", r)
			transcodeProgress.Unlock()
		}
	}()

	info, err := os.Stat(filePath)
	if err != nil {
		transcodeProgress.Lock()
		transcodeProgress.status = "error"
		transcodeProgress.message = "获取文件信息失败: " + err.Error()
		transcodeProgress.Unlock()
		return
	}

	dir := filepath.Dir(filePath)
	ext := filepath.Ext(filePath)
	baseName := strings.TrimSuffix(info.Name(), ext)

	// 转码输出容器格式：MPG/AVI/WMV/RM等容器不支持H264+AAC，必须使用MKV
	outputExt := ext
	unsupportedContainers := map[string]bool{".mpg": true, ".mpeg": true, ".avi": true, ".wmv": true, ".rm": true, ".rmvb": true, ".flv": true, ".ts": true}
	if unsupportedContainers[strings.ToLower(ext)] {
		outputExt = ".mkv"
	}
	tempPath := filepath.Join(dir, baseName+"_temp"+outputExt)
	finalPath := filepath.Join(dir, baseName+outputExt)

	videoCodec, allAudioIsAAC, _, _, videoBitrate, audioBitrate, _ := getMediaInfo(filePath)
	duration := getDuration(filePath)
	videoIsH264 := strings.EqualFold(videoCodec, "h264")
	ffmpegPath := getFFmpegPath()

	// 检查文件轨道完整性（无视频/无音频等异常情况）
	trackWarning := checkMediaTracks(filePath)
	if trackWarning != nil {
		transcodeProgress.Lock()
		transcodeProgress.trackWarning = trackWarning
		transcodeProgress.Unlock()
	}

	var success bool
	var usedGPU bool

	// 关键判断：如果源视频已经是H264，只需copy视频流+转码音频，GPU编码器不会被使用
	// 此时跳过GPU尝试，避免误导性日志和浪费时间
	if videoIsH264 {
		fmt.Printf("[转码] 视频已是H264，仅需转码音频 (视频copy)\n")
		success = executeTranscode(ffmpegPath, filePath, tempPath, videoCodec, videoIsH264, allAudioIsAAC, videoBitrate, audioBitrate, duration, "copy", "aac", false)
	} else if videoEncoder != "libx264" {
		// 非H264视频需要重新编码 → 尝试GPU全流水线(解码+编码)
		transcodeProgress.Lock()
		transcodeProgress.status = "running"
		transcodeProgress.progress = 0
		transcodeProgress.message = "尝试GPU加速编码(全流水线:解码+编码)..."
		transcodeProgress.Unlock()
		fmt.Printf("[转码] 尝试GPU全流水线(解码+编码): %s\n", videoEncoder)

		success = executeTranscode(ffmpegPath, filePath, tempPath, videoCodec, videoIsH264, allAudioIsAAC, videoBitrate, audioBitrate, duration, videoEncoder, audioEncoder, true)

		if !success {
			if checkTranscodeResult(tempPath, duration) {
				fmt.Printf("[GPU编码成功] 虽然返回错误码，但输出文件有效\n")
				success = true
				usedGPU = true
			}
		} else {
			usedGPU = true
		}

		// GPU全流水线失败 → 降级：CPU解码 + GPU编码
		if !success {
			transcodeProgress.Lock()
			transcodeProgress.status = "running"
			transcodeProgress.progress = 0
			transcodeProgress.message = "GPU全流水线失败，尝试CPU解码+GPU编码..."
			transcodeProgress.Unlock()
			fmt.Printf("[转码] 降级: CPU解码 + GPU编码(%s)\n", videoEncoder)

			success = executeTranscode(ffmpegPath, filePath, tempPath, videoCodec, videoIsH264, allAudioIsAAC, videoBitrate, audioBitrate, duration, videoEncoder, audioEncoder, false)

			if !success {
				if checkTranscodeResult(tempPath, duration) {
					fmt.Printf("[GPU编码成功] 虽然返回错误码，但输出文件有效\n")
					success = true
					usedGPU = true
				}
			} else {
				usedGPU = true
			}
		}

		if !success {
			transcodeProgress.Lock()
			transcodeProgress.status = "running"
			transcodeProgress.progress = 0
			transcodeProgress.message = "GPU编码失败，回退到CPU编码..."
			transcodeProgress.Unlock()
			fmt.Printf("[转码] 回退CPU编码 (libx264)\n")
			success = executeTranscode(ffmpegPath, filePath, tempPath, videoCodec, videoIsH264, allAudioIsAAC, videoBitrate, audioBitrate, duration, "libx264", "aac", false)
		}
	} else {
		success = executeTranscode(ffmpegPath, filePath, tempPath, videoCodec, videoIsH264, allAudioIsAAC, videoBitrate, audioBitrate, duration, "libx264", "aac", false)
	}

	if success && usedGPU {
		transcodeProgress.Lock()
		transcodeProgress.message = "GPU加速转码完成"
		transcodeProgress.Unlock()
	}

	if !success {
		fmt.Printf("[转码失败] 文件: %s\n", filePath)
		os.Remove(tempPath)
		return
	}

	fmt.Printf("[转码成功] 文件: %s\n", filePath)

	// 删除旧的MKV文件（如果存在）
	os.Remove(finalPath)

	if err := os.Rename(tempPath, finalPath); err != nil {
		if copyErr := copyFile(tempPath, finalPath); copyErr != nil {
			transcodeProgress.Lock()
			transcodeProgress.status = "completed"
			transcodeProgress.progress = 100
			transcodeProgress.message = "转码完成，但目标文件正在被使用，已保存为临时文件。请关闭播放器后手动替换: " + tempPath
			transcodeProgress.Unlock()
			return
		}
		os.Remove(tempPath)
	}

	// 如果输出文件名与原文件名不同，删除原文件
	if finalPath != filePath {
		os.Remove(filePath)
	}

	// 将输出路径转换为前端可用的相对路径格式（dirBase/relativePath）
	outputRelPath := finalPath
	for _, dir := range mediaDirs {
		if strings.HasPrefix(finalPath, dir+string(os.PathSeparator)) || strings.HasPrefix(finalPath, dir+"/") {
			rel, err := filepath.Rel(dir, finalPath)
			if err == nil {
				outputRelPath = filepath.Base(dir) + "/" + strings.ReplaceAll(rel, "\\", "/")
				break
			}
		}
	}

	transcodeProgress.Lock()
	transcodeProgress.status = "completed"
	transcodeProgress.progress = 100
	transcodeProgress.outputPath = outputRelPath
	transcodeProgress.Unlock()

	// 将转码产出文件路径加入内存映射表，删除原文件路径（原文件已被删除）
	// 注意：必须先删除旧路径，再添加新路径，避免 outputRelPath == requestKey 时自删
	newBasename := filepath.Base(finalPath)
	oldBasename := filepath.Base(requestKey)
	pathToAbsFile.Lock()
	basenameToAbsFile.Lock()
	// 1. 先删除旧路径
	delete(pathToAbsFile.m, requestKey)
	if oldPaths, ok := basenameToAbsFile.m[oldBasename]; ok {
		newPaths := make([]string, 0, len(oldPaths))
		for _, p := range oldPaths {
			if p != filePath {
				newPaths = append(newPaths, p)
			}
		}
		if len(newPaths) > 0 {
			basenameToAbsFile.m[oldBasename] = newPaths
		} else {
			delete(basenameToAbsFile.m, oldBasename)
		}
	}
	// 2. 再添加新路径
	pathToAbsFile.m[outputRelPath] = finalPath
	basenameToAbsFile.m[newBasename] = append(basenameToAbsFile.m[newBasename], finalPath)
	basenameToAbsFile.Unlock()
	pathToAbsFile.Unlock()
	log.Printf("[转码] 路径映射已更新: %s -> %s (移除旧路径: %s)", outputRelPath, filepath.Base(finalPath), requestKey)

	fmt.Printf("[转码完成] %s -> %s\n", filepath.Base(filePath), filepath.Base(finalPath))

	// 保存完成结果，供前端轮询查询
	lastCompletedTranscode.Lock()
	lastCompletedTranscode.requestKey = requestKey
	lastCompletedTranscode.outputPath = outputRelPath
	lastCompletedTranscode.status = "completed"
	lastCompletedTranscode.Unlock()
}

func executeTranscode(ffmpegPath, filePath, tempPath string, videoCodec string, videoIsH264, allAudioIsAAC bool, videoBitrate, audioBitrate string, duration float64, vEncoder, aEncoder string, useHWAccel bool) bool {
	mediaInfoStr := fmt.Sprintf("视频编码: %s (H264: %t)\n音频编码: 全部AAC: %t\n视频码率: %s\n音频码率: %s\n时长: %.2fs\n视频编码器: %s\n音频编码器: %s",
		videoCodec, videoIsH264, allAudioIsAAC, videoBitrate, audioBitrate, duration, vEncoder, aEncoder)

	var cmd *exec.Cmd
	var cmdStr string
	if videoIsH264 {
		if allAudioIsAAC {
			transcodeProgress.Lock()
			transcodeProgress.status = "completed"
			transcodeProgress.progress = 100
			transcodeProgress.message = "文件已满足H264+AAC格式，无需转码"
			transcodeProgress.mediaInfo = mediaInfoStr
			transcodeProgress.Unlock()
			return true
		} else {
			cmd = exec.Command(ffmpegPath, "-i", filePath,
				"-c:v", "copy",
				"-c:a", aEncoder, "-b:a", audioBitrate,
				"-map", "0:v", "-map", "0:a",
				"-progress", "pipe:1",
				"-y", tempPath)
			cmdStr = ffmpegPath + " -i \"" + filePath + "\" -c:v copy -c:a " + aEncoder + " -b:a " + audioBitrate + " -map 0:v -map 0:a -progress pipe:1 -y \"" + tempPath + "\""
		}
	} else {
		if strings.Contains(vEncoder, "nvenc") {
			// NVENC 编码：useHWAccel=true 表示尝试GPU全流水线（解码+编码）
			// -hwaccel cuda 让FFmpeg自动选择CUVID解码器（h264_cuvid/hevc_cuvid/vp9_cuvid等）
			// 数据全程留在GPU显存，最大化GPU利用率
			if useHWAccel {
				if allAudioIsAAC {
					cmd = exec.Command(ffmpegPath, "-hwaccel", "cuda", "-i", filePath,
						"-c:v", "h264_nvenc", "-preset", "p4", "-cq", "18", "-profile:v", "high",
						"-c:a", "copy",
						"-map", "0:v", "-map", "0:a",
						"-progress", "pipe:1",
						"-y", tempPath)
					cmdStr = ffmpegPath + " -hwaccel cuda -i \"" + filePath + "\" -c:v h264_nvenc -preset p4 -cq 18 -profile:v high -c:a copy -map 0:v -map 0:a -progress pipe:1 -y \"" + tempPath + "\""
				} else {
					cmd = exec.Command(ffmpegPath, "-hwaccel", "cuda", "-i", filePath,
						"-c:v", "h264_nvenc", "-preset", "p4", "-cq", "18", "-profile:v", "high",
						"-c:a", aEncoder, "-b:a", audioBitrate,
						"-map", "0:v", "-map", "0:a",
						"-progress", "pipe:1",
						"-y", tempPath)
					cmdStr = ffmpegPath + " -hwaccel cuda -i \"" + filePath + "\" -c:v h264_nvenc -preset p4 -cq 18 -profile:v high -c:a " + aEncoder + " -b:a " + audioBitrate + " -map 0:v -map 0:a -progress pipe:1 -y \"" + tempPath + "\""
				}
			} else {
				// 降级模式：CPU解码 + NVENC编码（数据需CPU→GPU搬运，GPU利用率较低）
				if allAudioIsAAC {
					cmd = exec.Command(ffmpegPath, "-i", filePath,
						"-c:v", "h264_nvenc", "-preset", "p4", "-cq", "18", "-profile:v", "high",
						"-c:a", "copy",
						"-map", "0:v", "-map", "0:a",
						"-progress", "pipe:1",
						"-y", tempPath)
					cmdStr = ffmpegPath + " -i \"" + filePath + "\" -c:v h264_nvenc -preset p4 -cq 18 -profile:v high -c:a copy -map 0:v -map 0:a -progress pipe:1 -y \"" + tempPath + "\""
				} else {
					cmd = exec.Command(ffmpegPath, "-i", filePath, "-c:v", "h264_nvenc", "-preset", "p4", "-cq", "18", "-profile:v", "high", "-c:a", aEncoder, "-b:a", audioBitrate, "-map", "0:v", "-map", "0:a", "-progress", "pipe:1", "-y", tempPath)
					cmdStr = ffmpegPath + " -i \"" + filePath + "\" -c:v h264_nvenc -preset p4 -cq 18 -profile:v high -c:a " + aEncoder + " -b:a " + audioBitrate + " -map 0:v -map 0:a -progress pipe:1 -y \"" + tempPath + "\""
				}
			}
		} else if strings.Contains(vEncoder, "qsv") {
			if allAudioIsAAC {
				cmd = exec.Command(ffmpegPath, "-i", filePath, "-c:v", vEncoder, "-b:v", videoBitrate, "-c:a", "copy", "-map", "0:v", "-map", "0:a", "-progress", "pipe:1", "-y", tempPath)
				cmdStr = ffmpegPath + " -i \"" + filePath + "\" -c:v " + vEncoder + " -b:v " + videoBitrate + " -c:a copy -map 0:v -map 0:a -progress pipe:1 -y \"" + tempPath + "\""
			} else {
				cmd = exec.Command(ffmpegPath, "-i", filePath, "-c:v", vEncoder, "-b:v", videoBitrate, "-c:a", aEncoder, "-b:a", audioBitrate, "-map", "0:v", "-map", "0:a", "-progress", "pipe:1", "-y", tempPath)
				cmdStr = ffmpegPath + " -i \"" + filePath + "\" -c:v " + vEncoder + " -b:v " + videoBitrate + " -c:a " + aEncoder + " -b:a " + audioBitrate + " -map 0:v -map 0:a -progress pipe:1 -y \"" + tempPath + "\""
			}
		} else {
			if allAudioIsAAC {
				cmd = exec.Command(ffmpegPath, "-i", filePath, "-c:v", vEncoder, "-b:v", videoBitrate, "-c:a", "copy", "-map", "0:v", "-map", "0:a", "-progress", "pipe:1", "-y", tempPath)
				cmdStr = ffmpegPath + " -i \"" + filePath + "\" -c:v " + vEncoder + " -b:v " + videoBitrate + " -c:a copy -map 0:v -map 0:a -progress pipe:1 -y \"" + tempPath + "\""
			} else {
				cmd = exec.Command(ffmpegPath, "-i", filePath, "-c:v", vEncoder, "-b:v", videoBitrate, "-c:a", aEncoder, "-b:a", audioBitrate, "-map", "0:v", "-map", "0:a", "-progress", "pipe:1", "-y", tempPath)
				cmdStr = ffmpegPath + " -i \"" + filePath + "\" -c:v " + vEncoder + " -b:v " + videoBitrate + " -c:a " + aEncoder + " -b:a " + audioBitrate + " -map 0:v -map 0:a -progress pipe:1 -y \"" + tempPath + "\""
			}
		}
	}

	transcodeProgress.Lock()
	transcodeProgress.command = cmdStr
	transcodeProgress.mediaInfo = mediaInfoStr
	transcodeProgress.Unlock()

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		transcodeProgress.Lock()
		transcodeProgress.status = "error"
		transcodeProgress.message = "创建输出管道失败: " + err.Error()
		transcodeProgress.Unlock()
		return false
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		transcodeProgress.Lock()
		transcodeProgress.status = "error"
		transcodeProgress.message = "创建错误管道失败: " + err.Error()
		transcodeProgress.Unlock()
		return false
	}

	if err := cmd.Start(); err != nil {
		fmt.Printf("[转码错误] 启动失败 - 编码器: %s, 文件: %s, 错误: %v\n", vEncoder, filePath, err)
		transcodeProgress.Lock()
		transcodeProgress.status = "error"
		transcodeProgress.message = "启动转码失败: " + err.Error()
		transcodeProgress.Unlock()
		return false
	}

	go parseFFmpegOutput(stdout, duration)

	var stderrBuf strings.Builder
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			stderrBuf.WriteString(line + "\n")
		}
	}()

	if err := cmd.Wait(); err != nil {
		errorMsg := stderrBuf.String()
		fmt.Printf("[转码错误] 编码器: %s, 文件: %s, 错误: %v\n", vEncoder, filePath, err)
		transcodeProgress.Lock()
		transcodeProgress.status = "error"
		transcodeProgress.message = "转码执行失败: " + err.Error()
		if errorMsg != "" {
			transcodeProgress.log = errorMsg
		}
		transcodeProgress.Unlock()
		return false
	}

	return true
}

func checkTranscodeResult(tempPath string, expectedDuration float64) bool {
	info, err := os.Stat(tempPath)
	if err != nil {
		return false
	}

	if info.Size() < 1024 {
		return false
	}

	if expectedDuration <= 0 {
		return info.Size() > 100*1024
	}

	ffprobePath := getFFprobePath()
	durationCmd := exec.Command(ffprobePath, "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", tempPath)
	durationOutput, err := durationCmd.CombinedOutput()
	if err != nil {
		return info.Size() > 100*1024
	}

	durationStr := strings.TrimSpace(string(durationOutput))
	if durationStr == "" || durationStr == "N/A" {
		return info.Size() > 100*1024
	}

	duration, err := strconv.ParseFloat(durationStr, 64)
	if err != nil {
		return info.Size() > 100*1024
	}

	return duration >= expectedDuration*0.9
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

func getFFprobePath() string {
	ffmpegPath := getFFmpegPath()
	if ffmpegPath != "ffmpeg" {
		ffprobePath := strings.Replace(ffmpegPath, "ffmpeg.exe", "ffprobe.exe", -1)
		if _, err := os.Stat(ffprobePath); err == nil {
			return ffprobePath
		}
	}

	if path, err := exec.LookPath("ffprobe"); err == nil {
		return path
	}

	return "ffprobe"
}

func getDuration(filePath string) float64 {
	ffprobePath := getFFprobePath()

	durationCmd := exec.Command(ffprobePath, "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", filePath)
	durationOutput, err := durationCmd.CombinedOutput()
	if err != nil {
		fmt.Printf("[转码] 获取时长失败: %v\n", err)
	}

	durationStr := strings.TrimSpace(string(durationOutput))

	if durationStr == "" || durationStr == "N/A" || durationStr == "NA" {
		durationCmd2 := exec.Command(ffprobePath, "-v", "error", "-select_streams", "v:0", "-show_entries", "stream=duration", "-of", "default=noprint_wrappers=1:nokey=1", filePath)
		durationOutput, err = durationCmd2.CombinedOutput()
		if err == nil {
			durationStr = strings.TrimSpace(string(durationOutput))
		}
	}

	if durationStr == "" || durationStr == "N/A" || durationStr == "NA" {
		framesCmd := exec.Command(ffprobePath, "-v", "error", "-select_streams", "v:0", "-show_entries", "stream=nb_frames", "-of", "default=noprint_wrappers=1:nokey=1", filePath)
		framesOutput, err := framesCmd.CombinedOutput()
		if err == nil {
			framesStr := strings.TrimSpace(string(framesOutput))
			if framesStr != "" && framesStr != "N/A" {
				frames, _ := strconv.Atoi(framesStr)
				fpsCmd := exec.Command(ffprobePath, "-v", "error", "-select_streams", "v:0", "-show_entries", "stream=r_frame_rate", "-of", "default=noprint_wrappers=1:nokey=1", filePath)
				fpsOutput, err := fpsCmd.CombinedOutput()
				if err == nil {
					fpsStr := strings.TrimSpace(string(fpsOutput))
					if strings.Contains(fpsStr, "/") {
						parts := strings.Split(fpsStr, "/")
						if len(parts) == 2 {
							num, _ := strconv.Atoi(parts[0])
							den, _ := strconv.Atoi(parts[1])
							if den > 0 {
								fps := float64(num) / float64(den)
								return float64(frames) / fps
							}
						}
					} else {
						fps, _ := strconv.ParseFloat(fpsStr, 64)
						if fps > 0 {
							return float64(frames) / fps
						}
					}
				}
			}
		}
	}

	duration, err := strconv.ParseFloat(durationStr, 64)
	if err != nil {
		fmt.Printf("[转码] 时长解析失败: %v\n", err)
		return 0
	}
	return duration
}

func getMediaInfo(filePath string) (videoCodec string, allAudioIsAAC bool, allAudioIsMP3 bool, allAudioBrowserSupported bool, videoBitrate, audioBitrate string, audioCodecStr string) {
	totalStart := time.Now()

	// 阶段1：os.Stat（检测磁盘休眠唤醒——如果休眠，这里会卡很久）
	statStart := time.Now()
	fileInfo, statErr := os.Stat(filePath)
	statMs := time.Since(statStart).Milliseconds()
	if statMs > 100 {
		log.Printf("[getMediaInfo] 阶段1 os.Stat慢 耗时=%dms file=%s (疑似磁盘休眠唤醒)", statMs, filepath.Base(filePath))
	}

	// 阶段2：查缓存
	if statErr == nil {
		modTime := fileInfo.ModTime()
		cacheStart := time.Now()
		mediaInfoCache.RLock()
		if entry, ok := mediaInfoCache.cache[filePath]; ok && entry.modTime.Equal(modTime) {
			result := entry
			mediaInfoCache.RUnlock()
			log.Printf("[getMediaInfo] 缓存命中 总耗时=%dms file=%s", time.Since(totalStart).Milliseconds(), filepath.Base(filePath))
			return result.videoCodec, result.allAudioIsAAC, result.allAudioIsMP3, result.allAudioBrowserSupported, result.videoBitrate, result.audioBitrate, result.audioCodecStr
		}
		mediaInfoCache.RUnlock()
		log.Printf("[getMediaInfo] 缓存查询耗时=%dms file=%s", time.Since(cacheStart).Milliseconds(), filepath.Base(filePath))
	}

	// 阶段3：文件预热——主动读一小块数据，用来检测磁盘是否休眠
	// （如果文件所在磁盘处于休眠状态，这个读取会等待磁盘唤醒，时间会显示在日志中）
	// 同时启动一个 goroutine 监控：如果 500ms 内 Read 没返回，则认为磁盘休眠，
	// 通过全局状态告知前端显示"硬盘已休眠，正在等待硬盘唤醒响应..."
	preheatStart := time.Now()
	wakingFlag := int32(0) // 1 = 已经通知前端进入休眠等待
	done := make(chan struct{})
	go func() {
		select {
		case <-time.After(500 * time.Millisecond):
			// 500ms 还没完成，认为是磁盘休眠
			atomic.StoreInt32(&wakingFlag, 1)
			setDiskSleeping(true, filePath)
		case <-done:
			// Read 已返回，无需通知
		}
	}()
	if f, err := os.Open(filePath); err == nil {
		buf := make([]byte, 4096)
		f.Read(buf)
		f.Close()
	}
	close(done)
	preheatMs := time.Since(preheatStart).Milliseconds()
	if atomic.LoadInt32(&wakingFlag) == 1 {
		// 唤醒完成，清除前端提示
		setDiskSleeping(false, filePath)
	}
	if preheatMs > 100 {
		log.Printf("[getMediaInfo] 阶段3 文件预热慢 耗时=%dms file=%s (磁盘休眠唤醒)", preheatMs, filepath.Base(filePath))
	}

	// 阶段4：启动 ffprobe 子进程
	startTime := time.Now()
	ffprobePath := getFFprobePath()
	log.Printf("[getMediaInfo] 缓存未命中，调用ffprobe: file=%s", filepath.Base(filePath))

	// 一次 ffprobe 调用拿到所有信息（视频/音频 codec、码率、声道数、格式码率）
	// 替代原先的 4 次串行调用
	cmd := exec.Command(ffprobePath, "-v", "error",
		"-show_entries", "stream=codec_type,codec_name,bit_rate,channels:format=bit_rate",
		"-of", "json", filePath)
	output, cmdErr := cmd.CombinedOutput()
	ffprobeMs := time.Since(startTime).Milliseconds()
	log.Printf("[getMediaInfo] 阶段4 ffprobe完成 耗时=%dms 输出长度=%d 错误=%v file=%s",
		ffprobeMs, len(output), cmdErr, filepath.Base(filePath))

	type probeStream struct {
		CodecType string `json:"codec_type"`
		CodecName string `json:"codec_name"`
		BitRate   string `json:"bit_rate"`
		Channels  int    `json:"channels,omitempty"`
	}
	type probeFormat struct {
		BitRate string `json:"bit_rate"`
	}
	type probeResult struct {
		Streams []probeStream `json:"streams"`
		Format  probeFormat   `json:"format"`
	}

	var pr probeResult
	if cmdErr == nil {
		json.Unmarshal(output, &pr)
	}

	var audioCodecNames []string
	var streamVideoBitrate string // 视频流码率
	var firstAudioBitrate string  // 第一条音频流码率
	allAudioIsAAC = true
	allAudioIsMP3 = true
	allAudioBrowserSupported = true

	for _, s := range pr.Streams {
		switch strings.ToLower(s.CodecType) {
		case "video":
			if videoCodec == "" {
				videoCodec = strings.TrimSpace(s.CodecName)
				streamVideoBitrate = strings.TrimSpace(s.BitRate)
			}
		case "audio":
			codec := strings.TrimSpace(s.CodecName)
			if codec != "" {
				audioCodecNames = append(audioCodecNames, strings.ToUpper(codec))
				codecLower := strings.ToLower(codec)
				if codecLower != "aac" {
					allAudioIsAAC = false
				}
				if codecLower != "mp3" {
					allAudioIsMP3 = false
				}
				if codecLower != "aac" && codecLower != "mp3" {
					allAudioBrowserSupported = false
				}
			}
			// 记录第一条音频流码率
			if firstAudioBitrate == "" {
				firstAudioBitrate = strings.TrimSpace(s.BitRate)
			}
		}
	}
	audioCodecStr = strings.Join(audioCodecNames, ",")

	// 视频码率：优先用 stream.bit_rate，失败则 fallback 到 format.bit_rate（一次 ffprobe 已拿到，无需再调一次）
	bitrateStr := streamVideoBitrate
	if bitrateStr == "" || bitrateStr == "N/A" {
		bitrateStr = strings.TrimSpace(pr.Format.BitRate)
	}
	if bitrateStr == "" || bitrateStr == "N/A" {
		videoBitrate = "1000k"
	} else if bitrateInt, err := strconv.Atoi(bitrateStr); err != nil || bitrateInt <= 0 {
		videoBitrate = "1000k"
	} else {
		videoBitrate = fmt.Sprintf("%dk", bitrateInt/1000)
	}

	// 音频码率
	if firstAudioBitrate == "" || firstAudioBitrate == "0" {
		audioBitrate = "192k"
	} else {
		bitrate, _ := strconv.Atoi(firstAudioBitrate)
		if bitrate <= 0 {
			audioBitrate = "192k"
		} else {
			audioBitrate = fmt.Sprintf("%dk", bitrate/1000)
		}
	}

	// 存入缓存
	if statErr == nil {
		mediaInfoCache.Lock()
		// 缓存上限检测，超过时清空一半
		if len(mediaInfoCache.cache) >= mediaInfoCacheMaxEntries {
			count := 0
			for key := range mediaInfoCache.cache {
				delete(mediaInfoCache.cache, key)
				count++
				if count >= mediaInfoCacheMaxEntries/2 {
					break
				}
			}
		}
		mediaInfoCache.cache[filePath] = &mediaInfoCacheEntry{
			videoCodec:               videoCodec,
			allAudioIsAAC:            allAudioIsAAC,
			allAudioIsMP3:            allAudioIsMP3,
			allAudioBrowserSupported: allAudioBrowserSupported,
			videoBitrate:             videoBitrate,
			audioBitrate:             audioBitrate,
			audioCodecStr:            audioCodecStr,
			modTime:                  fileInfo.ModTime(),
		}
		mediaInfoCache.Unlock()
	}

	log.Printf("[getMediaInfo] ffprobe完成: file=%s 总耗时=%dms (stat=%dms preheat=%dms ffprobe=%dms) vCodec=%s aCodec=%s",
		filepath.Base(filePath), time.Since(totalStart).Milliseconds(), statMs, preheatMs, ffprobeMs, videoCodec, audioCodecStr)

	return
}

// MediaTrackWarning 媒体轨道问题警告
type MediaTrackWarning struct {
	NoVideo         bool   `json:"noVideo"`         // 视频文件无视频轨
	NoAudio         bool   `json:"noAudio"`         // 无音频轨
	AudioTrackCount int    `json:"audioTrackCount"` // 音频轨道数
	AudioChannels   int    `json:"audioChannels"`   // 第一条音频轨的声道数（1=单声道, 2=立体声）
	Message         string `json:"message"`         // 人类可读的警告信息
}

// mediaTracksCache checkMediaTracks 的结果缓存，避免同一文件多次调用 ffprobe
var mediaTracksCache struct {
	sync.RWMutex
	cache map[string]*mediaTracksCacheEntry
}

type mediaTracksCacheEntry struct {
	warning *MediaTrackWarning
	modTime time.Time
}

// checkMediaTracks 检查文件是否缺少视频轨或音频轨
// 对于视频容器（mkv/mp4/mpg/avi等）中缺少视频或音频的情况发出警告
func checkMediaTracks(filePath string) *MediaTrackWarning {
	// 查缓存：先获取文件 ModTime，命中且 ModTime 一致则直接返回
	fileInfo, statErr := os.Stat(filePath)
	if statErr == nil {
		modTime := fileInfo.ModTime()
		mediaTracksCache.RLock()
		if entry, ok := mediaTracksCache.cache[filePath]; ok && entry.modTime.Equal(modTime) {
			w := entry.warning
			mediaTracksCache.RUnlock()
			return w
		}
		mediaTracksCache.RUnlock()
	}

	// 一次 ffprobe 调用拿到视频/音频轨存在性、音频轨数、声道数
	// 替代原先的 3 次串行调用
	ffprobePath := getFFprobePath()
	ext := strings.ToLower(filepath.Ext(filePath))
	videoExts := map[string]bool{
		".mkv": true, ".mp4": true, ".mpg": true, ".mpeg": true,
		".avi": true, ".wmv": true, ".rm": true, ".rmvb": true,
		".flv": true, ".ts": true, ".mov": true, ".webm": true,
		".3gp": true, ".vob": true,
	}
	isVideoContainer := videoExts[ext]

	startTime := time.Now()
	cmd := exec.Command(ffprobePath, "-v", "error",
		"-show_entries", "stream=codec_type,channels",
		"-of", "json", filePath)
	output, _ := cmd.CombinedOutput()
	log.Printf("[checkMediaTracks] ffprobe完成: file=%s 耗时=%dms 输出长度=%d",
		filepath.Base(filePath), time.Since(startTime).Milliseconds(), len(output))

	type probeStream struct {
		CodecType string `json:"codec_type"`
		Channels  int    `json:"channels,omitempty"`
	}
	type probeResult struct {
		Streams []probeStream `json:"streams"`
	}
	var pr probeResult
	json.Unmarshal(output, &pr)

	hasVideo := false
	hasAudio := false
	audioTrackCount := 0
	audioChannels := 0
	for _, s := range pr.Streams {
		switch strings.ToLower(s.CodecType) {
		case "video":
			hasVideo = true
		case "audio":
			audioTrackCount++
			if audioTrackCount == 1 {
				audioChannels = s.Channels
			}
		}
	}
	if audioTrackCount > 0 {
		hasAudio = true
	}

	var warnings []string
	w := &MediaTrackWarning{AudioTrackCount: audioTrackCount, AudioChannels: audioChannels}

	if isVideoContainer && !hasVideo {
		w.NoVideo = true
		warnings = append(warnings, "无视频轨（该文件虽然有视频容器格式，但没有视频画面）")
	}
	if !hasAudio {
		w.NoAudio = true
		warnings = append(warnings, "无音频轨（该文件没有声音）")
	}
	if isVideoContainer && hasAudio && audioTrackCount == 1 {
		warnings = append(warnings, "仅有1条音轨（无法切换原唱/伴奏）")
	}

	if len(warnings) > 0 {
		w.Message = "曲库文件异常: " + strings.Join(warnings, "，") + "。这不是系统问题，是源文件本身的问题。"
		fmt.Printf("[警告] %s: %s\n", filepath.Base(filePath), w.Message)
	} else {
		log.Printf("[checkMediaTracks] ffprobe完成: file=%s audioTrackCount=%d channels=%d hasVideo=%v",
			filepath.Base(filePath), audioTrackCount, audioChannels, hasVideo)
	}

	// 存入缓存
	if statErr == nil {
		mediaTracksCache.Lock()
		if len(mediaTracksCache.cache) >= mediaInfoCacheMaxEntries {
			count := 0
			for key := range mediaTracksCache.cache {
				delete(mediaTracksCache.cache, key)
				count++
				if count >= mediaInfoCacheMaxEntries/2 {
					break
				}
			}
		}
		mediaTracksCache.cache[filePath] = &mediaTracksCacheEntry{
			warning: w,
			modTime: fileInfo.ModTime(),
		}
		mediaTracksCache.Unlock()
	}

	return w
}

func isVideoCodecSupported(codec string) bool {
	supported := []string{"h264", "vp8", "vp9", "av1", "mpeg4", "wmv", "flv"}
	codec = strings.ToLower(codec)
	for _, s := range supported {
		if strings.Contains(codec, s) {
			return true
		}
	}
	return false
}

func isAudioCodecSupported(codec string) bool {
	supported := []string{"aac", "mp3", "opus", "vorbis", "pcm_s16le", "flac", "wma"}
	codec = strings.ToLower(codec)
	for _, s := range supported {
		if strings.Contains(codec, s) {
			return true
		}
	}
	return false
}

func getVideoEncoder() string {
	ffmpegPath := getFFmpegPath()

	gpuEncoders := []string{"h264_nvenc", "h264_amf", "h264_qsv"}

	// 先获取编码器列表
	encodersOutput, err := exec.Command(ffmpegPath, "-encoders").CombinedOutput()
	if err != nil {
		return "libx264"
	}
	encodersList := strings.ToLower(string(encodersOutput))

	for _, encoder := range gpuEncoders {
		// 检查编码器是否编译在内
		if !strings.Contains(encodersList, encoder) {
			continue
		}
		// 实际可用性测试：编码1帧64x64黑帧，验证GPU驱动是否支持
		testCmd := exec.Command(ffmpegPath,
			"-f", "lavfi", "-i", "color=black:size=64x64:duration=0.1:rate=1",
			"-c:v", encoder, "-f", "null", "-")
		testOutput, testErr := testCmd.CombinedOutput()
		if testErr != nil {
			// 从stderr中提取关键错误信息
			errLines := []string{}
			for _, line := range strings.Split(string(testOutput), "\n") {
				line = strings.TrimSpace(line)
				if line != "" && (strings.Contains(line, "Error") || strings.Contains(line, "error") ||
					strings.Contains(line, "fail") || strings.Contains(line, "not support") ||
					strings.Contains(line, "minimum") || strings.Contains(line, "required")) {
					errLines = append(errLines, line)
				}
			}
			if len(errLines) > 0 {
				fmt.Printf("[GPU] %s 不可用: %s\n", encoder, errLines[0])
			} else {
				fmt.Printf("[GPU] %s 不可用\n", encoder)
			}
			continue
		}
		return encoder
	}

	return "libx264"
}

func parseFFmpegOutput(reader io.ReadCloser, duration float64) {
	defer reader.Close()

	scanner := bufio.NewScanner(reader)
	var currentTime float64 = 0
	var logBuffer strings.Builder

	for scanner.Scan() {
		line := scanner.Text()

		logBuffer.WriteString(line + "\n")
		if logBuffer.Len() > 4000 {
			logStr := logBuffer.String()
			if len(logStr) > 2000 {
				logStr = logStr[len(logStr)-2000:]
			}
			logBuffer.Reset()
			logBuffer.WriteString(logStr)
		}

		transcodeProgress.Lock()
		transcodeProgress.log = logBuffer.String()
		transcodeProgress.Unlock()

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			switch key {
			case "out_time_ms":
				us, _ := strconv.ParseFloat(value, 64)
				currentTime = us / 1000000

				if duration > 0 {
					progress := int((currentTime / duration) * 100)
					if progress > 99 {
						progress = 99
					}

					transcodeProgress.Lock()
					if progress > transcodeProgress.progress {
						transcodeProgress.progress = progress
					}
					transcodeProgress.Unlock()
				}
			case "out_time":
				if duration > 0 {
					timeParts := strings.Split(value, ":")
					if len(timeParts) == 3 {
						hours, _ := strconv.Atoi(timeParts[0])
						minutes, _ := strconv.Atoi(timeParts[1])
						seconds, _ := strconv.ParseFloat(timeParts[2], 64)
						currentTime = float64(hours)*3600 + float64(minutes)*60 + seconds

						progress := int((currentTime / duration) * 100)
						if progress > 99 {
							progress = 99
						}

						transcodeProgress.Lock()
						if progress > transcodeProgress.progress {
							transcodeProgress.progress = progress
						}
						transcodeProgress.Unlock()
					}
				}
			}
		}
	}
}

type TranscodeStatus struct {
	NeedsTranscode bool   `json:"needsTranscode"`
	VideoCodec     string `json:"videoCodec"`
	AudioCodec     string `json:"audioCodec"`
	Message        string `json:"message"`
}

func CheckTranscodeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	fileName := r.URL.Query().Get("fileName")
	if fileName == "" {
		http.Error(w, "缺少fileName参数", 400)
		return
	}

	foundPath := findMediaFile(fileName)
	if foundPath == "" {
		http.Error(w, "文件未找到", 404)
		return
	}

	videoCodec, _, _, allAudioBrowserSupported, _, _, _ := getMediaInfo(foundPath)
	videoIsH264 := strings.EqualFold(videoCodec, "h264")

	result := TranscodeStatus{
		VideoCodec: videoCodec,
		AudioCodec: "AAC",
	}

	if videoIsH264 && allAudioBrowserSupported {
		result.NeedsTranscode = false
		result.Message = "文件已满足H264+AAC/MP3格式，无需转码"
	} else {
		result.NeedsTranscode = true
		result.AudioCodec = "非全部AAC/MP3"
		if !videoIsH264 {
			result.Message = "视频编码: " + videoCodec + "，需要转码为H264"
		} else {
			result.Message = "音频编码需要转换"
		}
	}

	json.NewEncoder(w).Encode(result)
}

// TrackSwitchHandler 处理手机端音轨切换请求
// 通过FFmpeg重组文件，选择指定音轨输出
func TrackSwitchHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	if r.Method != http.MethodPost {
		http.Error(w, "不支持的方法", 405)
		return
	}

	var req struct {
		FileName   string `json:"fileName"`
		TrackIndex int    `json:"trackIndex"` // 0=原唱, 1=伴奏
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "解析请求失败", 400)
		return
	}

	if req.FileName == "" {
		http.Error(w, "缺少fileName参数", 400)
		return
	}

	foundPath := findMediaFile(req.FileName)
	if foundPath == "" {
		http.Error(w, "文件未找到", 404)
		return
	}

	// 检查是否已有缓存文件
	dir := filepath.Dir(foundPath)
	ext := filepath.Ext(foundPath)
	baseName := strings.TrimSuffix(filepath.Base(foundPath), ext)

	// 如果原文件是转码后的mkv，也要处理
	unsupportedContainers := map[string]bool{".mpg": true, ".mpeg": true, ".avi": true, ".wmv": true, ".rm": true, ".rmvb": true, ".flv": true, ".ts": true}
	if unsupportedContainers[strings.ToLower(ext)] {
		ext = ".mkv"
	}

	trackSuffix := "_track0"
	if req.TrackIndex == 1 {
		trackSuffix = "_track1"
	}
	cachedPath := filepath.Join(dir, baseName+trackSuffix+ext)

	// 检查缓存文件是否存在且有效
	if info, err := os.Stat(cachedPath); err == nil && info.Size() > 1024 {
		// 转换为前端可用的相对路径
		outputRelPath := cachedPath
		for _, mdir := range mediaDirs {
			if strings.HasPrefix(cachedPath, mdir+string(os.PathSeparator)) || strings.HasPrefix(cachedPath, mdir+"/") {
				rel, err := filepath.Rel(mdir, cachedPath)
				if err == nil {
					outputRelPath = filepath.Base(mdir) + "/" + strings.ReplaceAll(rel, "\\", "/")
					break
				}
			}
		}
		// 命中缓存，无需额外日志
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":    true,
			"outputPath": outputRelPath,
			"cached":     true,
		})
		return
	}

	// 需要用FFmpeg重组
	videoCodec, _, _, allAudioBrowserSupported, videoBitrate, audioBitrate, _ := getMediaInfo(foundPath)
	videoIsH264 := strings.EqualFold(videoCodec, "h264")
	audioSupported := allAudioBrowserSupported

	ffmpegPath := getFFmpegPath()
	tempPath := filepath.Join(dir, baseName+trackSuffix+"_tmp"+ext)

	var cmd *exec.Cmd

	if videoIsH264 && audioSupported {
		// 视频和音频都兼容浏览器，只需重组：复制视频流+选择指定音轨
		cmd = exec.Command(ffmpegPath, "-i", foundPath,
			"-map", "0:v:0", "-map", "0:a:"+strconv.Itoa(req.TrackIndex),
			"-c", "copy",
			"-y", tempPath)
	} else if videoIsH264 && !audioSupported {
		// 视频兼容，音频需要转码
		cmd = exec.Command(ffmpegPath, "-i", foundPath,
			"-map", "0:v:0", "-map", "0:a:"+strconv.Itoa(req.TrackIndex),
			"-c:v", "copy", "-c:a", "aac", "-b:a", audioBitrate,
			"-y", tempPath)
	} else {
		// 视频也需要转码，使用配置的编码器
		vEnc := videoEncoder
		aEnc := audioEncoder
		if vEnc == "" {
			vEnc = "libx264"
		}
		if aEnc == "" {
			aEnc = "aac"
		}

		if audioSupported {
			cmd = exec.Command(ffmpegPath, "-i", foundPath,
				"-map", "0:v:0", "-map", "0:a:"+strconv.Itoa(req.TrackIndex),
				"-c:v", vEnc, "-b:v", videoBitrate, "-c:a", "copy",
				"-y", tempPath)
		} else {
			cmd = exec.Command(ffmpegPath, "-i", foundPath,
				"-map", "0:v:0", "-map", "0:a:"+strconv.Itoa(req.TrackIndex),
				"-c:v", vEnc, "-b:v", videoBitrate, "-c:a", aEnc, "-b:a", audioBitrate,
				"-y", tempPath)
		}
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("[音轨切换] FFmpeg错误: %v, 输出: %s\n", err, string(output))
		os.Remove(tempPath)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "音轨切换失败: " + err.Error(),
		})
		return
	}

	// 检查输出文件
	if info, err := os.Stat(tempPath); err != nil || info.Size() < 1024 {
		fmt.Printf("[音轨切换] 输出文件无效\n")
		os.Remove(tempPath)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "音轨切换失败: 输出文件无效",
		})
		return
	}

	// 重命名为缓存文件
	os.Rename(tempPath, cachedPath)

	// 转换为前端可用的相对路径
	outputRelPath := cachedPath
	for _, mdir := range mediaDirs {
		if strings.HasPrefix(cachedPath, mdir+string(os.PathSeparator)) || strings.HasPrefix(cachedPath, mdir+"/") {
			rel, err := filepath.Rel(mdir, cachedPath)
			if err == nil {
				outputRelPath = filepath.Base(mdir) + "/" + strings.ReplaceAll(rel, "\\", "/")
				break
			}
		}
	}

	fmt.Printf("[音轨切换] 完成: %s (音轨%d)\n", filepath.Base(foundPath), req.TrackIndex)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":    true,
		"outputPath": outputRelPath,
		"cached":     false,
	})
}
