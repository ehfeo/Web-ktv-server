package main

import (
	"log"
	"net/http"
	"os"
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

	// 优先用findMediaFile（内存映射表O(1)查找，避免Walk磁盘扫描）
	foundPath := findMediaFile(name)
	if foundPath != "" {
		http.ServeFile(w, r, foundPath)
		return
	}

	// fallback: 检查上传目录
	if uploadDirPath != "" {
		// 从文件名中提取纯文件名
		fileNameOnly := name
		if idx := strings.LastIndex(name, "/"); idx != -1 {
			fileNameOnly = name[idx+1:]
		}
		uploadFilePath := uploadDirPath + string(os.PathSeparator) + fileNameOnly
		if _, err := os.Stat(uploadFilePath); err == nil {
			log.Printf("[FileHandler] 在上传目录找到文件: %s", fileNameOnly)
			http.ServeFile(w, r, uploadFilePath)
			return
		}
	}

	log.Printf("[FileHandler][ERROR] 文件未找到: name=%s", name)
	http.Error(w, "not found", 404)
}
