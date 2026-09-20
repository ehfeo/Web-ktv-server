# v1.13.0

## 核心修复：隔行扫描 (1080i) 视频浏览器播放卡死
- 直接可播判定收紧：视频必须为 **H.264 且非隔行扫描**，且音频为 AAC/MP3 等浏览器可解码格式才直接播放。
- 通过 ffprobe `field_order` 自动检测隔行（tt/bb/tb/bt），失败可靠回退。
- 隔行 H.264 不再直放（原直放会整读卡死），改为 **yadif 去隔行重编码**（GPU 流水线优先，自动降级 libx264），产物覆盖原文件，下次直接可用。

## 新特性：边写边播（规则C）
- 队列中无任何"已就绪"曲目、且排在最前的曲目已真正进入转码时，直接用分片 MP4（fragmented MP4，含 moov 于头部可渐进播放）边写盘边推流。
- 转码一完成即切换为最终文件播放，前端无需等待整首转码完成，显著缩短隔行视频首响时间。
- 智能保留：其它排队（waiting）曲目在原点播页各自排队，互不冲突。
- 服务端以流式（无固定 Content-Length）推送增长中的文件，转码覆盖原文件后已打开的句柄仍可读完整。

## 提升
- 移除了不可靠的 faststart（moov 前置）方案，转码为可渐进播放的分片 MP4，彻底解决 1080i 卡死。

## 说明
- 编译使用 `-ldflags="-s -w"` 压缩体积，Heat 版与 32 位版同步发布。
- 隔行源边写边播仅对分片 MP4 生效（mkv 等产物写入未完成浏览器无法渐进播放）。

---

# v1.13.0

## Core Fix: Interlaced (1080i) Video Hangs in Browser
- Tightened direct-play rule: a video must be **H.264 AND progressive (non-interlaced)**, with browser-decodable audio (AAC/MP3...) to play directly.
- Auto-detects interlacing via ffprobe `field_order` (tt/bb/tb/bt) with a 2000-entry LRU cache.
- Interlaced H.264 is no longer played raw (previously froze the whole file in memory); it now goes through **yadif deinterlacing re-encode** (GPU pipeline preferred, auto-fallback to libx264). The result overwrites the original file, so next time it plays directly.

## New Feature: Play-While-Transcoding (Rule C)
- When no queued song is "ready" and the front item has genuinely entered transcoding, the app streams the in-progress fragmented MP4 (moov at head, progressively playable) to the player in real time.
- On transcode completion it seamlessly switches to the final file; the front-end no longer waits the full conversion.
- Safe queuing: other waiting items keep their own per-page queue positions, no conflicts across on-demand pages.
- Server pushes the growing file with no fixed Content-Length; after the temp file overwrites the original, the already-open handle still reads to the end.

## Improvement
- Removed the unreliable faststart (moov front-loading) approach; transcoded output is now a progressively-playable fragmented MP4, finally fixing the 1080i hang.

## Notes
- Built with `-ldflags="-s -w"` to reduce size; 64-bit and 32-bit builds ship together.
- Play-while-transcoding applies only to fragmented MP4 (unfinished mkv is not progressively playable in browsers).