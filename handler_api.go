package main

import (
	"encoding/json"
	"math/rand"
	"io/ioutil"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

func SongListHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	pageStr := r.URL.Query().Get("page")
	pageSizeStr := r.URL.Query().Get("pageSize")
	keyword := r.URL.Query().Get("keyword")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 {
		pageSize = 24
	}

	songs, total := getMediaListPaged(page, pageSize, keyword)

	result := struct {
		Songs      []MediaFile `json:"songs"`
		Total      int         `json:"total"`
		Page       int         `json:"page"`
		PageSize   int         `json:"pageSize"`
		TotalPages int         `json:"totalPages"`
	}{
		Songs:      songs,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: (total + pageSize - 1) / pageSize,
	}

	json.NewEncoder(w).Encode(result)

	if total == 0 && keyword != "" {
		logZeroResultKeyword(keyword)
	}
}

func ConfigHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	if r.Method == http.MethodGet {
		result := struct {
			MediaDirs    []string `json:"mediaDirs"`
			Port         string   `json:"port"`
			QREnabled    bool     `json:"qrEnabled"`
			QRServerAddr string   `json:"qrServerAddr"`
			QRPassword   string   `json:"qrPassword"`
			QRMode        string   `json:"qrMode"`
			QRCtrlEnabled bool     `json:"qrCtrlEnabled"`
			AudioTranscodeBitrate int `json:"audioTranscodeBitrate"`
		}{
			MediaDirs:    mediaDirs,
			Port:         port,
			QREnabled:    qrEnabled,
			QRServerAddr: qrServerAddr,
			QRPassword:   qrPassword,
			QRMode:       qrMode,
			QRCtrlEnabled: qrCtrlEnabled,
			AudioTranscodeBitrate: audioTranscodeBitrate,
		}
		json.NewEncoder(w).Encode(result)
		return
	}

	if r.Method == http.MethodPost {
		data, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "读取数据失败", 400)
			return
		}

		var config struct {
			MediaDirs    []string `json:"mediaDirs"`
			Port         string   `json:"port"`
			QREnabled    bool     `json:"qrEnabled"`
			QRServerAddr string   `json:"qrServerAddr"`
			QRPassword   string   `json:"qrPassword"`
			QRMode        string   `json:"qrMode"`
			QRCtrlEnabled bool     `json:"qrCtrlEnabled"`
			AudioTranscodeBitrate int `json:"audioTranscodeBitrate"`
		}

		if err := json.Unmarshal(data, &config); err != nil {
			http.Error(w, "解析数据失败", 400)
			return
		}

		if len(config.MediaDirs) > 0 {
			mediaDirs = config.MediaDirs
		}
		if config.Port != "" {
			port = config.Port
		}
		qrEnabled = config.QREnabled
		qrServerAddr = config.QRServerAddr
		qrPassword = config.QRPassword
		qrCtrlEnabled = config.QRCtrlEnabled
		if config.AudioTranscodeBitrate >= 32 && config.AudioTranscodeBitrate <= 512 {
			audioTranscodeBitrate = config.AudioTranscodeBitrate
		}
		if config.QRMode == "internal" {
			qrMode = "internal"
		} else if config.QRMode == "external" {
			qrMode = "external"
		} else if qrMode == "" {
			qrMode = "external"
		}

		saveConfig()
		saveQRConfig()

		// 遥控权限可能已变化，向所有已连接手机推送最新状态
		broadcastQRControlState()

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
		return
	}

	http.Error(w, "不支持的方法", 405)
}

// RandomSongHandler 随机返回一首歌曲
func RandomSongHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	list := getCachedMediaList()
	if len(list) == 0 {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "曲库为空"})
		return
	}
	song := list[rand.Intn(len(list))]
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"name":    song.Name,
		"path":    song.Path,
		"type":    song.Type,
	})
}

// singerFirstChar 获取歌手名的首字母分类键
func singerFirstChar(name string) string {
	if len(name) == 0 {
		return "#"
	}
	r := []rune(name)[0]
	// 英文字母
	if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
		return strings.ToUpper(string(r))
	}
	// 中文字符用拼音首字母
	if unicode.Is(unicode.Han, r) {
		return pinyinInitial(r)
	}
	// 其他字符
	return "#"
}

// pinyinInitial 返回汉字的拼音首字母
// 通过将 Unicode 转 GB2312 编码，利用 GB2312 level1 按拼音排序的特性
func pinyinInitial(r rune) string {
	b := string(r)
	gb, err := utf8ToGBK(b)
	if err != nil || len(gb) != 2 {
		return "#"
	}
	hi := int(gb[0])
	lo := int(gb[1])
	if hi < 0xB0 || hi > 0xF7 || lo < 0xA1 || lo > 0xFE {
		return "#"
	}

	// GB2312 顺序位置 (0 开始)
	pos := (hi-0xB0)*94 + (lo - 0xA1)

	// GB2312 拼音首字母起始位置表（基于 level1 实测数据）
	// 每个 entry 表示该拼音首字母从哪个位置开始
	// 对于 level2 字符（pos >= 3755），映射不可靠，回退到 "#"
	if pos >= 3755 {
		return "#"
	}

	pinyinStarts := []struct {
		letter byte
		start  int
	}{
		{'A', 0},    // 啊
		{'B', 40},   // 吧/八
		{'C', 220},  // 擦
		{'D', 453},  // 搭
		{'E', 637},  // 蛾
		{'F', 659},  // 发
		{'G', 784},  // 噶
		{'H', 939},  // 哈（蛤pos=833是GB2312排序异常，排在G区）
		{'J', 1120}, // 击
		{'K', 1415}, // 喀
		{'L', 1515}, // 垃
		{'M', 1763}, // 妈
		{'N', 1914}, // 拿
		{'O', 1995}, // 哦（噢在level2）
		{'P', 2003}, // 啪
		{'Q', 2125}, // 期
		{'R', 2282}, // 然
		{'S', 2341}, // 撒（仨在level2）
		{'T', 2628}, // 他
		{'W', 2783}, // 挖
		{'X', 2906}, // 西
		{'Y', 3126}, // 压
		{'Z', 3432}, // 匝
	}

	// 从后往前找，找到第一个 start <= pos 的字母
	for i := len(pinyinStarts) - 1; i >= 0; i-- {
		if pos >= pinyinStarts[i].start {
			return string(pinyinStarts[i].letter)
		}
	}
	return "#"
}

// utf8ToGBK 将 UTF-8 字符串转换为 GBK 编码
func utf8ToGBK(s string) ([]byte, error) {
	reader := transform.NewReader(strings.NewReader(s), simplifiedchinese.GBK.NewEncoder())
	result, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// computePinyinInitials 计算字符串中所有汉字的拼音首字母序列
// 例如："张国荣-风继续吹" → "ZGRFJXC"
func computePinyinInitials(s string) string {
	var buf strings.Builder
	for _, r := range s {
		if r >= 'a' && r <= 'z' {
			buf.WriteByte(byte(r - 'a' + 'A'))
		} else if r >= 'A' && r <= 'Z' {
			buf.WriteByte(byte(r))
		} else if unicode.Is(unicode.Han, r) {
			buf.WriteString(pinyinInitial(r))
		}
		// 跳过其他字符（数字、标点、空格等）
	}
	return buf.String()
}

// SingerIndexItem 歌手索引条目
type SingerIndexItem struct {
	Letter string `json:"letter"`
	Name   string `json:"name"`
	Count  int    `json:"count"`
}

// SingerIndexHandler 返回歌手首字母索引
func SingerIndexHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	list := getCachedMediaList()

	// 统计每个歌手的歌曲数量
	singerCount := make(map[string]int)
	for _, item := range list {
		if item.Singer != "" && item.Singer != "未知歌手" {
			singerCount[item.Singer]++
		}
	}

	// 按首字母分组
	letterMap := make(map[string][]SingerIndexItem)
	for singer, count := range singerCount {
		letter := singerFirstChar(singer)
		letterMap[letter] = append(letterMap[letter], SingerIndexItem{
			Letter: letter,
			Name:   singer,
			Count:  count,
		})
	}

	// 每个字母内按歌曲数量降序排列（数量相同则按歌手名排序）
	for letter := range letterMap {
		sort.Slice(letterMap[letter], func(i, j int) bool {
			if letterMap[letter][i].Count != letterMap[letter][j].Count {
				return letterMap[letter][i].Count > letterMap[letter][j].Count
			}
			return letterMap[letter][i].Name < letterMap[letter][j].Name
		})
	}

	// 构建有序结果
	letters := make([]string, 0, len(letterMap))
	for l := range letterMap {
		letters = append(letters, l)
	}
	sort.Strings(letters)

	result := make(map[string][]SingerIndexItem)
	for _, l := range letters {
		result[l] = letterMap[l]
	}

	json.NewEncoder(w).Encode(result)
}

// SongsBySingerHandler 返回指定歌手的歌曲列表
func SongsBySingerHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	singer := r.URL.Query().Get("singer")
	if singer == "" {
		json.NewEncoder(w).Encode([]MediaFile{})
		return
	}

	list := getCachedMediaList()
	var songs []MediaFile
	for _, item := range list {
		if item.Singer == singer {
			songs = append(songs, item)
		}
	}

	if songs == nil {
		songs = []MediaFile{}
	}
	json.NewEncoder(w).Encode(songs)
}

// CategoryItem 分类条目
type CategoryItem struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// LanguageIndexHandler 返回语种分类列表
func LanguageIndexHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	list := getCachedMediaList()

	langCount := make(map[string]int)
	for _, item := range list {
		lang := item.Language
		if lang == "" {
			lang = "未知"
		}
		langCount[lang]++
	}

	var result []CategoryItem
	for lang, count := range langCount {
		result = append(result, CategoryItem{Name: lang, Count: count})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Count > result[j].Count
	})

	if result == nil {
		result = []CategoryItem{}
	}
	json.NewEncoder(w).Encode(result)
}

// CategoryIndexHandler 返回曲种分类列表
func CategoryIndexHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	list := getCachedMediaList()

	catCount := make(map[string]int)
	for _, item := range list {
		cat := item.Category
		if cat == "" {
			cat = "未知"
		}
		catCount[cat]++
	}

	var result []CategoryItem
	for cat, count := range catCount {
		result = append(result, CategoryItem{Name: cat, Count: count})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Count > result[j].Count
	})

	if result == nil {
		result = []CategoryItem{}
	}
	json.NewEncoder(w).Encode(result)
}

// SongsByLanguageHandler 返回指定语种的歌曲列表
func SongsByLanguageHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	lang := r.URL.Query().Get("language")
	if lang == "" {
		json.NewEncoder(w).Encode([]MediaFile{})
		return
	}

	list := getCachedMediaList()
	var songs []MediaFile
	for _, item := range list {
		itemLang := item.Language
		if itemLang == "" {
			itemLang = "未知"
		}
		if itemLang == lang {
			songs = append(songs, item)
		}
	}

	if songs == nil {
		songs = []MediaFile{}
	}
	json.NewEncoder(w).Encode(songs)
}

// SongsByCategoryHandler 返回指定曲种的歌曲列表
func SongsByCategoryHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	cat := r.URL.Query().Get("category")
	if cat == "" {
		json.NewEncoder(w).Encode([]MediaFile{})
		return
	}

	list := getCachedMediaList()
	var songs []MediaFile
	for _, item := range list {
		itemCat := item.Category
		if itemCat == "" {
			itemCat = "未知"
		}
		if itemCat == cat {
			songs = append(songs, item)
		}
	}

	if songs == nil {
		songs = []MediaFile{}
	}
	json.NewEncoder(w).Encode(songs)
}