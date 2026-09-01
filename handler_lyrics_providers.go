package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// 额外歌词源（来自 macOS 歌词工具 Lyrics-Plus / LyricsX 实装方案）：
// 酷狗 / 酷我 / 咪咕 / LRCLIB。均可在服务端免密钥调用。
// 复用 handler_lyrics_online.go 中的 lyricCleanName / lyricSimilarity / getDuration 等。

// lyricScoreCandidate 统一的 歌名→歌手→时长 打分。
func lyricScoreCandidate(candTitle, candArtist string, candDurSec float64, title, artist string, duration float64) float64 {
	titleSim := lyricSimilarity(lyricCleanName(title), lyricCleanName(candTitle))
	artistSim := 1.0
	if artist != "" {
		artistSim = 0.0
		normArtist := lyricCleanName(artist)
		for _, part := range strings.FieldsFunc(candArtist, func(r rune) bool {
			return r == '/' || r == '、' || r == ',' || r == '&' || r == ' '
		}) {
			if as := lyricSimilarity(normArtist, lyricCleanName(part)); as > artistSim {
				artistSim = as
			}
		}
	}
	durSim := 1.0
	if duration > 0 && candDurSec > 0 {
		diff := duration - candDurSec
		if diff < 0 {
			diff = -diff
		}
		if diff > 30 {
			durSim = 0.0
		} else {
			durSim = 1.0 - diff/120.0
		}
	}
	return titleSim*10000 + artistSim*100 + durSim
}

// lyricHTTPGetWithHeaders 带自定义请求头的 GET，用于酷狗/酷我等需要 Referer 的接口。
func lyricHTTPGetWithHeaders(rawURL string, headers map[string]string) (string, http.Header, error) {
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return "", nil, err
	}
	req.Header.Set("User-Agent", "KTV-Server/1.0 (lyrics downloader)")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := lyricHTTPClient.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	body := make([]byte, 0, 32768)
	buf := make([]byte, 4096)
	for {
		n, errRead := resp.Body.Read(buf)
		if n > 0 {
			body = append(body, buf[:n]...)
		}
		if errRead != nil {
			break
		}
	}
	return strings.TrimPrefix(string(body), "\ufeff"), resp.Header, nil
}

// ---------------- 酷狗 Kugou ----------------
type kugouSong struct {
	FileHash   string `json:"FileHash"`
	SongName   string `json:"SongName"`
	SingerName string `json:"SingerName"`
	AlbumName  string `json:"AlbumName"`
	Duration   *uint  `json:"Duration"`
	MixSongID  string `json:"MixSongID"`
}

func lyricSearchKugou(title, artist string, duration float64) string {

	q := url.Values{
		"keyword":  {strings.TrimSpace(title + " " + artist)},
		"page":     {"1"},
		"pagesize": {"10"},
		"platform": {"WebFilter"},
	}
	searchURL := "https://songsearch.kugou.com/song_search_v2?" + q.Encode()
	text, _, err := lyricHTTPGetWithHeaders(searchURL, map[string]string{"Referer": "https://www.kugou.com/"})
	if err != nil {
		return ""
	}
	var env struct {
		Data struct {
			Lists []kugouSong `json:"lists"`
		} `json:"data"`
	}
	if json.Unmarshal([]byte(text), &env) != nil || len(env.Data.Lists) == 0 {
		return ""
	}
	best := &env.Data.Lists[0]
	bestScore := -1.0
	for i := range env.Data.Lists {
		s := &env.Data.Lists[i]
		var durSec float64
		if s.Duration != nil {
			durSec = float64(*s.Duration)
		}
		sc := lyricScoreCandidate(s.SongName, s.SingerName, durSec, title, artist, duration)
		if sc > bestScore {
			bestScore = sc
			best = s
		}
	}
	if lyricSimilarity(lyricCleanName(title), lyricCleanName(best.SongName)) < 0.35 {
		return ""
	}

	var durMS int64
	if best.Duration != nil {
		durMS = int64(*best.Duration) * 1000
	}
	// 2) 查找歌词候选
	lq := url.Values{
		"ver":    {"1"},
		"man":    {"yes"},
		"client": {"pc"},
		"hash":   {best.FileHash},
	}
	if durMS > 0 {
		lq.Set("duration", fmt.Sprintf("%d", durMS))
	}
	if best.MixSongID != "" {
		lq.Set("album_audio_id", best.MixSongID)
	}
	lcText, _, err := lyricHTTPGetWithHeaders("https://lyrics.kugou.com/search?"+lq.Encode(), map[string]string{"Referer": "https://www.kugou.com/"})
	if err != nil {
		return ""
	}
	var lcEnv struct {
		Candidates []struct {
			ID        string `json:"id"`
			AccessKey string `json:"accesskey"`
		} `json:"candidates"`
	}
	if json.Unmarshal([]byte(lcText), &lcEnv) != nil || len(lcEnv.Candidates) == 0 {
		return ""
	}
	cand := lcEnv.Candidates[0]
	// 3) 下载
	dq := url.Values{
		"ver":       {"1"},
		"client":    {"pc"},
		"id":        {cand.ID},
		"fmt":       {"lrc"},
		"charset":   {"utf8"},
		"accesskey": {cand.AccessKey},
	}
	dlText, _, err := lyricHTTPGetWithHeaders("https://lyrics.kugou.com/download?"+dq.Encode(), map[string]string{"Referer": "https://www.kugou.com/"})
	if err != nil {
		return ""
	}
	var dlEnv struct {
		Content string `json:"content"`
	}
	if json.Unmarshal([]byte(dlText), &dlEnv) != nil || dlEnv.Content == "" {
		return ""
	}
	raw, errByte := base64.StdEncoding.DecodeString(dlEnv.Content)
	if errByte != nil {
		return ""
	}
	lrc := strings.TrimSpace(string(raw))
	if lyricHasTimestamps(lrc) {
		return lrc
	}
	return ""
}

// ---------------- 酷我 Kuwo ----------------
type kuwoSong struct {
	MusicRID string `json:"MUSICRID"`
	SongName string `json:"SONGNAME"`
	Artist   string `json:"ARTIST"`
	Album    string `json:"ALBUM"`
	Duration string `json:"DURATION"`
}

func lyricSearchKuwo(title, artist string, duration float64) string {

	q := url.Values{
		"all":      {strings.TrimSpace(title + " " + artist)},
		"ft":       {"music"},
		"itemset":  {"web_2013"},
		"client":   {"kt"},
		"pn":       {"0"},
		"rn":       {"10"},
		"rformat":  {"json"},
		"encoding": {"utf8"},
		"pcjson":   {"1"},
	}
	searchURL := "https://search.kuwo.cn/r.s?" + q.Encode()
	text, _, err := lyricHTTPGetWithHeaders(searchURL, map[string]string{"Referer": "https://www.kuwo.cn/"})
	if err != nil {
		return ""
	}
	var env struct {
		Abslist []kuwoSong `json:"abslist"`
	}
	if json.Unmarshal([]byte(text), &env) != nil || len(env.Abslist) == 0 {
		return ""
	}
	best := &env.Abslist[0]
	bestScore := -1.0
	for i := range env.Abslist {
		s := &env.Abslist[i]
		var durSec float64
		if s.Duration != "" {
			fmt.Sscanf(s.Duration, "%f", &durSec)
		}
		sc := lyricScoreCandidate(s.SongName, s.Artist, durSec, title, artist, duration)
		if sc > bestScore {
			bestScore = sc
			best = s
		}
	}
	if lyricSimilarity(lyricCleanName(title), lyricCleanName(best.SongName)) < 0.35 {
		return ""
	}
	// MUSIC_RID 形如 MUSIC_51685512，取下划线后的数字为 musicId
	id := best.MusicRID
	if i := strings.LastIndex(id, "_"); i >= 0 {
		id = id[i+1:]
	}
	if id == "" {
		return ""
	}
	lq := url.Values{"musicId": {id}}
	lyText, _, err := lyricHTTPGetWithHeaders("https://kuwo.cn/openapi/v1/www/lyric/getlyric?"+lq.Encode(), map[string]string{"Referer": "https://kuwo.cn/"})
	if err != nil {
		return ""
	}
	var lyEnv struct {
		Data struct {
			Lrclist []struct {
				Time       string `json:"time"`
				LineLyrics string `json:"lineLyric"`
			} `json:"lrclist"`
		} `json:"data"`
	}
	if json.Unmarshal([]byte(lyText), &lyEnv) != nil || len(lyEnv.Data.Lrclist) == 0 {
		return ""
	}
	var sb strings.Builder
	for _, line := range lyEnv.Data.Lrclist {
		if line.Time == "" || strings.TrimSpace(line.LineLyrics) == "" {
			continue
		}
		secs := 0.0
		fmt.Sscanf(strings.TrimSpace(line.Time), "%f", &secs)
		if secs < 0 {
			secs = 0
		}
		m := int(secs) / 60
		s := secs - float64(m)*60
		sb.WriteString(fmt.Sprintf("[%02d:%05.2f]%s\n", m, s, line.LineLyrics))
	}
	lrc := strings.TrimSpace(sb.String())
	if lyricHasTimestamps(lrc) {
		return lrc
	}
	return ""
}

// ---------------- 咪咕 Migu ----------------
type miguSong struct {
	CopyrightID string `json:"copyrightId"`
	Name        string `json:"name"`
	Singers     []struct {
		Name string `json:"name"`
	} `json:"singers"`
	Duration string `json:"duration"`
	LyricURL string `json:"lyricUrl"`
	TrcURL   string `json:"trcUrl"`
}

func lyricSearchMigu(title, artist string, duration float64) string {

	q := url.Values{
		"text":         {strings.TrimSpace(title + " " + artist)},
		"pageNo":       {"1"},
		"pageSize":     {"10"},
		"searchSwitch": {`{"song":1}`},
	}
	searchURL := "https://c.musicapp.migu.cn/MIGUM3.0/v1.0/content/search_all.do?" + q.Encode()
	text, hdr, err := lyricHTTPGetWithHeaders(searchURL, nil)
	if err != nil {
		return ""
	}
	// 咪咕接口需要携带 UA 等参数，若返回非 JSON 尝试标准 GET
	_ = hdr
	var env struct {
		SongResultData struct {
			Result []miguSong `json:"result"`
		} `json:"songResultData"`
	}
	if json.Unmarshal([]byte(text), &env) != nil || len(env.SongResultData.Result) == 0 {
		return ""
	}
	best := &env.SongResultData.Result[0]
	bestScore := -1.0
	for i := range env.SongResultData.Result {
		s := &env.SongResultData.Result[i]
		names := make([]string, 0, len(s.Singers))
		for _, sg := range s.Singers {
			names = append(names, sg.Name)
		}
		artistStr := strings.Join(names, " / ")
		sc := lyricScoreCandidate(s.Name, artistStr, miguDurSec(s.Duration), title, artist, duration)
		if sc > bestScore {
			bestScore = sc
			best = s
		}
	}
	if lyricSimilarity(lyricCleanName(title), lyricCleanName(best.Name)) < 0.35 {
		return ""
	}
	// 优先 trcUrl（带字粒时间轴），否则 lyricUrl
	for _, u := range []string{best.TrcURL, best.LyricURL} {
		if u == "" {
			continue
		}
		if strings.HasPrefix(u, "//") {
			u = "https:" + u
		}
		if !strings.HasPrefix(u, "http") {
			continue
		}
		body, _, errGet := lyricHTTPGetWithHeaders(u, nil)
		if errGet == nil {
			lrc := strings.TrimSpace(body)
			if lyricHasTimestamps(lrc) {
				return lrc
			}
		}
	}
	return ""
}

func miguDurSec(raw string) float64 {
	if raw == "" {
		return 0
	}
	if ms, e := parseFloatSafe(raw); e == nil {
		if ms < 10000 { // 秒
			return ms
		}
		return ms / 1000 // 毫秒
	}
	mm, ss := 0, 0
	if _, err := fmt.Sscanf(raw, "%d:%d", &mm, &ss); err == nil {
		return float64(mm*60 + ss)
	}
	return 0
}

func parseFloatSafe(s string) (float64, error) {
	var v float64
	_, err := fmt.Sscanf(strings.TrimSpace(s), "%f", &v)
	return v, err
}

// ---------------- LRCLIB ----------------
func lyricSearchLrclib(title, artist string, duration float64) string {
	q := url.Values{
		"track_name":     {title},
		"artist_name":    {artist},
		"syncedLyrics":   {"true"},
	}
	searchURL := "https://lrclib.net/api/search?" + q.Encode()
	text, _, err := lyricHTTPGetWithHeaders(searchURL, nil)
	if err != nil {
		return ""
	}
	var items []struct {
		TrackName     string  `json:"trackName"`
		ArtistName    string  `json:"artistName"`
		Duration      float64 `json:"duration"`
		SyncedLyrics  string  `json:"syncedLyrics"`
	}
	if json.Unmarshal([]byte(text), &items) != nil || len(items) == 0 {
		return ""
	}
	best := &items[0]
	bestScore := -1.0
	for i := range items {
		it := &items[i]
		if strings.TrimSpace(it.SyncedLyrics) == "" {
			continue
		}
		sc := lyricScoreCandidate(it.TrackName, it.ArtistName, it.Duration, title, artist, duration)
		if sc > bestScore {
			bestScore = sc
			best = it
		}
	}
	if lyricSimilarity(lyricCleanName(title), lyricCleanName(best.TrackName)) < 0.35 {
		return ""
	}
	if lyricHasTimestamps(best.SyncedLyrics) {
		return strings.TrimSpace(best.SyncedLyrics)
	}
	return ""
}