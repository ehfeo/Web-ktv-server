package main

import (
	"encoding/json"
	"net/http"
	"os"
	"sort"
	"sync"
	"time"
)

// HotPlayStats 热播统计
var HotPlayStats struct {
	sync.RWMutex
	counts   map[string]int // filePath -> playCount
	dirty    bool
	lastSave time.Time
}

// HotSong 热播歌曲条目
type HotSong struct {
	Path  string `json:"path"`
	Name  string `json:"name"`
	Count int    `json:"count"`
}

func init() {
	HotPlayStats.counts = make(map[string]int)
}

// IncrementPlayCount 增加点播次数
func IncrementPlayCount(filePath string) {
	HotPlayStats.Lock()
	HotPlayStats.counts[filePath]++
	HotPlayStats.dirty = true
	HotPlayStats.Unlock()

	// 防抖保存：距上次保存超过5秒才保存
	go func() {
		time.Sleep(5 * time.Second)
		SaveHotPlayData()
	}()
}

// SaveHotPlayData 保存热播数据到文件
func SaveHotPlayData() {
	HotPlayStats.Lock()
	defer HotPlayStats.Unlock()

	if !HotPlayStats.dirty {
		return
	}
	if time.Since(HotPlayStats.lastSave) < 5*time.Second {
		return
	}

	data, err := json.Marshal(HotPlayStats.counts)
	if err != nil {
		return
	}
	os.WriteFile("ktv_hotplay.json", data, 0644)
	HotPlayStats.dirty = false
	HotPlayStats.lastSave = time.Now()
}

// LoadHotPlayData 启动时加载热播数据
func LoadHotPlayData() {
	data, err := os.ReadFile("ktv_hotplay.json")
	if err != nil {
		return
	}
	HotPlayStats.Lock()
	defer HotPlayStats.Unlock()
	json.Unmarshal(data, &HotPlayStats.counts)
}

// GetHotSongs 获取热播歌曲列表（按点播次数降序，最多limit首）
func GetHotSongs(limit int) []HotSong {
	HotPlayStats.RLock()
	defer HotPlayStats.RUnlock()

	var list []HotSong
	// 遍历所有媒体文件，用f.Path(相对路径)匹配点播次数
	allFiles := getCachedMediaList()
	countMap := make(map[string]int)
	for _, f := range allFiles {
		if c, ok := HotPlayStats.counts[f.Path]; ok {
			countMap[f.Path] = c
		}
	}

	for path, count := range countMap {
		if count > 0 {
			// 从path提取name
			name := path
			for i := len(path) - 1; i >= 0; i-- {
				if path[i] == '/' || path[i] == '\\' {
					name = path[i+1:]
					break
				}
			}
			list = append(list, HotSong{Path: path, Name: name, Count: count})
		}
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].Count > list[j].Count
	})

	if limit > 0 && len(list) > limit {
		list = list[:limit]
	}
	return list
}

// GetPlayCount 获取单首歌曲的点播次数
func GetPlayCount(filePath string) int {
	HotPlayStats.RLock()
	defer HotPlayStats.RUnlock()
	return HotPlayStats.counts[filePath]
}

// HotSongsHandler API handler for /api/hot-songs
func HotSongsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	limit := 100
	songs := GetHotSongs(limit)
	if songs == nil {
		songs = []HotSong{}
	}
	json.NewEncoder(w).Encode(songs)
}

// PlayCountHandler API handler for /api/play-count?name=xxx
func PlayCountHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	name := r.URL.Query().Get("name")
	count := GetPlayCount(name)
	json.NewEncoder(w).Encode(map[string]int{"count": count})
}

// IncrementPlayHandler API handler for /api/increment-play
func IncrementPlayHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, "missing name", 400)
		return
	}
	IncrementPlayCount(name)
	w.Write([]byte(`{"ok":true}`))
}
