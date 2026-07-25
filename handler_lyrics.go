package main

import (
	"encoding/binary"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func LyricsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

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

	// 优先读取同名 .lrc 文件
	lrcPath := foundPath[:len(foundPath)-len(filepath.Ext(foundPath))] + ".lrc"
	if data, err := os.ReadFile(lrcPath); err == nil && len(data) > 0 {
		w.Write(data)
		return
	}

	// 回退到内嵌歌词
	lyrics, err := extractLyricsFromFile(foundPath)
	if err != nil {
		http.Error(w, "读取歌词失败: "+err.Error(), 500)
		return
	}

	if lyrics == "" {
		http.Error(w, "未找到歌词", 404)
		return
	}

	saveLyricsAsLRC(foundPath, lyrics)

	w.Write([]byte(lyrics))
}

func saveLyricsAsLRC(mediaPath string, lyrics string) {
	ext := filepath.Ext(mediaPath)
	lrcPath := mediaPath[:len(mediaPath)-len(ext)] + ".lrc"

	if _, err := os.Stat(lrcPath); err == nil {
		return
	}

	os.WriteFile(lrcPath, []byte(lyrics), 0644)
}

func extractLyricsFromFile(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return "", err
	}
	fileSize := stat.Size()

	lyrics := extractLyrics3(f, fileSize)
	if lyrics != "" {
		return lyrics, nil
	}

	lyrics = extractID3v2USLT(f)
	if lyrics != "" {
		return lyrics, nil
	}

	return "", nil
}

func extractLyrics3(f *os.File, fileSize int64) string {
	if fileSize < 15+128 {
		return ""
	}

	f.Seek(-128-15, 2)
	var buf [15]byte
	if _, err := io.ReadFull(f, buf[:]); err != nil {
		return ""
	}

	marker := string(buf[6:15])
	hasID3v1 := true

	if marker != "LYRICS200" && marker != "LYRICSEND" {
		f.Seek(-15, 2)
		var buf2 [15]byte
		if _, err := io.ReadFull(f, buf2[:]); err != nil {
			return ""
		}
		marker = string(buf2[6:15])
		if marker != "LYRICS200" && marker != "LYRICSEND" {
			return ""
		}
		buf = buf2
		hasID3v1 = false
	}

	var lyricsSize int
	if marker == "LYRICS200" {
		sizeStr := strings.TrimSpace(string(buf[0:6]))
		var err error
		lyricsSize, err = strconv.Atoi(sizeStr)
		if err != nil {
			return ""
		}
	} else {
		lyricsSize = 5100
	}

	var seekPos int64
	if hasID3v1 {
		seekPos = fileSize - 128 - 15 - int64(lyricsSize)
	} else {
		seekPos = fileSize - 15 - int64(lyricsSize)
	}
	if seekPos < 0 {
		seekPos = 0
	}

	f.Seek(seekPos, 0)
	lyricsData := make([]byte, lyricsSize)
	if _, err := io.ReadFull(f, lyricsData); err != nil {
		return ""
	}

	dataStr := string(lyricsData)

	if !strings.HasPrefix(dataStr, "LYRICSBEGIN") {
		return ""
	}

	pos := 11
	for pos+8 <= len(dataStr) {
		fieldID := dataStr[pos : pos+3]
		fieldSizeStr := dataStr[pos+3 : pos+8]
		fieldSize, err := strconv.Atoi(fieldSizeStr)
		if err != nil {
			break
		}

		if fieldID == "LYR" {
			contentStart := pos + 8
			contentEnd := contentStart + fieldSize
			if contentEnd > len(dataStr) {
				contentEnd = len(dataStr)
			}
			return dataStr[contentStart:contentEnd]
		}

		pos += 8 + fieldSize
	}

	return ""
}

func extractID3v2USLT(f *os.File) string {
	f.Seek(0, 0)
	var header [10]byte
	if _, err := io.ReadFull(f, header[:]); err != nil {
		return ""
	}

	if header[0] != 'I' || header[1] != 'D' || header[2] != '3' {
		return ""
	}

	id3Size := int((header[6]&0x7f)<<21 | (header[7]&0x7f)<<14 | (header[8]&0x7f)<<7 | (header[9]&0x7f))
	id3Data := make([]byte, id3Size)
	if _, err := io.ReadFull(f, id3Data); err != nil {
		return ""
	}

	for i := 0; i < len(id3Data)-10; {
		frameID := string(id3Data[i : i+4])
		if frameID[0] == 0 {
			break
		}

		frameSize := int(id3Data[i+4])<<24 | int(id3Data[i+5])<<16 | int(id3Data[i+6])<<8 | int(id3Data[i+7])
		if frameSize <= 0 || i+10+frameSize > len(id3Data) {
			break
		}

		if frameID == "USLT" {
			frameData := id3Data[i+10 : i+10+frameSize]
			return parseUSLT(frameData)
		}

		i += 10 + frameSize
	}
	return ""
}

func parseUSLT(data []byte) string {
	if len(data) < 4 {
		return ""
	}

	encoding := data[0]
	rest := data[4:]

	var descEnd int
	if encoding == 1 {
		for j := 0; j < len(rest)-1; j++ {
			if rest[j] == 0 && rest[j+1] == 0 {
				descEnd = j + 2
				break
			}
		}
	} else {
		for j := 0; j < len(rest); j++ {
			if rest[j] == 0 {
				descEnd = j + 1
				break
			}
		}
	}

	if descEnd >= len(rest) {
		return ""
	}

	lyricsData := rest[descEnd:]

	switch encoding {
	case 1:
		if len(lyricsData) >= 2 && lyricsData[0] == 0xFE && lyricsData[1] == 0xFF {
			return decodeUTF16BE(lyricsData[2:])
		}
		if len(lyricsData) >= 2 && lyricsData[0] == 0xFF && lyricsData[1] == 0xFE {
			return decodeUTF16LE(lyricsData[2:])
		}
		if len(lyricsData)%2 == 0 {
			return decodeUTF16BE(lyricsData)
		}
		return string(lyricsData)
	case 2:
		return string(lyricsData)
	case 3:
		return string(lyricsData)
	default:
		return string(lyricsData)
	}
}

func decodeUTF16BE(data []byte) string {
	if len(data)%2 != 0 {
		data = data[:len(data)-1]
	}
	runes := make([]rune, 0, len(data)/2)
	for i := 0; i < len(data); i += 2 {
		r := rune(binary.BigEndian.Uint16(data[i : i+2]))
		runes = append(runes, r)
	}
	return string(runes)
}

func decodeUTF16LE(data []byte) string {
	if len(data)%2 != 0 {
		data = data[:len(data)-1]
	}
	runes := make([]rune, 0, len(data)/2)
	for i := 0; i < len(data); i += 2 {
		r := rune(binary.LittleEndian.Uint16(data[i : i+2]))
		runes = append(runes, r)
	}
	return string(runes)
}
