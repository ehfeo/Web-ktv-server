package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

// 自检结果全局变量（启动时填充，供 /api/selfcheck 与欢迎页面使用）
var selfCheckResult *SelfCheckReport

// SelfCheckReport 自检报告
type SelfCheckReport struct {
	GeneratedAt    string          `json:"generatedAt"`    // 报告生成时间
	Arch           string          `json:"arch"`           // 程序架构 (386/amd64)
	FFmpeg         FFmpegCheck     `json:"ffmpeg"`         // FFmpeg 依赖检查
	FFprobe        FFprobeCheck    `json:"ffprobe"`        // FFprobe 依赖检查
	GPU            GPUCheck        `json:"gpu"`            // GPU 加速检查
	MediaDirs      []MediaDirStat  `json:"mediaDirs"`      // 曲库目录状态
	TotalSongs     int             `json:"totalSongs"`     // 曲库有效文件总数
	MediaScanDone  bool            `json:"mediaScanDone"`  // 曲库扫描是否完成
	OverallStatus  string          `json:"overallStatus"`  // ok / warning / critical
	BlockingIssues []BlockingIssue `json:"blockingIssues"` // 阻断性问题（需用户处理）
}

// FFmpegCheck FFmpeg 自检结果
type FFmpegCheck struct {
	Found       bool   `json:"found"`
	Path        string `json:"path"`
	Version     string `json:"version"`     // 版本字符串第一行
	HasEncoders bool   `json:"hasEncoders"` // -encoders 是否可用
}

// FFprobeCheck FFprobe 自检结果
type FFprobeCheck struct {
	Found   bool   `json:"found"`
	Path    string `json:"path"`
	Version string `json:"version"`
}

// GPUCheck GPU 加速自检结果
type GPUCheck struct {
	DetectedEncoder string         `json:"detectedEncoder"` // 实际生效的编码器
	IsGPU           bool           `json:"isGPU"`           // 是否为 GPU 编码器
	HWAccelCUDA     bool           `json:"hwAccelCuda"`     // -hwaccel cuda 是否可用（GPU全流水线）
	Encoders        []EncoderProbe `json:"encoders"`        // 各编码器探测详情
}

// EncoderProbe 单个编码器探测结果
type EncoderProbe struct {
	Name      string `json:"name"`
	Available bool   `json:"available"` // 编译进 ffmpeg
	Usable    bool   `json:"usable"`    // 实际可工作（驱动+硬件）
	Detail    string `json:"detail"`    // 错误信息或成功说明
}

// MediaDirStat 单个曲库目录状态（自检时仅检查目录存在性，文件计数由曲库扫描步骤填充）
type MediaDirStat struct {
	Path        string `json:"path"`
	Exists      bool   `json:"exists"`
	FileCount   int    `json:"fileCount"`   // 曲库扫描后由 updateSelfCheckMediaStats 填充
	VideoCount  int    `json:"videoCount"`
	AudioCount  int    `json:"audioCount"`
	IsReadable  bool   `json:"isReadable"`
	ErrorReason string `json:"errorReason,omitempty"`
}

// BlockingIssue 阻断性问题：需要用户处理才能正常使用
type BlockingIssue struct {
	Level      string `json:"level"`               // critical / warning
	Category   string `json:"category"`            // ffmpeg / ffprobe / mediaDir / gpu
	Title      string `json:"title"`
	Detail     string `json:"detail"`
	Action     string `json:"action"`              // 解决方案描述
	ActionURL  string `json:"actionURL,omitempty"` // 跳转URL（如 /settings）
	ActionType string `json:"actionType,omitempty"`// link / button / settings
}

// runSelfCheck 执行完整的系统自检
// 注意：曲库扫描结果由调用方在扫描完成后通过 updateSelfCheckMediaStats 更新
func runSelfCheck() *SelfCheckReport {
	report := &SelfCheckReport{
		GeneratedAt: time.Now().Format("2006-01-02 15:04:05"),
		Arch:        archLabel(),
	}

	// 1. FFmpeg 检查
	fmt.Println("  [1/4] 检查 FFmpeg...")
	report.FFmpeg = checkFFmpeg()
	// 2. FFprobe 检查
	fmt.Println("  [2/4] 检查 FFprobe...")
	report.FFprobe = checkFFprobe()
	// 3. GPU 加速检查（仅在 ffmpeg 可用时）
	fmt.Println("  [3/4] 检查 GPU 加速...")
	report.GPU = checkGPUAcceleration(report.FFmpeg.Found)
	// 4. 曲库目录检查（仅检查目录存在性，不遍历文件）
	fmt.Println("  [4/4] 检查曲库目录...")
	report.MediaDirs = checkMediaDirs()

	// 综合评估
	report.BlockingIssues = collectBlockingIssues(report)
	report.OverallStatus = computeOverallStatus(report.BlockingIssues)

	return report
}

// updateSelfCheckMediaStats 在曲库扫描/缓存加载完成后更新自检报告中的歌曲数
// mediaList 为完整的缓存歌曲列表，从中统计每个目录的文件数
// 使用 MediaFile.Dir（绝对路径）精确匹配，避免同名目录歧义
func updateSelfCheckMediaStats(mediaList []MediaFile) {
	if selfCheckResult == nil {
		return
	}

	totalSongs := len(mediaList)
	selfCheckResult.TotalSongs = totalSongs
	selfCheckResult.MediaScanDone = true

	// 建立绝对路径→目录统计 的映射
	dirStats := make(map[string]*MediaDirStat)
	for i := range selfCheckResult.MediaDirs {
		// 统一路径分隔符，确保匹配
		normalized := strings.ReplaceAll(selfCheckResult.MediaDirs[i].Path, "/", string(os.PathSeparator))
		dirStats[normalized] = &selfCheckResult.MediaDirs[i]
	}

	for _, mf := range mediaList {
		dir := mf.Dir
		if dir == "" {
			continue
		}
		// 统一路径分隔符
		dir = strings.ReplaceAll(dir, "/", string(os.PathSeparator))
		if stat, ok := dirStats[dir]; ok {
			stat.FileCount++
			if mf.Type == "video" {
				stat.VideoCount++
			} else {
				stat.AudioCount++
			}
		}
	}

	// 重新评估阻断性问题
	selfCheckResult.BlockingIssues = collectBlockingIssues(selfCheckResult)
	selfCheckResult.OverallStatus = computeOverallStatus(selfCheckResult.BlockingIssues)
}

// archLabel 返回当前进程架构的人类可读标签
// 通过编译期常量 uintptrBits 判定，32位与64位编译产物都正确
func archLabel() string {
	if uintptrBits == 64 {
		return "64位 (amd64)"
	}
	return "32位 (386)"
}

// uintptrBits 编译期常量：指针位宽
// 在 32 位平台上为 32，在 64 位平台上为 64
// 计算原理：^uint(0) 在 32 位下高位为 0（>>63 得 0），在 64 位下高位为 1（>>63 得 1）
var uintptrBits = 32 << (^uint(0) >> 63)

// checkFFmpeg 检测 ffmpeg 是否可用
func checkFFmpeg() FFmpegCheck {
	result := FFmpegCheck{}
	ffmpegPath := getFFmpegPath()
	result.Path = ffmpegPath

	// 尝试调用 ffmpeg -version
	cmd := exec.Command(ffmpegPath, "-version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		result.Found = false
		result.Version = fmt.Sprintf("调用失败: %v", err)
		return result
	}
	result.Found = true
	// 取第一行作为版本信息
	lines := strings.Split(string(output), "\n")
	if len(lines) > 0 {
		result.Version = strings.TrimSpace(lines[0])
	}

	// 检测 -encoders 选项
	encCmd := exec.Command(ffmpegPath, "-encoders")
	encOut, encErr := encCmd.CombinedOutput()
	if encErr == nil && len(encOut) > 0 {
		result.HasEncoders = true
	}

	return result
}

// checkFFprobe 检测 ffprobe 是否可用
func checkFFprobe() FFprobeCheck {
	result := FFprobeCheck{}
	probePath := getFFprobePath()
	result.Path = probePath

	cmd := exec.Command(probePath, "-version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		result.Found = false
		result.Version = fmt.Sprintf("调用失败: %v", err)
		return result
	}
	result.Found = true
	lines := strings.Split(string(output), "\n")
	if len(lines) > 0 {
		result.Version = strings.TrimSpace(lines[0])
	}
	return result
}

// checkGPUAcceleration 检查 GPU 加速可用性
// 返回每个候选编码器的探测详情，并标记实际生效的编码器
func checkGPUAcceleration(ffmpegAvailable bool) GPUCheck {
	result := GPUCheck{}

	if !ffmpegAvailable {
		result.DetectedEncoder = "libx264"
		result.Encoders = []EncoderProbe{}
		return result
	}

	// 候选 GPU 编码器（按优先级）
	candidates := []string{"h264_nvenc", "h264_amf", "h264_qsv"}
	probes := make([]EncoderProbe, 0, len(candidates))

	// 一次性获取编码器列表（避免原 detectGPUEencoders 中的 bug：每个 encoder 都重新调用一次）
	ffmpegPath := getFFmpegPath()
	encodersOutput, encErr := exec.Command(ffmpegPath, "-encoders").CombinedOutput()
	encodersList := ""
	if encErr == nil {
		encodersList = strings.ToLower(string(encodersOutput))
	}

	chosenEncoder := "libx264" // 默认回退 CPU

	for _, enc := range candidates {
		probe := EncoderProbe{Name: enc}
		if encErr != nil || !strings.Contains(encodersList, enc) {
			probe.Available = false
			probe.Usable = false
			probe.Detail = "ffmpeg 未编译此编码器"
			probes = append(probes, probe)
			continue
		}
		probe.Available = true

		// 实际可用性测试：编码1帧64x64黑帧，验证驱动+硬件
		testCmd := exec.Command(ffmpegPath,
			"-f", "lavfi", "-i", "color=black:size=64x64:duration=0.1:rate=1",
			"-c:v", enc, "-f", "null", "-")
		testOutput, testErr := testCmd.CombinedOutput()
		if testErr != nil {
			probe.Usable = false
			// 提取关键错误信息
			errLines := []string{}
			for _, line := range strings.Split(string(testOutput), "\n") {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				if strings.Contains(line, "Error") || strings.Contains(line, "error") ||
					strings.Contains(line, "fail") || strings.Contains(line, "not support") ||
					strings.Contains(line, "minimum") || strings.Contains(line, "required") ||
					strings.Contains(line, "Cannot") || strings.Contains(line, "No capable") {
					errLines = append(errLines, line)
				}
			}
			if len(errLines) > 0 {
				probe.Detail = errLines[0]
			} else {
				probe.Detail = "硬件/驱动不支持"
			}
		} else {
			probe.Usable = true
			probe.Detail = "可用"
			// 选择第一个可用的 GPU 编码器
			if chosenEncoder == "libx264" {
				chosenEncoder = enc
			}
		}
		probes = append(probes, probe)
	}

	result.Encoders = probes
	result.DetectedEncoder = chosenEncoder
	result.IsGPU = chosenEncoder != "libx264"

	// 测试 -hwaccel cuda 是否可用（GPU全流水线关键：解码+编码都在GPU）
	// 用实际合成视频测试，而非空帧
	if result.IsGPU {
		hwTestCmd := exec.Command(ffmpegPath,
			"-hwaccel", "cuda",
			"-f", "lavfi", "-i", "color=black:size=320x240:duration=0.5:rate=25",
			"-c:v", chosenEncoder, "-f", "null", "-")
		hwTestOut, hwTestErr := hwTestCmd.CombinedOutput()
		if hwTestErr != nil {
			result.HWAccelCUDA = false
			_ = hwTestOut
		} else {
			result.HWAccelCUDA = true
		}
	}

	return result
}

// checkMediaDirs 检查所有配置的曲库目录状态（仅检查目录存在性，不遍历文件）
// 文件数量统计由曲库扫描步骤完成后通过 updateSelfCheckMediaStats 填充
func checkMediaDirs() []MediaDirStat {
	stats := make([]MediaDirStat, 0, len(mediaDirs))

	for _, dir := range mediaDirs {
		stat := MediaDirStat{Path: dir}

		info, err := os.Stat(dir)
		if err != nil {
			if os.IsNotExist(err) {
				stat.ErrorReason = "目录不存在"
			} else if os.IsPermission(err) {
				stat.ErrorReason = "无访问权限"
			} else {
				stat.ErrorReason = err.Error()
			}
			stats = append(stats, stat)
			continue
		}
		if !info.IsDir() {
			stat.ErrorReason = "路径不是目录"
			stats = append(stats, stat)
			continue
		}
		stat.Exists = true
		stat.IsReadable = true
		// FileCount/VideoCount/AudioCount 留给曲库扫描步骤填充
		stats = append(stats, stat)
	}

	return stats
}

// collectBlockingIssues 根据自检结果生成阻断性问题列表
func collectBlockingIssues(r *SelfCheckReport) []BlockingIssue {
	issues := []BlockingIssue{}

	// FFmpeg 缺失（严重）
	if !r.FFmpeg.Found {
		issues = append(issues, BlockingIssue{
			Level:      "critical",
			Category:   "ffmpeg",
			Title:      "FFmpeg 未找到",
			Detail:     "转码与流媒体功能无法工作，无法播放非 H264+AAC 格式的歌曲。",
			Action:     "请将 ffmpeg.exe 放到程序同目录，或安装到系统 PATH 环境变量中。",
			ActionURL:  "https://ffmpeg.org/download.html",
			ActionType: "link",
		})
	}

	// FFprobe 缺失（严重）
	if !r.FFprobe.Found {
		issues = append(issues, BlockingIssue{
			Level:      "critical",
			Category:   "ffprobe",
			Title:      "FFprobe 未找到",
			Detail:     "无法获取媒体时长/编码信息，转码决策失效。",
			Action:     "请将 ffprobe.exe 放到程序同目录（通常与 ffmpeg.exe 一起分发），或安装到系统 PATH。",
			ActionURL:  "https://ffmpeg.org/download.html",
			ActionType: "link",
		})
	}

	// 曲库目录问题
	for _, d := range r.MediaDirs {
		if !d.Exists {
			issues = append(issues, BlockingIssue{
				Level:      "critical",
				Category:   "mediaDir",
				Title:      "曲库目录不存在: " + d.Path,
				Detail:     "该路径不存在或无法访问，KTV 无法从中读取歌曲。",
				Action:     "请在设置中修改曲库目录，或在文件管理器中创建该目录后放入歌曲文件。",
				ActionURL:  "/settings",
				ActionType: "settings",
			})
		} else if r.MediaScanDone && d.FileCount == 0 {
			issues = append(issues, BlockingIssue{
				Level:      "warning",
				Category:   "mediaDir",
				Title:      "曲库目录为空: " + d.Path,
				Detail:     "目录存在但未发现任何支持格式的音视频文件（支持 mp3/flac/wav/mp4/mkv/avi 等格式）。",
				Action:     "请将歌曲文件放入该目录，或修改曲库目录指向实际的歌曲存放位置。",
				ActionURL:  "/settings",
				ActionType: "settings",
			})
		}
	}

	// 曲库总数为 0（曲库扫描完成后由 updateSelfCheckMediaStats 更新）
	if r.TotalSongs == 0 {
		// 仅当存在至少一个目录时才追加（避免与上面的"目录不存在"重复）
		hasExistingDir := false
		for _, d := range r.MediaDirs {
			if d.Exists {
				hasExistingDir = true
				break
			}
		}
		if hasExistingDir && r.MediaScanDone {
			issues = append(issues, BlockingIssue{
				Level:      "critical",
				Category:   "mediaDir",
				Title:      "曲库为空（0 首歌曲）",
				Detail:     "所有曲库目录中均未发现支持格式的文件，点歌功能将无内容可用。",
				Action:     "请上传歌曲或修改曲库目录指向实际的歌曲存放位置。",
				ActionURL:  "/upload",
				ActionType: "settings",
			})
		}
	}

	return issues
}

// computeOverallStatus 根据阻断性问题计算整体状态
func computeOverallStatus(issues []BlockingIssue) string {
	hasCritical := false
	hasWarning := false
	for _, iss := range issues {
		if iss.Level == "critical" {
			hasCritical = true
		} else if iss.Level == "warning" {
			hasWarning = true
		}
	}
	if hasCritical {
		return "critical"
	}
	if hasWarning {
		return "warning"
	}
	return "ok"
}

// SelfCheckHandler HTTP 接口：返回自检报告
func SelfCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if selfCheckResult == nil {
		// 兜底：若启动自检未执行，临时执行一次（不含曲库扫描的延迟操作）
		selfCheckResult = runSelfCheck()
	}
	json.NewEncoder(w).Encode(selfCheckResult)
}
