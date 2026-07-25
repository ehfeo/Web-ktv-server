package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

// buildStreamArgs 构建FFmpeg流媒体转码参数
func buildStreamArgs(foundPath string, trackIndex int, quality string, encoder string) []string {
	args := []string{
		"-analyzeduration", "2000000", // 2秒分析，更准确检测编码格式
		"-fflags", "+genpts+discardcorrupt",
	}

	// GPU全流水线：-hwaccel cuda 让FFmpeg自动选CUVID解码器（h264_cuvid/hevc_cuvid等）
	// 数据全程留在GPU显存，解码→缩放→编码不经过CPU内存，最大化GPU利用率
	if strings.Contains(encoder, "nvenc") {
		args = append(args, "-hwaccel", "cuda")
	}

	args = append(args, "-i", foundPath)

	// 映射视频流和指定音轨
	args = append(args, "-map", "0:v:0")
	args = append(args, "-map", fmt.Sprintf("0:a:%d", trackIndex))

	// 视频滤镜：降分辨率（KTV画面降分辨率是提升画质最有效的手段）
	if quality == "high" {
		args = append(args, "-vf", "scale=-2:720")
	} else {
		args = append(args, "-vf", "scale=-2:480")
	}

	// 视频编码器（KTV优化：低运动场景用较长GOP+film调优）
	// 注意：NVENC/QSV/AMF的某些高级参数（rc-lookahead, spatial-aq等）
	// 依赖特定SDK版本和GPU型号，不通用，因此只使用基础参数
	if strings.Contains(encoder, "nvenc") {
		args = append(args, "-c:v", "h264_nvenc",
			"-preset", "p4",
			"-tune", "hq",
			"-profile:v", "high",
			"-rc", "vbr",
			"-cq", "28",
			"-b:v", "384k",
			"-maxrate", "512k",
			"-bufsize", "1024k",
			"-g", "50",
			"-keyint_min", "25")
	} else if strings.Contains(encoder, "qsv") {
		args = append(args, "-c:v", "h264_qsv",
			"-preset", "medium",
			"-profile:v", "high",
			"-b:v", "384k",
			"-maxrate", "512k",
			"-bufsize", "1024k",
			"-g", "50",
			"-keyint_min", "25")
	} else if strings.Contains(encoder, "amf") {
		args = append(args, "-c:v", "h264_amf",
			"-quality", "balanced",
			"-profile:v", "high",
			"-rc", "vbr_peak",
			"-b:v", "384k",
			"-maxrate", "512k",
			"-bufsize", "1024k",
			"-g", "50",
			"-keyint_min", "25")
	} else {
		args = append(args, "-c:v", "libx264",
			"-preset", "fast",
			"-tune", "film",
			"-profile:v", "high",
			"-b:v", "384k",
			"-maxrate", "512k",
			"-bufsize", "1024k",
			"-g", "50",
			"-keyint_min", "25",
			"-bf", "3",
			"-rc-lookahead", "20",
			"-aq-mode", "3")
	}

	// 音频编码器
	args = append(args, "-c:a", "aac", "-b:a", "128k")

	// Fragmented MP4输出
	args = append(args, "-movflags", "frag_keyframe+empty_moov+default_base_moof")
	args = append(args, "-f", "mp4", "pipe:1")

	return args
}

// tryStream 尝试用指定参数启动流媒体，返回是否成功
// 如果FFmpeg在发送首字节前就失败（编码器初始化失败），返回false
func tryStream(w http.ResponseWriter, r *http.Request, ffmpegPath string, args []string, encoder string) bool {
	fmt.Printf("[流媒体] 启动: 编码器=%s\n", encoder)

	cmd := exec.CommandContext(r.Context(), ffmpegPath, args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Printf("[流媒体] 创建stdout管道失败: %v\n", err)
		return false
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		fmt.Printf("[流媒体] 创建stderr管道失败: %v\n", err)
		return false
	}

	if err := cmd.Start(); err != nil {
		fmt.Printf("[流媒体] 启动FFmpeg失败: %v\n", err)
		return false
	}

	// 用管道读取stderr，检测编码器初始化是否成功
	var stderrBuf strings.Builder
	var stderrMu sync.Mutex
	stderrDone := make(chan struct{})
	go func() {
		defer close(stderrDone)
		buf := make([]byte, 4096)
		for {
			n, err := stderr.Read(buf)
			if err != nil {
				return
			}
			if n > 0 {
				output := string(buf[:n])
				stderrMu.Lock()
				stderrBuf.WriteString(output)
				stderrMu.Unlock()
				trimmed := strings.TrimSpace(output)
				if trimmed != "" && (strings.Contains(trimmed, "Error") || strings.Contains(trimmed, "error") || strings.Contains(trimmed, "fail")) {
					fmt.Printf("[流媒体] FFmpeg: %s\n", trimmed)
				}
			}
		}
	}()

	// 等待首字节数据或FFmpeg退出
	// 如果5秒内没有首字节数据，检查stderr是否有错误
	readBuf := make([]byte, 32*1024)
	firstByteTimeout := time.After(5 * time.Second)
	firstByteReceived := false

	n, readErr := stdout.Read(readBuf)
	if n > 0 {
		firstByteReceived = true
	} else {
		// 没有首字节数据，等待超时或进程退出
		select {
		case <-firstByteTimeout:
		case <-stderrDone:
		}
		// 再试读一次
		if !firstByteReceived {
			n2, readErr2 := stdout.Read(readBuf)
			if n2 > 0 {
				n = n2
				readErr = readErr2
				firstByteReceived = true
			}
		}
	}

	if !firstByteReceived {
		// FFmpeg没有输出任何数据，检查错误
		cmd.Process.Kill()
		cmd.Wait()
		stderrMu.Lock()
		errOutput := stderrBuf.String()
		stderrMu.Unlock()

		isEncoderError := strings.Contains(errOutput, "No capable devices") ||
			strings.Contains(errOutput, "not supported") ||
			strings.Contains(errOutput, "Error initializing") ||
			strings.Contains(errOutput, "Conversion failed") ||
			strings.Contains(errOutput, "cannot open encoder") ||
			strings.Contains(errOutput, "failed to open") ||
			readErr != nil

		if isEncoderError {
			fmt.Printf("[流媒体] 编码器 %s 初始化失败，将回退到CPU编码\n", encoder)
			return false
		}
		return false
	}

	// 首字节已收到，编码器工作正常，开始流式传输
	w.Header().Set("Content-Type", "video/mp4")
	w.Header().Set("Cache-Control", "no-cache, no-store")
	w.Header().Set("Connection", "keep-alive")

	// 流控参数
	const initialBurst int64 = 1024 * 1024 // 初始突发量 1MB，保证快速启动
	const startupDelay = 3 * time.Second   // 客户端启动播放延迟估算
	const bufferDuration = 15.0            // 客户端最大缓冲秒数
	const maxRate int64 = 88 * 1024        // 最大发送速率 88KB/s = 700kbps

	startTime := time.Now()

	// 写入首块数据
	w.Write(readBuf[:n])
	var totalWritten int64 = int64(n)
	flusher, canFlush := w.(http.Flusher)
	if canFlush {
		flusher.Flush()
	}

	// 继续流式传输
	for {
		n, readErr := stdout.Read(readBuf)
		if n > 0 {
			written, writeErr := w.Write(readBuf[:n])
			totalWritten += int64(written)
			if canFlush {
				flusher.Flush()
			}
			if writeErr != nil {
				fmt.Printf("[流媒体] 写入中断: %v (已发送 %d 字节)\n", writeErr, totalWritten)
				cmd.Process.Kill()
				cmd.Wait()
				return true
			}

			// 动态流控：限制客户端缓冲不超过bufferDuration秒
			// 使用实际编码码率（基于FFmpeg输出统计）和最大码率的双重策略
			if totalWritten > initialBurst {
				elapsed := time.Since(startTime)
				// 基于最大码率的限制：确保不超过 maxRate + initialBurst
				maxAllowed := int64(elapsed.Seconds()*float64(maxRate)) + initialBurst
				if totalWritten > maxAllowed {
					excessBytes := totalWritten - maxAllowed
					sleepDuration := time.Duration(float64(excessBytes) / float64(maxRate) * float64(time.Second))
					if sleepDuration > 5*time.Second {
						sleepDuration = 5 * time.Second
					}
					if sleepDuration > 0 {
						time.Sleep(sleepDuration)
					}
				}

				// 基于实际码率的缓冲限制：确保客户端缓冲不超过bufferDuration秒
				if elapsed.Seconds() > 0 {
					actualRate := float64(totalWritten) / elapsed.Seconds()
					playbackElapsed := elapsed - startupDelay
					if playbackElapsed < 0 {
						playbackElapsed = 0
					}
					consumedBytes := playbackElapsed.Seconds() * actualRate
					clientBuffer := float64(totalWritten) - consumedBytes
					maxBuffer := bufferDuration * actualRate
					if clientBuffer > maxBuffer {
						excessDuration := (clientBuffer - maxBuffer) / actualRate
						sleepDuration := time.Duration(excessDuration * float64(time.Second))
						if sleepDuration > 5*time.Second {
							sleepDuration = 5 * time.Second
						}
						if sleepDuration > 0 {
							time.Sleep(sleepDuration)
						}
					}
				}
			}
		}
		if readErr != nil {
			break
		}
	}
	cmd.Wait()

	fmt.Printf("[流媒体] 传输完成: %d 字节, 耗时: %v\n", totalWritten, time.Since(startTime).Round(time.Second))
	return true
}

// StreamHandler 实时转码流媒体端点，输出Fragmented MP4
// 支持GPU加速（NVENC/QSV/AMF），GPU失败自动回退到CPU编码
// 流控：仅缓冲15秒数据，边播边传，节省带宽
func StreamHandler(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, "缺少name参数", 400)
		return
	}

	trackIndex := 0
	if ti := r.URL.Query().Get("trackIndex"); ti != "" {
		if v, err := strconv.Atoi(ti); err == nil && v >= 0 {
			trackIndex = v
		}
	}

	quality := r.URL.Query().Get("quality")
	if quality == "" {
		quality = "low"
	}

	foundPath := findMediaFile(name)
	if foundPath == "" {
		http.Error(w, "文件未找到", 404)
		return
	}

	// 检查文件轨道完整性（无视频/无音频等异常情况）
	trackWarning := checkMediaTracks(foundPath)
	if trackWarning != nil {
		// 通过HTTP头传递警告信息，前端可读取并显示
		w.Header().Set("X-Track-Warning", trackWarning.Message)
		if trackWarning.NoVideo {
			w.Header().Set("X-No-Video", "1")
		}
		if trackWarning.NoAudio {
			w.Header().Set("X-No-Audio", "1")
		}
	}

	ffmpegPath := getFFmpegPath()

	// 三级回退策略：GPU全流水线 → CPU解码+GPU编码 → 纯CPU编码
	if videoEncoder != "libx264" {
		// 第一级：GPU全流水线（-hwaccel cuda + NVENC）
		args := buildStreamArgs(foundPath, trackIndex, quality, videoEncoder)
		if tryStream(w, r, ffmpegPath, args, videoEncoder+"(全流水线)") {
			return
		}
		// 第二级：CPU解码 + GPU编码（无 -hwaccel）
		fmt.Printf("[流媒体] 降级: CPU解码 + GPU编码(%s)\n", videoEncoder)
		argsNoHwAccel := buildStreamArgsNoHwAccel(foundPath, trackIndex, quality, videoEncoder)
		if tryStream(w, r, ffmpegPath, argsNoHwAccel, videoEncoder) {
			return
		}
		// 第三级：纯CPU编码
		fmt.Printf("[流媒体] 回退到CPU编码 (libx264)\n")
	}

	// CPU编码（libx264）
	args := buildStreamArgs(foundPath, trackIndex, quality, "libx264")
	if !tryStream(w, r, ffmpegPath, args, "libx264") {
		http.Error(w, "转码启动失败", 500)
	}
}

// buildStreamArgsNoHwAccel 构建流媒体参数（CPU解码 + GPU编码，降级模式）
// 不添加 -hwaccel cuda，仅用GPU做编码，数据需CPU→GPU搬运
func buildStreamArgsNoHwAccel(foundPath string, trackIndex int, quality string, encoder string) []string {
	args := []string{
		"-analyzeduration", "2000000",
		"-fflags", "+genpts+discardcorrupt",
		"-i", foundPath,
	}

	// 映射视频流和指定音轨
	args = append(args, "-map", "0:v:0")
	args = append(args, "-map", fmt.Sprintf("0:a:%d", trackIndex))

	// 视频滤镜
	if quality == "high" {
		args = append(args, "-vf", "scale=-2:720")
	} else {
		args = append(args, "-vf", "scale=-2:480")
	}

	// 编码器参数（与buildStreamArgs相同，但无-hwaccel cuda）
	if strings.Contains(encoder, "nvenc") {
		args = append(args, "-c:v", "h264_nvenc",
			"-preset", "p4",
			"-tune", "hq",
			"-profile:v", "high",
			"-rc", "vbr",
			"-cq", "28",
			"-b:v", "384k",
			"-maxrate", "512k",
			"-bufsize", "1024k",
			"-g", "50",
			"-keyint_min", "25")
	} else if strings.Contains(encoder, "qsv") {
		args = append(args, "-c:v", "h264_qsv",
			"-preset", "medium",
			"-profile:v", "high",
			"-b:v", "384k",
			"-maxrate", "512k",
			"-bufsize", "1024k",
			"-g", "50",
			"-keyint_min", "25")
	} else if strings.Contains(encoder, "amf") {
		args = append(args, "-c:v", "h264_amf",
			"-quality", "balanced",
			"-profile:v", "high",
			"-rc", "vbr_peak",
			"-b:v", "384k",
			"-maxrate", "512k",
			"-bufsize", "1024k",
			"-g", "50",
			"-keyint_min", "25")
	} else {
		args = append(args, "-c:v", "libx264",
			"-preset", "fast",
			"-tune", "film",
			"-profile:v", "high",
			"-b:v", "384k",
			"-maxrate", "512k",
			"-bufsize", "1024k",
			"-g", "50",
			"-keyint_min", "25",
			"-bf", "3",
			"-rc-lookahead", "20",
			"-aq-mode", "3")
	}

	args = append(args, "-c:a", "aac", "-b:a", "128k")
	args = append(args, "-movflags", "frag_keyframe+empty_moov+default_base_moof")
	args = append(args, "-f", "mp4", "pipe:1")

	return args
}

// MediaDurationHandler 返回媒体文件的时长（秒）
func MediaDurationHandler(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, "缺少name参数", 400)
		return
	}

	foundPath := findMediaFile(name)
	if foundPath == "" {
		http.Error(w, "文件未找到", 404)
		return
	}

	duration := getDuration(foundPath)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"duration": duration,
	})
}
