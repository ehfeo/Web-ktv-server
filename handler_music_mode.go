package main

import (
	"net/http"
	"os/exec"
	"strconv"
	"strings"
)

// MusicModeStreamHandler 音乐模式：从视频文件中抽取音频流并转码为AAC流式输出
func MusicModeStreamHandler(w http.ResponseWriter, r *http.Request) {
	fileName := r.URL.Query().Get("name")
	if fileName == "" {
		http.Error(w, "缺少文件名", 400)
		return
	}

	trackIndexStr := r.URL.Query().Get("trackIndex")
	trackIndex := 0
	if trackIndexStr != "" {
		if idx, err := strconv.Atoi(trackIndexStr); err == nil {
			trackIndex = idx
		}
	}

	filePath := findMediaFile(fileName)
	if filePath == "" {
		http.Error(w, "文件未找到", 404)
		return
	}

	ffmpegPath := getFFmpegPath()
	ffprobePath := getFFprobePath()

	// 检测音轨数，如果请求的trackIndex超出范围，回退到0
	// 纯音频文件（ape/wma等）只有一个音轨，trackIndex=1会失败
	if trackIndex > 0 {
		probeCmd := exec.Command(ffprobePath, "-v", "error", "-select_streams", "a", "-show_entries", "stream=index", "-of", "csv=p=0", filePath)
		probeOut, err := probeCmd.Output()
		if err == nil {
			audioStreamCount := strings.Count(string(probeOut), "\n")
			if audioStreamCount <= trackIndex {
				trackIndex = 0
			}
		}
	}

	// 构建ffmpeg命令：抽取指定音轨，转码为AAC，输出为ADTS/AAC流
	// -map 0:a:{trackIndex} 选择指定音轨（0=原唱, 1=伴奏）
	// 如果音轨已经是AAC且浏览器支持，直接copy；否则转码为AAC
	args := []string{
		"-i", filePath,
		"-map", "0:a:" + strconv.Itoa(trackIndex),
		"-c:a", "aac",
		"-b:a", strconv.Itoa(audioTranscodeBitrate) + "k",
		"-f", "adts",
		"-",
	}

	cmd := exec.Command(ffmpegPath, args...)
	cmd.Stderr = nil

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		http.Error(w, "创建管道失败", 500)
		return
	}

	if err := cmd.Start(); err != nil {
		http.Error(w, "启动ffmpeg失败", 500)
		return
	}

	// 设置响应头
	w.Header().Set("Content-Type", "audio/aac")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "close")

	// 流式输出
	buf := make([]byte, 4096)
	for {
		n, err := stdout.Read(buf)
		if n > 0 {
			if _, werr := w.Write(buf[:n]); werr != nil {
				cmd.Process.Kill()
				break
			}
		}
		if err != nil {
			break
		}
	}

	cmd.Wait()
}

// isMusicMode 检查请求是否开启了音乐模式
func isMusicMode(r *http.Request) bool {
	return r.URL.Query().Get("musicMode") == "1"
}
