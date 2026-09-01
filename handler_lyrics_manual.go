package main

import (
	"encoding/json"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// lyricCandidate 一条可手动选择的歌词候选，直接携带完整歌词文本（LRC 或裸文本）。
type lyricCandidate struct {
	Source   string  `json:"source"`
	Title    string  `json:"title"`
	Artist   string  `json:"artist"`
	Duration float64 `json:"duration"`
	Lyrics   string  `json:"lyrics"`
}

// LyricMetaHandler 返回文件名解析出的歌名与歌手，供前端手动选择歌词时预填搜索框。
func LyricMetaHandler(w http.ResponseWriter, r *http.Request) {
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
	title, artist := lyricParseTrackName(foundPath)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(map[string]string{"title": title, "artist": artist})
}

// LyricCandidatesHandler 返回歌曲可用歌词候选列表（含来自各接口的完整歌词），供前端手动选择。
func LyricCandidatesHandler(w http.ResponseWriter, r *http.Request) {
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
	title, artist := lyricParseTrackName(foundPath)
	// 允许前端手动覆盖解析歌名/歌手
	if t := strings.TrimSpace(r.URL.Query().Get("title")); t != "" {
		title = t
	}
	if a := strings.TrimSpace(r.URL.Query().Get("artist")); a != "" {
		artist = a
	}
	duration := getDuration(foundPath)
	normTitle := lyricCleanName(title)
	var out []lyricCandidate

	// --- 源1：QQ音乐候选（按时长→歌名评分降序，取前几首有歌词的） ---
	if songs := lyricGatherCandidates(title, artist); len(songs) > 0 {
		type scoredSong struct {
			s        LyricQQSong
			titleSim float64
			durSim   float64
		}
		var scored []scoredSong
		for _, s := range songs {
			ts := lyricSimilarity(normTitle, lyricCleanName(s.Name))
			if s.Title != "" {
				if v := lyricSimilarity(normTitle, lyricCleanName(s.Title)); v > ts {
					ts = v
				}
			}
			ds := 1.0
			if duration > 0 {
				qd := lyricQQDuration(s)
				if qd > 0 {
					if diff := math.Abs(duration - qd); diff > 30 {
						ds = 0
					} else {
						ds = 1 - diff/120.0
					}
				}
			}
			scored = append(scored, scoredSong{s, ts, ds})
		}
		sort.Slice(scored, func(i, j int) bool {
			return scored[i].durSim*10000+scored[i].titleSim > scored[j].durSim*10000+scored[j].titleSim
		})
		for _, c := range scored {
			if lrc, err := lyricFetchQQ(c.s.Mid); err == nil && lyricHasTimestamps(lrc) {
				var cn []string
				for _, sg := range c.s.Singer {
					cn = append(cn, strings.TrimSpace(sg.Name))
				}
				out = append(out, lyricCandidate{
					Source: "QQ音乐", Title: c.s.Name, Artist: strings.Join(cn, "/"),
					Duration: lyricQQDuration(c.s), Lyrics: lrc,
				})
			}
			if len(out) >= 4 {
				break
			}
		}
	}

	// --- 源2：网易云候选 ---
	if songs := lyricGatherNeteaseCandidates(title, artist); len(songs) > 0 {
		type scoredSong struct {
			s        LyricNeteaseSong
			titleSim float64
			durSim   float64
		}
		var scored []scoredSong
		for _, s := range songs {
			ts := lyricSimilarity(normTitle, lyricCleanName(s.Name))
			ds := 1.0
			if duration > 0 && s.Duration > 0 {
				if diff := math.Abs(duration - float64(s.Duration)/1000.0); diff > 30 {
					ds = 0
				} else {
					ds = 1 - diff/120.0
				}
			}
			scored = append(scored, scoredSong{s, ts, ds})
		}
		sort.Slice(scored, func(i, j int) bool {
			return scored[i].durSim*10000+scored[i].titleSim > scored[j].durSim*10000+scored[j].titleSim
		})
		for _, c := range scored {
			if lrc, err := lyricFetchNetease(c.s.Id); err == nil && lyricHasTimestamps(lrc) {
				var an []string
				for _, ar := range c.s.Artists {
					an = append(an, strings.TrimSpace(ar.Name))
				}
				out = append(out, lyricCandidate{
					Source: "网易云", Title: c.s.Name, Artist: strings.Join(an, "/"),
					Duration: float64(c.s.Duration) / 1000.0, Lyrics: lrc,
				})
			}
			if len(out) >= 4 {
				break
			}
		}
	}

	// --- 其余各源：取自动匹配到的最佳歌词（标题用文件名解析结果） ---
	addProvider := func(source, lrc string) {
		if lrc != "" {
			out = append(out, lyricCandidate{Source: source, Title: title, Artist: artist, Duration: duration, Lyrics: lrc})
		}
	}
	addProvider("LRCLIB", lyricSearchLrclib(title, artist, duration))
	addProvider("酷狗", lyricSearchKugou(title, artist, duration))
	addProvider("酷我", lyricSearchKuwo(title, artist, duration))
	addProvider("咪咕", lyricSearchMigu(title, artist, duration))
	if lrc, err := lyricSearchLrcCx(title, artist); err == nil && lrc != "" {
		addProvider("lrc.cx", lrc)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(out)
}

// LyricSaveHandler 把手动选中的歌词保存为歌曲同目录同名 .lrc，供下次自动命中。
// 手动选择以用户所选为准，因此即使已存在同名 .lrc 也会强制覆盖。
func LyricSaveHandler(w http.ResponseWriter, r *http.Request) {
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
	body, err := io.ReadAll(r.Body)
	if err != nil || len(body) == 0 {
		http.Error(w, "缺少歌词内容", 400)
		return
	}
	lrcPath := foundPath[:len(foundPath)-len(filepath.Ext(foundPath))] + ".lrc"
	if err := os.WriteFile(lrcPath, body, 0644); err != nil {
		http.Error(w, "保存歌词失败: "+err.Error(), 500)
		return
	}
	w.Write([]byte("ok"))
}