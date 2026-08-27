package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
)

var (
	mediaDirs      []string
	port           string
	server         *http.Server
	serverRunning  bool
	videoEncoder   string
	audioEncoder   string
)

const configFile = "ktv_config.json"

type Config struct {
	MediaDirs []string `json:"mediaDirs"`
	Port      string   `json:"port"`
}

func loadConfig() {
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		mediaDirs = []string{"D:\\CloudMusic"}
		port = "82"
		saveConfig()
		return
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		fmt.Println("配置文件格式错误，使用默认配置:", err)
		mediaDirs = []string{"D:\\CloudMusic"}
		port = "82"
		saveConfig()
		return
	}

	if len(config.MediaDirs) == 0 {
		mediaDirs = []string{"D:\\CloudMusic"}
	} else {
		mediaDirs = config.MediaDirs
	}
	
	if config.Port == "" {
		port = "82"
	} else {
		port = config.Port
	}
	
	fmt.Printf("加载配置: mediaDirs=%v, port=%s\n", mediaDirs, port)
}

func saveConfig() {
	config := Config{
		MediaDirs: mediaDirs,
		Port:      port,
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		log.Println("保存配置失败:", err)
		return
	}

	if err := ioutil.WriteFile(configFile, data, 0644); err != nil {
		log.Println("保存配置文件失败:", err)
	}
}

func startServer() error {
	// ===== 启动自检（FFmpeg/FFprobe/GPU/曲库目录）=====
	// 自检不阻断启动：即使存在问题也继续启动，让用户在前端看到完整诊断信息
	fmt.Println("=======================================")
	fmt.Println("正在执行系统自检...")
	fmt.Println("=======================================")
	selfCheckResult = runSelfCheck()
	printSelfCheckSummary(selfCheckResult)

	// 用自检结果填充 videoEncoder/audioEncoder（替代原 detectGPUEencoders）
	videoEncoder = selfCheckResult.GPU.DetectedEncoder
	audioEncoder = "aac"
	fmt.Printf("视频编码器: %s\n", videoEncoder)
	fmt.Printf("音频编码器: %s\n", audioEncoder)

	// 媒体目录检查：仅警告，不阻止启动（让用户在前端看到诊断）
	for _, dir := range mediaDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			fmt.Printf("[警告] 媒体目录不存在: %s（将跳过此目录）\n", dir)
		}
	}

	// 检查是否有缓存文件
	useCache := false
	if hasMediaCache() {
		fmt.Println("\n检测到上次保存的曲库缓存数据！")
		fmt.Println("1. 读取缓存（快速启动）")
		fmt.Println("2. 重新扫描曲库")
		fmt.Print("请选择: ")
		choice := getUserInput()
		if choice == "1" {
			if loadMediaCache() {
				useCache = true
			} else {
				fmt.Println("缓存加载失败，将重新扫描曲库...")
			}
		}
	}

	if !useCache {
		// 显示扫描进度
		fmt.Println("正在扫描曲库...")
		fmt.Println("  第1步: 统计文件数量...")
		getMediaListWithProgress(func(current, total int, fileName string) {
			if current == 1 {
				fmt.Printf("  第2步: 读取文件信息 (共%d个文件)...\n", total)
			}
			progress := float64(current) / float64(total) * 100
			fmt.Printf("\r  扫描进度: %.1f%% (%d/%d)", progress, current, total)
		})
		fmt.Print("\r")
		fmt.Println("曲库扫描完成！")

		initMediaCache()
	}

	// 曲库扫描完成后，更新自检报告中的歌曲总数（曲库为空的关键状态）
	updateSelfCheckMediaStats(getCachedMediaList())

	// 初始化QR配置并注册路由
	initQRConfig()
	registerQRHandlers()
	qrSetupMode() // 按模式分发回复并注册内置/外接路由
	go startQRClient() // 仅外接模式启动客户端连接

	http.HandleFunc("/", IndexHandler)
		http.HandleFunc("/player", PlayerHandler)
		http.HandleFunc("/audio-player", AudioPlayerHandler)
		http.HandleFunc("/settings", SettingsHandler)
		http.HandleFunc("/upload", UploadPageHandler)
		http.HandleFunc("/api/upload", UploadAPIHandler)
		http.HandleFunc("/missing", MissingPageHandler)
		http.HandleFunc("/api/missing/submit", MissingSubmitHandler)
		http.HandleFunc("/api/missing/list", MissingListHandler)
		http.HandleFunc("/m", MobilePageHandler)
		http.HandleFunc("/file", FileHandler)
		http.HandleFunc("/api/songs", SongListHandler)
		http.HandleFunc("/api/songs/search", SongSearchHandler)
		http.HandleFunc("/api/config", ConfigHandler)
		http.HandleFunc("/api/transcode", TranscodeHandler)
		http.HandleFunc("/api/transcode/progress", TranscodeProgressHandler)
		http.HandleFunc("/api/transcode/status", TranscodeStatusHandler)
		http.HandleFunc("/api/transcode/check-and-add", CheckAndAddTranscodeHandler)
		http.HandleFunc("/api/disk-status", DiskSleepStatusHandler)
		http.HandleFunc("/api/track-switch", TrackSwitchHandler)
		http.HandleFunc("/api/check-transcode", CheckTranscodeHandler)
		http.HandleFunc("/api/stream", StreamHandler)
		http.HandleFunc("/api/media-duration", MediaDurationHandler)
		http.HandleFunc("/api/random-song", RandomSongHandler)
		http.HandleFunc("/api/lyrics", LyricsHandler)
		http.HandleFunc("/api/singers", SingerIndexHandler)
		http.HandleFunc("/api/songs-by-singer", SongsBySingerHandler)
		http.HandleFunc("/api/languages", LanguageIndexHandler)
		http.HandleFunc("/api/categories", CategoryIndexHandler)
		http.HandleFunc("/api/songs-by-language", SongsByLanguageHandler)
		http.HandleFunc("/api/songs-by-category", SongsByCategoryHandler)
		http.HandleFunc("/api/selfcheck", SelfCheckHandler)
		http.HandleFunc("/api/check-tracks", CheckTracksHandler)
		http.HandleFunc("/api/hot-songs", HotSongsHandler)
		http.HandleFunc("/api/play-count", PlayCountHandler)
		http.HandleFunc("/api/increment-play", IncrementPlayHandler)
	http.HandleFunc("/api/music-mode-stream", MusicModeStreamHandler)

	server = &http.Server{
		Addr: ":" + port,
	}

	go func() {
		log.Printf("🎤 KTV双屏点歌机启动成功！")
		log.Printf("📂 媒体目录：%v", mediaDirs)
		
		// 获取所有网络适配器的IP地址
		ips := getLocalIPs()
		log.Println("🌐 访问地址：")
		for _, ip := range ips {
			log.Printf("  - http://%s:%s", ip, port)
		}
		log.Printf("  - http://localhost:%s", port)

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("服务器启动失败: %v", err)
		}
	}()

	serverRunning = true
	return nil
}

func stopServer() error {
	if server != nil {
		if err := server.Close(); err != nil {
			return err
		}
	}

	serverRunning = false
	log.Println("🎤 KTV双屏点歌机已停止")
	return nil
}

func main() {
	loadConfig()
	LoadHotPlayData()

	for {
		showMenu()
		choice := getUserInput()

		switch choice {
		case "1":
			if !serverRunning {
				if err := startServer(); err != nil {
					fmt.Printf("启动失败: %v\n", err)
					fmt.Println("按任意键继续...")
					getUserInput()
				} else {
					fmt.Println("服务器启动成功！")
					fmt.Println("按任意键继续...")
					getUserInput()
				}
			} else {
				fmt.Println("服务器已经在运行中！")
				fmt.Println("按任意键继续...")
				getUserInput()
			}
		case "2":
			if serverRunning {
				if err := stopServer(); err != nil {
					fmt.Printf("停止失败: %v\n", err)
					fmt.Println("按任意键继续...")
					getUserInput()
				} else {
					fmt.Println("服务器已停止！")
					fmt.Println("按任意键继续...")
					getUserInput()
				}
			} else {
				fmt.Println("服务器未运行！")
				fmt.Println("按任意键继续...")
				getUserInput()
			}
		case "3":
			fmt.Println("当前曲库文件夹:", mediaDirs)
			fmt.Println("请输入新的曲库文件夹路径（多个路径用逗号分隔）: ")
			newDirs := getUserInput()
			if newDirs != "" {
				dirs := strings.Split(newDirs, ",")
				var cleanedDirs []string
				for _, dir := range dirs {
					dir = strings.TrimSpace(dir)
					if dir != "" {
						cleanedDirs = append(cleanedDirs, dir)
					}
				}
				if len(cleanedDirs) > 0 {
					mediaDirs = cleanedDirs
					saveConfig()
					fmt.Println("曲库文件夹已更新！")
				} else {
					fmt.Println("未做更改")
				}
			} else {
				fmt.Println("未做更改")
			}
			fmt.Println("按任意键继续...")
			getUserInput()
		case "4":
			fmt.Println("当前服务器端口:", port)
			fmt.Print("请输入新的服务器端口: ")
			newPort := getUserInput()
			if newPort != "" {
				port = newPort
				saveConfig()
				fmt.Println("服务器端口已更新！")
			} else {
				fmt.Println("未做更改")
			}
			fmt.Println("按任意键继续...")
			getUserInput()
		case "5":
			fmt.Println("当前曲库文件夹:", mediaDirs)
			fmt.Print("请输入要添加的曲库文件夹路径: ")
			newDir := getUserInput()
			if newDir != "" {
				newDir = strings.TrimSpace(newDir)
				for _, dir := range mediaDirs {
					if dir == newDir {
						fmt.Println("该文件夹已在曲库列表中！")
						fmt.Println("按任意键继续...")
						getUserInput()
						continue
					}
				}
				mediaDirs = append(mediaDirs, newDir)
				saveConfig()
				fmt.Println("曲库文件夹已添加！")
			} else {
				fmt.Println("未做更改")
			}
			fmt.Println("按任意键继续...")
			getUserInput()
		case "6":
			if serverRunning {
				stopServer()
			}
			SaveHotPlayData()
			fmt.Println("退出程序...")
			return
		default:
			fmt.Println("无效选择，请重新输入！")
			fmt.Println("按任意键继续...")
			getUserInput()
		}
	}
}

func showMenu() {
	fmt.Println("=======================================")
	fmt.Println("        KTV双屏点歌机")
	fmt.Println("=======================================")
	fmt.Println("当前配置:")
	fmt.Println("曲库文件夹: ", mediaDirs)
	fmt.Println("服务器端口: ", port)
	fmt.Println("访问地址: http://localhost:", port)
	fmt.Println("=======================================")
	fmt.Println("1. 启动服务器")
	fmt.Println("2. 停止服务器")
	fmt.Println("3. 修改曲库文件夹")
	fmt.Println("4. 修改服务器端口")
	fmt.Println("5. 添加曲库文件夹")
	fmt.Println("6. 退出")
	fmt.Println("=======================================")
	fmt.Print("请选择操作: ")
}

func getUserInput() string {
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

// 获取所有本地IPv4地址（不包括回环地址）
func getLocalIPs() []string {
	var ips []string
	ifaces, err := net.Interfaces()
	if err != nil {
		return ips
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // 跳过未启用的接口
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // 跳过回环接口
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue // 不是IPv4地址
			}
			ips = append(ips, ip.String())
		}
	}
	return ips
}

// detectGPUEencoders 兼容保留：用自检模块的结果填充全局 videoEncoder/audioEncoder
// 注意：原实现存在 bug（每个 encoder 都重新调用一次 ffmpeg -encoders），现统一由 selfcheck.go 完成
func detectGPUEencoders() {
	if selfCheckResult == nil {
		selfCheckResult = runSelfCheck()
	}
	videoEncoder = selfCheckResult.GPU.DetectedEncoder
	audioEncoder = "aac"
}

// printSelfCheckSummary 在控制台打印自检结果摘要
func printSelfCheckSummary(r *SelfCheckReport) {
	fmt.Printf("架构: %s\n", r.Arch)

	// FFmpeg
	if r.FFmpeg.Found {
		fmt.Printf("[✓] FFmpeg: %s\n", r.FFmpeg.Version)
	} else {
		fmt.Printf("[✗] FFmpeg 未找到（路径: %s）\n", r.FFmpeg.Path)
	}

	// FFprobe
	if r.FFprobe.Found {
		fmt.Printf("[✓] FFprobe: %s\n", r.FFprobe.Version)
	} else {
		fmt.Printf("[✗] FFprobe 未找到（路径: %s）\n", r.FFprobe.Path)
	}

	// GPU
	if r.GPU.IsGPU {
		hwAccelMark := "✓"
		hwAccelDetail := "GPU全流水线(解码+编码)可用"
		if !r.GPU.HWAccelCUDA {
			hwAccelMark = "i"
			hwAccelDetail = "仅GPU编码(解码用CPU)，-hwaccel cuda不可用"
		}
		fmt.Printf("[✓] GPU加速: %s\n", r.GPU.DetectedEncoder)
		fmt.Printf("    [%s] %s\n", hwAccelMark, hwAccelDetail)
	} else {
		fmt.Println("[i] GPU加速: 不可用，使用CPU编码 (libx264)")
	}
	for _, p := range r.GPU.Encoders {
		mark := "✗"
		if p.Usable {
			mark = "✓"
		}
		fmt.Printf("    %s %s: %s\n", mark, p.Name, p.Detail)
	}

	// 曲库目录
	fmt.Printf("曲库目录 (%d 个):\n", len(r.MediaDirs))
	for _, d := range r.MediaDirs {
		if d.Exists {
			if r.MediaScanDone {
				fmt.Printf("  [✓] %s (文件: %d, 视频: %d, 音频: %d)\n",
					d.Path, d.FileCount, d.VideoCount, d.AudioCount)
			} else {
				fmt.Printf("  [✓] %s (文件数量待曲库扫描后统计)\n", d.Path)
			}
		} else {
			fmt.Printf("  [✗] %s (%s)\n", d.Path, d.ErrorReason)
		}
	}
	if r.MediaScanDone {
		fmt.Printf("曲库总文件数: %d\n", r.TotalSongs)
	} else {
		fmt.Println("曲库总文件数: 待曲库扫描后统计")
	}

	// 阻断性问题
	if len(r.BlockingIssues) == 0 {
		fmt.Println("自检状态: 全部正常")
	} else {
		fmt.Printf("自检状态: %s（共 %d 项需处理）\n", r.OverallStatus, len(r.BlockingIssues))
		for _, iss := range r.BlockingIssues {
			fmt.Printf("  [%s] %s\n        %s\n", iss.Level, iss.Title, iss.Action)
		}
	}
	fmt.Println("=======================================")
}