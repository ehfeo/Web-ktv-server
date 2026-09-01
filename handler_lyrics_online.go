package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// 歌词在线搜索：基于开源 lrc_download 方案，
// 多源聚合：api.ygking.top（QQ音乐）、网易云、api.lrc.cx，
// 匹配顺序为 歌名→歌手→时长，尾缀（live/DJ 等）干扰做归一化容错。

const lyricAPIBase = "https://api.ygking.top"

// 搜索结果缓存：同一文件名不重复请求外部接口，避免点歌时频繁拉起搜索。
// 同时记录命中来源，便于前端展示"歌词来源：xxx"。
type lyricCacheEntry struct {
	lyrics string
	source string
}

var lyricOnlineCache struct {
	sync.RWMutex
	cache map[string]lyricCacheEntry // key: songName -> lyrics+source（lyrics 为空表示已尝试过未命中）
}

func lyricGetCached(songName string) (lyricCacheEntry, bool) {
	lyricOnlineCache.RLock()
	defer lyricOnlineCache.RUnlock()
	if lyricOnlineCache.cache == nil {
		return lyricCacheEntry{}, false
	}
	v, ok := lyricOnlineCache.cache[songName]
	return v, ok
}

func lyricSetCache(songName string, e lyricCacheEntry) {
	lyricOnlineCache.Lock()
	defer lyricOnlineCache.Unlock()
	if lyricOnlineCache.cache == nil {
		lyricOnlineCache.cache = make(map[string]lyricCacheEntry)
	}
	lyricOnlineCache.cache[songName] = e
}

type LyricQQSinger struct {
	Name string `json:"name"`
}
type LyricQQSong struct {
	Mid      string            `json:"mid"`
	Name     string            `json:"name"`
	Title    string            `json:"title"`
	Singer   []LyricQQSinger   `json:"singer"`
	SongTime string            `json:"songTime"` // 部分接口返回的时长（毫秒）
	Interval interface{}       `json:"interval"` // 备选时长（秒），数字或字符串
}
type LyricQQSearchResult struct {
	Code int `json:"code"`
	Data struct {
		List []LyricQQSong `json:"list"`
	} `json:"data"`
	Msg string `json:"msg"`
}
type LyricQQLyricResult struct {
	Code int `json:"code"`
	Data struct {
		Lyric string `json:"lyric"`
		Trans string `json:"trans"`
		Roma  string `json:"roma"`
	} `json:"data"`
}

type LyricNeteaseSong struct {
	Id       int    `json:"id"`
	Name     string `json:"name"`
	Duration int    `json:"duration"` // ms
	Artists  []struct {
		Name string `json:"name"`
	} `json:"artists"`
}
type LyricNeteaseSearchResult struct {
	Code   int `json:"code"`
	Result struct {
		Songs []LyricNeteaseSong `json:"songs"`
	} `json:"result"`
}
type LyricNeteaseLyricResult struct {
	Lrc struct {
		Lyric string `json:"lyric"`
	} `json:"lrc"`
}

var lyricHTTPClient = &http.Client{Timeout: 8 * time.Second}

func lyricHTTPGet(rawURL string) (string, error) {
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "KTV-Server/1.0 (lyrics downloader)")
	resp, err := lyricHTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return strings.TrimPrefix(string(body), "\ufeff"), nil
}

// lyricCleanName 归一化：去后缀标签、去括号/方括号内容、小写、压缩空白
func lyricCleanName(name string) string {
	text := strings.ToLower(strings.TrimSpace(strings.TrimSuffix(filepath.Base(name), filepath.Ext(name))))
	reStrip := regexp.MustCompile(`\[[^\]]*\]|\([^)]*\)|（[^）]*）`)
	text = reStrip.ReplaceAllString(text, " ")
	text = regexp.MustCompile(`[-_.,，。·•|/\\]+`).ReplaceAllString(text, " ")
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
	return strings.TrimSpace(text)
}

// lyricParseTrackName 从文件名解析 歌名/歌手。
// 支持 "艺人 - 歌名" 等多种分隔符。
func lyricParseTrackName(path string) (title, artist string) {
	stem := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	stem = regexp.MustCompile(`\[[^\]]*\]|\([^)]*\)|（[^）]*）`).ReplaceAllString(stem, " ")
	stem = regexp.MustCompile(`\s+`).ReplaceAllString(strings.TrimSpace(stem), " ")
	parts := regexp.MustCompile(`\s+-\s+|\s+–\s+|\s+—\s+`).Split(stem, 2)
	if len(parts) == 2 {
		return strings.TrimSpace(parts[1]), strings.TrimSpace(parts[0])
	}
	// 无歌手分隔：可能形如 "06.歌名" / "12 歌名"，剥掉前导数字序号，仅剩歌名
	stemNoNum := regexp.MustCompile(`^\d+\s*[.、．\-‐—]\s*`).ReplaceAllString(stem, "")
	if stemNoNum != "" && stemNoNum != stem {
		stem = stemNoNum
	}
	return strings.TrimSpace(stem), ""
}

// lyricSimilarity 归一化后字符串相似度（0-1）。基于最长公共子序列，容错顺序/尾缀差异。
func lyricSimilarity(a, b string) float64 {
	aRunes, bRunes := []rune(a), []rune(b)
	if len(aRunes) == 0 || len(bRunes) == 0 {
		return 0
	}
	dp := make([]int, len(bRunes)+1)
	for i := 1; i <= len(aRunes); i++ {
		prev := 0
		for j := 1; j <= len(bRunes); j++ {
			temp := dp[j]
			if aRunes[i-1] == bRunes[j-1] {
				dp[j] = prev + 1
			} else if dp[j] < dp[j-1] {
				dp[j] = dp[j-1]
			}
			prev = temp
		}
	}
	return float64(2*dp[len(bRunes)]) / float64(len(aRunes)+len(bRunes))
}

func lyricHasTimestamps(text string) bool {
	return regexp.MustCompile(`\[\d{1,2}:\d{2}(?:[.:]\d{1,3})?\]`).MatchString(text)
}

func lyricSearchQQ(keyword string) ([]LyricQQSong, error) {
	q := url.Values{
		"keyword": {keyword},
		"type":    {"song"},
		"num":     {"20"},
	}
	searchURL := lyricAPIBase + "/api/search?" + q.Encode()
	text, err := lyricHTTPGet(searchURL)
	if err != nil {
		return nil, err
	}
	var result LyricQQSearchResult
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return nil, err
	}
	if result.Code != 0 {
		return nil, fmt.Errorf("QQ API error: code=%d %s", result.Code, result.Msg)
	}
	return result.Data.List, nil
}

// lyricGatherCandidates 按多种关键词组合拉取候选，去重返回。
func lyricGatherCandidates(title, artist string) []LyricQQSong {
	attempts := [][2]string{{title, artist}}
	if artist != "" {
		attempts = append(attempts, [2]string{title, ""}, [2]string{artist, ""})
	} else {
		attempts = append(attempts, [2]string{title, ""})
	}
	seen := map[string]bool{}
	var all []LyricQQSong
	for _, at := range attempts {
		kw := strings.TrimSpace(at[0])
		if kw == "" {
			continue
		}
		songs, err := lyricSearchQQ(kw)
		if err != nil || len(songs) == 0 {
			continue
		}
		for _, s := range songs {
			if s.Mid != "" && !seen[s.Mid] {
				seen[s.Mid] = true
				all = append(all, s)
			}
		}
	}
	return all
}

// lyricQQDuration 提取候选的时长（秒），无法解析返回 0。
func lyricQQDuration(s LyricQQSong) float64 {
	if s.SongTime != "" {
		// songTime 可能为毫秒或秒，按量级判断
		if v, err := strconv.ParseFloat(strings.TrimSpace(s.SongTime), 64); err == nil {
			if v > 1000 { // ms
				return v / 1000
			}
			return v
		}
	}
	if s.Interval != nil {
		switch iv := s.Interval.(type) {
		case float64:
			return iv
		case int:
			return float64(iv)
		case string:
			if v, err := strconv.ParseFloat(strings.TrimSpace(iv), 64); err == nil {
				return v
			}
		}
	}
	return 0
}

// lyricPickBest 按 歌名→歌手→时长 优先级选出最佳候选。
func lyricPickBest(songs []LyricQQSong, title, artist string, duration float64) *LyricQQSong {
	normTitle := lyricCleanName(title)
	var best *LyricQQSong
	bestScore := -1.0
	bestTitleSim := 0.0
	for i := range songs {
		s := &songs[i]
		// 1) 歌名相似度（权重最大）
		titleSim := lyricSimilarity(normTitle, lyricCleanName(s.Name))
		if s.Title != "" {
			ts := lyricSimilarity(normTitle, lyricCleanName(s.Title))
			if ts > titleSim {
				titleSim = ts
			}
		}
		// 2) 歌手相似度
		artistSim := 1.0
		if artist != "" {
			artistSim = 0.0
			normArtist := lyricCleanName(artist)
			for _, sg := range s.Singer {
				for _, part := range strings.FieldsFunc(sg.Name, func(r rune) bool {
					return r == '/' || r == '、' || r == ',' || r == '&' || r == ' '
				}) {
					as := lyricSimilarity(normArtist, lyricCleanName(part))
					if as > artistSim {
						artistSim = as
					}
				}
			}
		}
		// 3) 时长相似度
		durSim := 1.0
		if duration > 0 {
			qd := lyricQQDuration(*s)
			if qd > 0 {
				diff := math.Abs(duration - qd)
				if diff > 30 { // 相差超过30s，视为时长不匹配
					durSim = 0.0
				} else {
					durSim = 1.0 - diff/120.0
				}
			}
		}
		// 优先级：时长匹配 最大 → 歌名 → 歌手
		score := durSim*10000 + titleSim*100 + artistSim
		if score > bestScore {
			bestScore = score
			best = s
			bestTitleSim = titleSim
		}
	}
	// 歌名相似度过低的候选直接舍弃（避免点到毫不相干的歌）
	if best != nil && bestTitleSim < 0.35 {
		return nil
	}
	return best
}

func lyricFetchQQ(mid string) (string, error) {
	q := url.Values{"mid": {mid}}
	lyricURL := lyricAPIBase + "/api/lyric?" + q.Encode()
	text, err := lyricHTTPGet(lyricURL)
	if err != nil {
		return "", err
	}
	var result LyricQQLyricResult
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return "", err
	}
	if result.Code != 0 {
		return "", fmt.Errorf("QQ lyric error: code=%d", result.Code)
	}
	return result.Data.Lyric, nil
}

func lyricSearchNetease(keyword string) ([]LyricNeteaseSong, error) {
	q := url.Values{
		"s":      {keyword},
		"type":   {"1"},
		"offset": {"0"},
		"limit":  {"20"},
	}
	searchURL := "http://music.163.com/api/search/get/web?" + q.Encode()
	text, err := lyricHTTPGet(searchURL)
	if err != nil {
		return nil, err
	}
	var result LyricNeteaseSearchResult
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return nil, err
	}
	if result.Code != 200 {
		return nil, fmt.Errorf("Netease API error: code=%d", result.Code)
	}
	return result.Result.Songs, nil
}

func lyricPickBestNetease(songs []LyricNeteaseSong, title, artist string, duration float64) *LyricNeteaseSong {
	normTitle := lyricCleanName(title)
	var best *LyricNeteaseSong
	bestScore := -1.0
	bestTitleSim := 0.0
	for i := range songs {
		s := &songs[i]
		titleSim := lyricSimilarity(normTitle, lyricCleanName(s.Name))
		artistSim := 1.0
		if artist != "" {
			artistSim = 0.0
			normArtist := lyricCleanName(artist)
			for _, ar := range s.Artists {
				for _, part := range strings.FieldsFunc(ar.Name, func(r rune) bool {
					return r == '/' || r == '、' || r == ',' || r == '&' || r == ' '
				}) {
					as := lyricSimilarity(normArtist, lyricCleanName(part))
					if as > artistSim {
						artistSim = as
					}
				}
			}
		}
		durSim := 1.0
		if duration > 0 && s.Duration > 0 {
			diff := math.Abs(duration - float64(s.Duration)/1000.0)
			if diff > 30 {
				durSim = 0.0
			} else {
				durSim = 1.0 - diff/120.0
			}
		}
		// 优先级：时长匹配 最大 → 歌名 → 歌手
		score := durSim*10000 + titleSim*100 + artistSim
		if score > bestScore {
			bestScore = score
			best = s
			bestTitleSim = titleSim
		}
	}
	if best != nil && bestTitleSim < 0.35 {
		return nil
	}
	return best
}

func lyricFetchNetease(id int) (string, error) {
	lyricURL := fmt.Sprintf("http://music.163.com/api/song/lyric?id=%d&lv=-1&kv=-1&tv=-1", id)
	text, err := lyricHTTPGet(lyricURL)
	if err != nil {
		return "", err
	}
	var result LyricNeteaseLyricResult
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return "", err
	}
	return result.Lrc.Lyric, nil
}

// lyricSearchLrcCx 直接接口（服务端按 title/artist 匹配）
func lyricSearchLrcCx(title, artist string) (string, error) {
	q := url.Values{
		"title":  {title},
		"artist": {artist},
	}
	searchURL := "https://api.lrc.cx/lyrics?" + q.Encode()
	text, err := lyricHTTPGet(searchURL)
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(text, "{") && strings.Contains(text, "\"lyrics\"") {
		var res struct {
			Lyrics string `json:"lyrics"`
		}
		if err := json.Unmarshal([]byte(text), &res); err == nil && res.Lyrics != "" {
			text = res.Lyrics
		}
	}
	if lyricHasTimestamps(text) {
		return text, nil
	}
	return "", nil
}

// lyricSearchLocal 检查同目录同名 .lrc 是否已有内容。
// 返回在线命中的歌词及其来源接口代码（ASCII，如"qq"、"netease"），未命中则返回 ("","")。
func lyricSearchOnline(mediaPath, songName string) (string, string) {
	// 缓存
	if cached, ok := lyricGetCached(songName); ok {
		return cached.lyrics, cached.source
	}

	var lyrics, source string
	title, artist := lyricParseTrackName(mediaPath)
	duration := getDuration(mediaPath) // 0 表示未知

	// 源1：QQ音乐（api.ygking.top）—— 按 时长→歌名→歌手 匹配
	if lyrics == "" {
		candidates := lyricGatherCandidates(title, artist)
		if len(candidates) > 0 {
			if best := lyricPickBest(candidates, title, artist, duration); best != nil {
				if lrc, err := lyricFetchQQ(best.Mid); err == nil && lyricHasTimestamps(lrc) {
					lyrics = lrc
					source = "qq"
				}
			}
		}
	}
	// 源2：网易云音乐
	if lyrics == "" {
		neteaseCandidates := lyricGatherNeteaseCandidates(title, artist)
		if len(neteaseCandidates) > 0 {
			if best := lyricPickBestNetease(neteaseCandidates, title, artist, duration); best != nil {
				if lrc, err := lyricFetchNetease(best.Id); err == nil && lyricHasTimestamps(lrc) {
					lyrics = lrc
					source = "netease"
				}
			}
		}
	}
	// 源3：LRCLIB
	if lyrics == "" {
		if lc := lyricSearchLrclib(title, artist, duration); lc != "" {
			lyrics = lc
			source = "lrclib"
		}
	}
	// 源4：酷狗
	if lyrics == "" {
		if lc := lyricSearchKugou(title, artist, duration); lc != "" {
			lyrics = lc
			source = "kugou"
		}
	}
	// 源5：酷我
	if lyrics == "" {
		if lc := lyricSearchKuwo(title, artist, duration); lc != "" {
			lyrics = lc
			source = "kuwo"
		}
	}
	// 源6：咪咕
	if lyrics == "" {
		if lc := lyricSearchMigu(title, artist, duration); lc != "" {
			lyrics = lc
			source = "migu"
		}
	}
	// 源7：api.lrc.cx（服务端直接匹配）
	if lyrics == "" {
		if artist != "" {
			if lrc, err := lyricSearchLrcCx(title, artist); err == nil && lrc != "" {
				lyrics = lrc
				source = "lrccx"
			}
		}
		if lyrics == "" {
			if lrc, err := lyricSearchLrcCx(title, ""); err == nil && lrc != "" {
				lyrics = lrc
				source = "lrccx"
			}
		}
	}

	lyricSetCache(songName, lyricCacheEntry{lyrics: lyrics, source: source})
	return lyrics, source
}

// lyricGatherNeteaseCandidates 按多种关键词组合拉取网易云候选，去重返回。
func lyricGatherNeteaseCandidates(title, artist string) []LyricNeteaseSong {
	attempts := []string{title}
	if artist != "" {
		attempts = append(attempts, title+" "+artist)
	}
	seen := map[int]bool{}
	var all []LyricNeteaseSong
	for _, kw := range attempts {
		kw = strings.TrimSpace(kw)
		if kw == "" {
			continue
		}
		songs, err := lyricSearchNetease(kw)
		if err != nil || len(songs) == 0 {
			continue
		}
		for _, s := range songs {
			if s.Id != 0 && !seen[s.Id] {
				seen[s.Id] = true
				all = append(all, s)
			}
		}
	}
	return all
}