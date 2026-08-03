package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// 使用全局变量mediaDir，由配置文件或GUI设置

type MediaFile struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName,omitempty"` // 同名歌曲时带大小标注，仅供前端显示
	Path        string `json:"path"`
	Dir         string `json:"dir"`                    // 所属曲库目录的绝对路径
	Type        string `json:"type"`                 // video / audio
	Singer      string `json:"singer"`
	Language    string `json:"language"`    // 语种（从文件名解析）
	Category    string `json:"category"`    // 曲种（从文件名解析）
	Size        int64  `json:"size"`        // 文件大小（字节）
	PinyinInit  string `json:"-"`           // 预计算的拼音首字母，仅用于搜索
}

// parseFilenameFields 按规则 歌手-歌名-语种-歌曲类别 解析文件名
func parseFilenameFields(name string) (singer, language, category string) {
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	parts := strings.SplitN(base, "-", 4)
	if len(parts) >= 1 {
		singer = strings.TrimSpace(parts[0])
	}
	if len(parts) >= 3 {
		language = strings.TrimSpace(parts[2])
	}
	if len(parts) >= 4 {
		category = strings.TrimSpace(parts[3])
	}
	if singer == "" {
		singer = "未知歌手"
	}
	return
}

// 扫描进度回调函数类型
type ScanProgressCallback func(current, total int, fileName string)

// 扫描媒体文件（带进度回调）
func getMediaListWithProgress(callback ScanProgressCallback) []MediaFile {
	var list []MediaFile
	var totalFiles int
	var currentCount int

	// 支持的格式
	exts := map[string]bool{
		"mp3":  true,
		"flac": true,
		"wav":  true,
		"mp4":  true,
		"webm": true,
		"mkv":  true,
		"mpg":  true,
		"mpeg": true,
		"avi":  true,
		"mov":  true,
		"wmv":  true,
		"rm":   true,
		"rmvb": true,
		"ts":   true,
		"flv":  true,
		"aac":  true,
		"m4a":  true,
		"ogg":  true,
		"wma":  true,
		"ape":  true,
	}

	// 先统计所有媒体目录中支持格式的文件总数
	for _, dir := range mediaDirs {
		filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if !info.IsDir() {
				ext := strings.ToLower(filepath.Ext(info.Name()))
				if len(ext) > 1 {
					ext = ext[1:]
				}
				if exts[ext] {
					totalFiles++
				}
			}
			return nil
		})
	}

	// 递归扫描所有媒体目录
	for _, dir := range mediaDirs {
		filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			ext := strings.ToLower(filepath.Ext(info.Name()))
			if len(ext) > 1 {
				ext = ext[1:]
			}
			if !exts[ext] {
				return nil
			}

			currentCount++

			// 类型判断
			t := "audio"
			if ext == "mp4" || ext == "webm" || ext == "mkv" || ext == "mpg" || ext == "mpeg" || ext == "avi" || ext == "mov" || ext == "wmv" || ext == "rm" || ext == "rmvb" || ext == "ts" || ext == "flv" {
				t = "video"
			}

			// 歌手/语种/曲种提取（文件名格式：歌手-歌名-语种-歌曲类别）
			name := info.Name()
			singer, language, category := parseFilenameFields(name)

			relPath, _ := filepath.Rel(dir, path)
			// 添加目录前缀以区分不同目录的文件，确保使用正斜杠
			fullPath := filepath.Base(dir) + "/" + strings.ReplaceAll(relPath, "\\", "/")

			list = append(list, MediaFile{
				Name:       name,
				Path:       fullPath,
				Dir:        dir,
				Type:       t,
				Singer:     singer,
				Language:   language,
				Category:   category,
				PinyinInit: computePinyinInitials(strings.TrimSuffix(name, filepath.Ext(name))),
			})

			// 回调进度
			if callback != nil {
				callback(currentCount, totalFiles, info.Name())
			}

			return nil
		})
	}

	return list
}

// 扫描媒体文件（不带进度回调，兼容旧接口）
func getMediaList() []MediaFile {
	return getMediaListWithProgress(nil)
}

// 获取所有歌手列表
func getSingerList() []string {
	singerMap := make(map[string]bool)
	list := getMediaList()
	for _, item := range list {
		if item.Singer != "未知歌手" {
			singerMap[item.Singer] = true
		}
	}
	res := []string{}
	for s := range singerMap {
		res = append(res, s)
	}
	return res
}

// 拆分缓存：固定目录（启动时扫描一次）+ 上传目录（上传后重扫）
var (
	cachedFixedMediaList  []MediaFile // 配置中的固定目录缓存
	cachedUploadMediaList []MediaFile // 上传目录缓存
	uploadDirPath         string      // 上传目录绝对路径
	cachedMergedMediaList []MediaFile // 拼接+去重后的完整缓存
	cachedLowerNames      []string    // 与 cachedMergedMediaList 对应的小写名称，加速搜索
	cachedLowerPinyins    []string    // 与 cachedMergedMediaList 对应的小写拼音首字母，加速搜索
)

// pathToAbsFile 媒体路径到绝对文件路径的映射（从内存曲库构建，O(1)查找）
var pathToAbsFile struct {
	sync.RWMutex
	m map[string]string
}

// basenameToAbsFile 文件名到绝对路径的反向映射（裸文件名查找，避免Walk）
var basenameToAbsFile struct {
	sync.RWMutex
	m map[string][]string // 一个文件名可能对应多个路径
}

// scanDir 扫描单个目录中的媒体文件
func scanDir(dir string) []MediaFile {
	var list []MediaFile

	exts := map[string]bool{
		"mp3": true, "flac": true, "wav": true, "mp4": true, "webm": true,
		"mkv": true, "mpg": true, "mpeg": true, "avi": true, "mov": true,
		"wmv": true, "rm": true, "rmvb": true, "ts": true, "flv": true,
		"aac": true, "m4a": true, "ogg": true, "wma": true, "ape": true,
	}

	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(info.Name()))
		if len(ext) > 1 {
			ext = ext[1:]
		}
		if !exts[ext] {
			return nil
		}

		t := "audio"
		if ext == "mp4" || ext == "webm" || ext == "mkv" || ext == "mpg" || ext == "mpeg" || ext == "avi" || ext == "mov" || ext == "wmv" || ext == "rm" || ext == "rmvb" || ext == "ts" || ext == "flv" {
			t = "video"
		}

		name := info.Name()
		singer, language, category := parseFilenameFields(name)

		relPath, _ := filepath.Rel(dir, path)
		fullPath := filepath.Base(dir) + "/" + strings.ReplaceAll(relPath, "\\", "/")

		list = append(list, MediaFile{
			Name:       name,
			Path:       fullPath,
			Dir:        dir,
			Type:       t,
			Singer:     singer,
			Language:   language,
			Category:   category,
			Size:       info.Size(),
			PinyinInit: computePinyinInitials(strings.TrimSuffix(name, filepath.Ext(name))),
		})

		return nil
	})

	return list
}

// rebuildMergedCache 拼接两个缓存并处理同名歌曲，结果缓存到 cachedMergedMediaList
func rebuildMergedCache() {
	result := make([]MediaFile, 0, len(cachedFixedMediaList)+len(cachedUploadMediaList))
	result = append(result, cachedFixedMediaList...)
	result = append(result, cachedUploadMediaList...)

	// 统计每个文件名出现的次数
	nameCount := make(map[string]int)
	for _, item := range result {
		nameCount[item.Name]++
	}

	// 同名歌曲追加文件大小标注到 DisplayName
	dupCount := 0
	for i, item := range result {
		if nameCount[item.Name] > 1 {
			sizeMB := float64(item.Size) / 1024 / 1024
			ext := filepath.Ext(item.Name)
			base := strings.TrimSuffix(item.Name, ext)
			result[i].DisplayName = fmt.Sprintf("%s (%.1fMB)%s", base, sizeMB, ext)
			dupCount++
		}
	}
	if dupCount > 0 {
		fmt.Printf("[Media] 发现 %d 个同名歌曲，已添加大小标注\n", dupCount)
	}

	cachedMergedMediaList = result

	// 预计算小写名称和拼音首字母，加速搜索
	cachedLowerNames = make([]string, len(result))
	cachedLowerPinyins = make([]string, len(result))
	for i, item := range result {
		cachedLowerNames[i] = strings.ToLower(item.Name)
		cachedLowerPinyins[i] = strings.ToLower(item.PinyinInit)
	}

	// 重建路径→绝对路径映射表，供 findMediaFile O(1) 查找
	buildPathToAbsFileMap()
}

// getCachedMediaList 返回去重后的完整缓存
func getCachedMediaList() []MediaFile {
	if cachedMergedMediaList == nil {
		rebuildMergedCache()
	}
	return cachedMergedMediaList
}

// buildPathToAbsFileMap 从内存曲库构建路径映射表
func buildPathToAbsFileMap() {
	list := getCachedMediaList()
	pathToAbsFile.Lock()
	basenameToAbsFile.Lock()
	pathToAbsFile.m = make(map[string]string, len(list))
	basenameToAbsFile.m = make(map[string][]string, len(list))
	for _, f := range list {
		// Path格式: "dirBase/relPath"，绝对路径 = filepath.Join(Dir, relPath)
		relPath := strings.TrimPrefix(f.Path, filepath.Base(f.Dir)+"/")
		absPath := filepath.Join(f.Dir, relPath)
		pathToAbsFile.m[f.Path] = absPath
		// 同时构建basename反向映射
		basename := f.Name
		basenameToAbsFile.m[basename] = append(basenameToAbsFile.m[basename], absPath)
	}
	basenameToAbsFile.Unlock()
	pathToAbsFile.Unlock()
}

// lookupAbsPath 从内存映射表查找绝对路径
func lookupAbsPath(mediaPath string) (string, bool) {
	pathToAbsFile.RLock()
	p, ok := pathToAbsFile.m[mediaPath]
	pathToAbsFile.RUnlock()
	return p, ok
}

// lookupAbsPathByBasename 通过裸文件名查找绝对路径
func lookupAbsPathByBasename(filename string) (string, bool) {
	basenameToAbsFile.RLock()
	paths, ok := basenameToAbsFile.m[filename]
	basenameToAbsFile.RUnlock()
	if !ok || len(paths) == 0 {
		return "", false
	}
	// 返回第一个存在的文件
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p, true
		}
	}
	return "", false
}

// rescanUploadDir 仅重新扫描上传目录
func rescanUploadDir() {
	if uploadDirPath == "" {
		return
	}
	fmt.Printf("[Media] 重新扫描上传目录: %s\n", uploadDirPath)
	cachedUploadMediaList = scanDir(uploadDirPath)
	fmt.Printf("[Media] 上传目录扫描完成 - 共 %d 首歌曲\n", len(cachedUploadMediaList))
	rebuildMergedCache()
}

const mediaCacheFile = "ktv_media_cache.json"

// MediaCache 缓存数据结构
type MediaCache struct {
	SavedAt  string      `json:"savedAt"`
	MediaDir []string    `json:"mediaDirs"`
	Fixed    []MediaFile `json:"fixedList"`
	Upload   []MediaFile `json:"uploadList"`
}

// saveMediaCache 将当前缓存保存到本地文件
func saveMediaCache() {
	cache := MediaCache{
		SavedAt:  time.Now().Format("2006-01-02 15:04:05"),
		MediaDir: mediaDirs,
		Fixed:    cachedFixedMediaList,
		Upload:   cachedUploadMediaList,
	}
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		fmt.Printf("[Media] 缓存序列化失败: %v\n", err)
		return
	}
	if err := os.WriteFile(mediaCacheFile, data, 0644); err != nil {
		fmt.Printf("[Media] 缓存写入失败: %v\n", err)
		return
	}
	fmt.Printf("[Media] 曲库缓存已保存到 %s (共 %d 首歌曲)\n", mediaCacheFile, len(cachedFixedMediaList)+len(cachedUploadMediaList))
}

// loadMediaCache 从本地文件加载缓存，返回是否成功
func loadMediaCache() bool {
	data, err := os.ReadFile(mediaCacheFile)
	if err != nil {
		return false
	}
	var cache MediaCache
	if err := json.Unmarshal(data, &cache); err != nil {
		fmt.Printf("[Media] 缓存文件格式错误: %v\n", err)
		return false
	}
	// 检查媒体目录是否与缓存一致
	if len(cache.MediaDir) != len(mediaDirs) {
		return false
	}
	dirMap := make(map[string]bool)
	for _, d := range mediaDirs {
		dirMap[filepath.Clean(d)] = true
	}
	for _, d := range cache.MediaDir {
		if !dirMap[filepath.Clean(d)] {
			return false
		}
	}

	// 初始化uploadDirPath（否则FileHandler找不到上传目录的文件）
	exePath, exeErr := os.Executable()
	if exeErr == nil {
		uploadDirPath = filepath.Join(filepath.Dir(exePath), "uploads")
		os.MkdirAll(uploadDirPath, 0755)
	}

	cachedFixedMediaList = cache.Fixed
	cachedUploadMediaList = cache.Upload

	// Dir 字段序列化到缓存，新缓存直接可用
	// 旧缓存无 dir 字段时，restoreDirField 通过 Path 前缀反推恢复
	restoreDirField(cachedFixedMediaList)
	restoreDirField(cachedUploadMediaList)

	rebuildMergedCache()
	fmt.Printf("[Media] 已从缓存加载曲库 (保存时间: %s, 共 %d 首歌曲)\n", cache.SavedAt, len(cachedMergedMediaList))
	return true
}

// restoreDirField 从 Path 前缀和 mediaDirs 反推恢复 Dir 字段
// Path 格式为 "目录名/相对路径"（如 "mtv/子目录/歌曲.mp4"）
// Dir 为绝对路径（如 "f:\mtv"），用于自检统计
func restoreDirField(list []MediaFile) {
	// 建立 目录名→绝对路径列表 的映射
	baseToDirs := make(map[string][]string)
	for _, dir := range mediaDirs {
		base := filepath.Base(dir)
		baseToDirs[base] = append(baseToDirs[base], dir)
	}

	for i := range list {
		if list[i].Dir != "" {
			continue
		}
		slashIdx := strings.Index(list[i].Path, "/")
		var base string
		if slashIdx > 0 {
			base = list[i].Path[:slashIdx]
		} else {
			base = list[i].Path
		}
		dirs, ok := baseToDirs[base]
		if !ok {
			continue
		}
		if len(dirs) == 1 {
			// 唯一匹配，直接赋值
			list[i].Dir = dirs[0]
			continue
		}
		// 多个同名目录：通过检查文件是否实际存在来精确匹配
		// Path 中 base 之后的部分是相对路径（如 "子目录/歌曲.mp4"）
		var relPart string
		if slashIdx > 0 && slashIdx+1 < len(list[i].Path) {
			relPart = list[i].Path[slashIdx+1:]
		}
		if relPart == "" {
			list[i].Dir = dirs[0]
			continue
		}
		matched := false
		for _, dir := range dirs {
			candidate := filepath.Join(dir, filepath.FromSlash(relPart))
			if _, err := os.Stat(candidate); err == nil {
				list[i].Dir = dir
				matched = true
				break
			}
		}
		if !matched {
			// 文件不存在（可能已被移动/删除），回退到第一个同名目录
			list[i].Dir = dirs[0]
		}
	}
}

// hasMediaCache 检查是否存在缓存文件
func hasMediaCache() bool {
	_, err := os.Stat(mediaCacheFile)
	return err == nil
}

// 初始化媒体列表缓存
func initMediaCache() {
	// 计算上传目录路径
	exePath, err := os.Executable()
	if err == nil {
		uploadDirPath = filepath.Join(filepath.Dir(exePath), "uploads")
		os.MkdirAll(uploadDirPath, 0755)
	}

	// 扫描固定目录（配置中的 mediaDirs，跳过上传目录避免重复）
	cachedFixedMediaList = []MediaFile{}
	for _, dir := range mediaDirs {
		if uploadDirPath != "" && filepath.Clean(dir) == filepath.Clean(uploadDirPath) {
			continue
		}
		list := scanDir(dir)
		cachedFixedMediaList = append(cachedFixedMediaList, list...)
	}

	// 扫描上传目录
	cachedUploadMediaList = []MediaFile{}
	if uploadDirPath != "" {
		cachedUploadMediaList = scanDir(uploadDirPath)
	}

	total := len(cachedFixedMediaList) + len(cachedUploadMediaList)
	fmt.Printf("[Media] 曲库扫描完成: 共 %d 首歌曲 (固定:%d + 上传:%d)\n", total, len(cachedFixedMediaList), len(cachedUploadMediaList))

	rebuildMergedCache()
	saveMediaCache()
}

// 分页获取媒体列表
func getMediaListPaged(page, pageSize int, keyword string) ([]MediaFile, int) {
	if len(cachedMergedMediaList) == 0 {
		initMediaCache()
	}

	total := len(cachedMergedMediaList)

	// 空关键词：直接从缓存切片，无需遍历
	if keyword == "" {
		start := (page - 1) * pageSize
		end := start + pageSize
		if start >= total {
			return []MediaFile{}, total
		}
		if end > total {
			end = total
		}
		result := make([]MediaFile, end-start)
		copy(result, cachedMergedMediaList[start:end])
		return result, total
	}

	// 有关键词：使用预计算的小写名称和拼音首字母搜索
	keywords := strings.Fields(strings.ToLower(keyword))
	var filtered []MediaFile
	for i, item := range cachedMergedMediaList {
		lowerName := cachedLowerNames[i]
		lowerPinyin := cachedLowerPinyins[i]
		match := true
		for _, kw := range keywords {
			if !strings.Contains(lowerName, kw) && !strings.Contains(lowerPinyin, kw) {
				match = false
				break
			}
		}
		if match {
			filtered = append(filtered, item)
		}
	}

	total = len(filtered)

	start := (page - 1) * pageSize
	end := start + pageSize

	if start >= total {
		return []MediaFile{}, total
	}

	if end > total {
		end = total
	}

	return filtered[start:end], total
}