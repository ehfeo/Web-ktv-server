package main

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func FileHandler(w http.ResponseWriter, r *http.Request) {
	// 接收文件名，移除URL参数
	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, "not found", 404)
		return
	}
	
	// 移除可能的时间戳参数
	if idx := strings.Index(name, "?"); idx != -1 {
		name = name[:idx]
	}

	// 构建搜索目录列表（mediaDirs + uploadDirPath）
	searchDirs := mediaDirs
	if uploadDirPath != "" {
		found := false
		for _, d := range mediaDirs {
			if filepath.Clean(d) == filepath.Clean(uploadDirPath) {
				found = true
				break
			}
		}
		if !found {
			searchDirs = append(searchDirs, uploadDirPath)
		}
	}

	// 尝试从文件名中提取目录前缀（格式：目录名/文件名或目录名/子目录/文件名）
	var foundPath string
	var fileNameOnly string = name
	
	// 检查是否包含目录前缀
	parts := strings.SplitN(name, "/", 2)
	if len(parts) == 2 {
		dirPrefix := parts[0]
		remainingPath := parts[1]
		fileNameOnly = filepath.Base(remainingPath)
		
		// 在对应目录中查找（包括子目录）
		for _, dir := range searchDirs {
			if filepath.Base(dir) == dirPrefix {
				// 先尝试直接路径
				fullPath := filepath.Join(dir, remainingPath)
				if _, err := os.Stat(fullPath); err == nil {
					http.ServeFile(w, r, fullPath)
					return
				}
				
				// 递归搜索该目录
				err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
					if err != nil {
						return nil
					}
					if !info.IsDir() && info.Name() == fileNameOnly {
						foundPath = path
						return filepath.SkipAll
					}
					return nil
				})
				
				if err == nil && foundPath != "" {
					http.ServeFile(w, r, foundPath)
					return
				}
			}
		}
	}

	// 如果没有前缀或查找失败，在所有媒体目录中搜索
	for _, dir := range searchDirs {
		// 先尝试直接路径
		fullPath := filepath.Join(dir, name)
		if _, err := os.Stat(fullPath); err == nil {
			http.ServeFile(w, r, fullPath)
			return
		}

		// 递归搜索整个目录树
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if !info.IsDir() && info.Name() == fileNameOnly {
				foundPath = path
				return filepath.SkipAll
			}
			return nil
		})

		if err == nil && foundPath != "" {
			http.ServeFile(w, r, foundPath)
			return
		}
	}

	http.Error(w, "not found", 404)
}