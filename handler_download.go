package main

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// ============================================================
// 下载已点歌曲
// 三种模式：
//   mode=1 原文件下载
//   mode=2 仅抽取音频数据流（-c:a copy，按音频编码生成合适的文件后缀）
//   mode=3 下载为MP3（非MP3音轨按 libmp3lame 256kbps 转码；源已是MP3则直接拷贝）
// 多音轨文件：前端按音轨序号逐个下载，文件名为 原文件名(音轨N).ext
// ============================================================

// downloadCacheDir 下载产物缓存目录：<程序运行目录>/download_cache
func downloadCacheDir() string {
	dir, err := os.Getwd()
	if err != nil || dir == "" {
		dir = "."
	}
	return filepath.Join(dir, "download_cache")
}

// cleanDownloadCache 清理超过 maxAge 的下载缓存文件（尽力而为，忽略错误）
func cleanDownloadCache(maxAge time.Duration) {
	dir := downloadCacheDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	cutoff := time.Now().Add(-maxAge)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			os.Remove(filepath.Join(dir, e.Name()))
		}
	}
}

// downloadFileKey 产物缓存键：绝对路径 + mtime + mode + 音轨序号
func downloadFileKey(absPath string, modTime time.Time, mode, track int) string {
	h := sha1.Sum([]byte(fmt.Sprintf("%s|%d|%d|%d", absPath, modTime.UnixNano(), mode, track)))
	return hex.EncodeToString(h[:])[:16]
}

func fileExists(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && !fi.IsDir() && fi.Size() > 0
}

// contentDisposition 生成 Content-Disposition（RFC 6266）：
// 同时给出 ASCII 兜底文件名和 UTF-8 百分号编码文件名，保证中文/全角括号在
// Chrome/Edge/Firefox 下载框里都显示为原始歌名。
func contentDisposition(filename string) string {
	fallback := make([]byte, 0, len(filename))
	for i := 0; i < len(filename); i++ {
		c := filename[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') ||
			c == '.' || c == '-' || c == '_' {
			fallback = append(fallback, c)
		} else {
			fallback = append(fallback, '_')
		}
	}
	escaped := url.QueryEscape(filename)
	escaped = strings.ReplaceAll(escaped, "+", "%20")
	return fmt.Sprintf("attachment; filename=\"%s\"; filename*=UTF-8''%s", string(fallback), escaped)
}

// audioExtByCodec mode=2：按音频流编码选择合适文件后缀。
// 返回空串表示没有合适的独立容器（如 ape/cook），由调用方退回MP3转码。
func audioExtByCodec(codec string) string {
	switch strings.ToLower(strings.TrimSpace(codec)) {
	case "aac":
		return ".aac"
	case "mp3", "mp3float", "mp3adu", "mp3on4":
		return ".mp3"
	case "flac":
		return ".flac"
	case "opus":
		return ".opus"
	case "vorbis":
		return ".ogg"
	case "ac3":
		return ".ac3"
	case "eac3":
		return ".eac3"
	case "dts":
		return ".dts"
	case "wmav2", "wmav1", "wmavoice":
		return ".wma"
	case "alac":
		return ".m4a"
	case "amr_nb", "amr_wb":
		return ".amr"
	case "pcm_s16le", "pcm_s24le", "pcm_s32le", "pcm_f32le", "pcm_s16be", "pcm_s24be",
		"pcm_s32be", "pcm_f32be", "pcm_u8", "pcm_s8", "pcm_mulaw", "pcm_alaw":
		return ".wav"
	}
	return ""
}

type downloadTrackInfo struct {
	Index    int    `json:"index"`    // 音频流序号（0-based，对应 ffmpeg -map 0:a:N）
	Codec    string `json:"codec"`    // 音频编码
	Channels int    `json:"channels"` // 声道数
}

// probeAudioTracks 探测文件音频轨列表。
// 复用 ffprobeMutex 串行锁，避免下载探测与点歌检查互相争抢曲库盘磁头。
func probeAudioTracks(filePath string) ([]downloadTrackInfo, error) {
	ffprobeMutex.Lock()
	defer ffprobeMutex.Unlock()

	ffprobePath := getFFprobePath()
	cmd := exec.Command(ffprobePath, "-v", "error",
		"-select_streams", "a",
		"-show_entries", "stream=codec_name,channels",
		"-of", "json", filePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("ffprobe失败: %v", err)
	}
	var pr struct {
		Streams []struct {
			CodecName string `json:"codec_name"`
			Channels  int    `json:"channels"`
		} `json:"streams"`
	}
	if err := json.Unmarshal(output, &pr); err != nil {
		return nil, fmt.Errorf("解析ffprobe输出失败: %v", err)
	}
	tracks := make([]downloadTrackInfo, 0, len(pr.Streams))
	for i, s := range pr.Streams {
		tracks = append(tracks, downloadTrackInfo{
			Index:    i,
			Codec:    strings.TrimSpace(s.CodecName),
			Channels: s.Channels,
		})
	}
	return tracks, nil
}

// resolveDownloadPath 解析下载目标文件的绝对路径。
// 与 FileHandler 保持一致：普通曲库文件走 findMediaFile + 审核拦截；
// Everything 直播项（abs=1&p=）按绝对路径直接取用（需设置里已勾选 Everything）。
func resolveDownloadPath(w http.ResponseWriter, r *http.Request) (string, bool) {
	if r.URL.Query().Get("abs") == "1" {
		absPath := r.URL.Query().Get("p")
		if idx := strings.Index(absPath, "?"); idx != -1 {
			absPath = absPath[:idx]
		}
		if everythingEnabled && absPath != "" {
			if fi, err := os.Stat(absPath); err == nil && !fi.IsDir() && mediaExtSupported(absPath) {
				return absPath, true
			}
		}
		http.Error(w, "not found", 404)
		return "", false
	}

	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, "缺少name参数", 400)
		return "", false
	}
	if idx := strings.Index(name, "?"); idx != -1 {
		name = name[:idx]
	}

	foundPath := findMediaFile(name)
	if foundPath == "" {
		http.Error(w, "文件未找到", 404)
		return "", false
	}
	if !reviewFileAllowed(foundPath) {
		http.Error(w, "该资源正在审核中，暂不可下载。", 403)
		return "", false
	}
	return foundPath, true
}

// DownloadTracksHandler GET /api/download/tracks?name=X
// 返回文件的音频轨数量与各轨编码，前端据此决定下载几个文件。
func DownloadTracksHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	absPath, ok := resolveDownloadPath(w, r)
	if !ok {
		return
	}

	tracks, err := probeAudioTracks(absPath)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"fileName": filepath.Base(absPath),
		"tracks":   tracks,
	})
}

// DownloadHandler GET /api/download?name=X&mode=1|2|3&track=N
// mode=1：原文件；mode=2：抽音频流按原编码；mode=3：统一MP3 256kbps。
// track 为音频轨序号（0-based），仅 mode=2/3 有效。
func DownloadHandler(w http.ResponseWriter, r *http.Request) {
	absPath, ok := resolveDownloadPath(w, r)
	if !ok {
		return
	}

	mode, err := strconv.Atoi(r.URL.Query().Get("mode"))
	if err != nil || mode < 1 || mode > 3 {
		http.Error(w, "mode参数错误(1/2/3)", 400)
		return
	}
	trackIdx := 0
	if t := r.URL.Query().Get("track"); t != "" {
		if v, err := strconv.Atoi(t); err == nil && v >= 0 {
			trackIdx = v
		}
	}

	fileInfo, statErr := os.Stat(absPath)
	if statErr != nil {
		http.Error(w, "文件不存在", 404)
		return
	}
	baseName := strings.TrimSuffix(filepath.Base(absPath), filepath.Ext(absPath))

	// mode=1：原文件直接下载（多音轨也只是一个文件，原样带走）
	if mode == 1 {
		log.Printf("[下载] 原文件下载: file=%s", filepath.Base(absPath))
		w.Header().Set("Content-Disposition", contentDisposition(filepath.Base(absPath)))
		http.ServeFile(w, r, absPath)
		return
	}

	// mode=2/3：先探音频轨
	tracks, err := probeAudioTracks(absPath)
	if err != nil {
		http.Error(w, "读取音频轨失败: "+err.Error(), 500)
		return
	}
	if len(tracks) == 0 {
		http.Error(w, "该文件没有音频轨", 404)
		return
	}
	if trackIdx >= len(tracks) {
		http.Error(w, "音轨序号超出范围", 400)
		return
	}
	tr := tracks[trackIdx]
	multiTrack := len(tracks) > 1

	// 多音轨时按 原文件名(音轨N) 命名；单音轨保持 原文件名
	trackSuffix := ""
	if multiTrack {
		trackSuffix = fmt.Sprintf("(音轨%d)", trackIdx+1)
	}

	cleanDownloadCache(24 * time.Hour)
	cacheDir := downloadCacheDir()
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		http.Error(w, "无法创建下载缓存目录", 500)
		return
	}
	ffmpegPath := getFFmpegPath()
	ctx := r.Context() // 客户端断开时杀掉 ffmpeg，避免残留进程

	var outPath, outName string

	if mode == 2 {
		ext := audioExtByCodec(tr.Codec)
		if ext == "" {
			// 无合适独立容器（ape/cook 等）：退回 MP3 256k，保证一定能下载
			log.Printf("[下载] 编码 %s 无合适独立容器，退回MP3 256k: file=%s track=%d",
				tr.Codec, filepath.Base(absPath), trackIdx)
			mode = 3
		} else {
			outPath = filepath.Join(cacheDir, downloadFileKey(absPath, fileInfo.ModTime(), 2, trackIdx)+ext)
			outName = baseName + trackSuffix + ext
			if !fileExists(outPath) {
				log.Printf("[下载] 抽取音频流(mode2): codec=%s track=%d file=%s",
					tr.Codec, trackIdx, filepath.Base(absPath))
				args := []string{"-v", "error", "-y", "-i", absPath,
					"-map", fmt.Sprintf("0:a:%d", trackIdx), "-c:a", "copy", outPath}
				if out, err := exec.CommandContext(ctx, ffmpegPath, args...).CombinedOutput(); err != nil {
					log.Printf("[下载] mode2抽取失败(%v)，退回MP3: %s", err, strings.TrimSpace(string(out)))
					os.Remove(outPath)
					mode = 3
				}
			}
		}
	}

	if mode == 3 {
		// 源音轨已是MP3：直接拷贝，不做二次有损编码；否则统一 libmp3lame 256kbps
		codecArgs := []string{"-c:a", "libmp3lame", "-b:a", "256k"}
		if strings.EqualFold(tr.Codec, "mp3") {
			codecArgs = []string{"-c:a", "copy"}
			log.Printf("[下载] 音轨已是MP3，直接拷贝: file=%s track=%d", filepath.Base(absPath), trackIdx)
		} else {
			log.Printf("[下载] 转码MP3 256k(mode3): codec=%s track=%d file=%s",
				tr.Codec, trackIdx, filepath.Base(absPath))
		}
		outPath = filepath.Join(cacheDir, downloadFileKey(absPath, fileInfo.ModTime(), 3, trackIdx)+".mp3")
		outName = baseName + trackSuffix + ".mp3"
		if !fileExists(outPath) {
			args := append([]string{"-v", "error", "-y", "-i", absPath,
				"-map", fmt.Sprintf("0:a:%d", trackIdx)}, codecArgs...)
			args = append(args, outPath)
			if out, err := exec.CommandContext(ctx, ffmpegPath, args...).CombinedOutput(); err != nil {
				log.Printf("[下载] MP3转码失败: %v %s", err, strings.TrimSpace(string(out)))
				os.Remove(outPath)
				http.Error(w, "音频转码失败: "+err.Error(), 500)
				return
			}
		}
	}

	log.Printf("[下载] 发送文件: %s (%s)", outName, filepath.Base(absPath))
	w.Header().Set("Content-Disposition", contentDisposition(outName))
	http.ServeFile(w, r, outPath)
}
