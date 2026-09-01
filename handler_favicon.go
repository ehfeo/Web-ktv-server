package main

import (
	_ "embed"
	"net/http"
)

// faviconIco 浏览器标签页图标（.ico，内嵌多分辨率 16-256px）。
// 设计：饱和紫色渐变圆角底 + 高饱和青色麦克风 + 加粗白色 KTV 字样，高对比便于一眼识别。
//
//go:embed favicon.ico
var faviconIco []byte

// FaviconHandler 返回 /favicon.ico（兼容所有浏览器的标准格式）。
func FaviconHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/x-icon")
	w.Header().Set("Cache-Control", "max-age=604800")
	w.Write(faviconIco)
}