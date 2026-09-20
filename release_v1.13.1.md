# v1.13.1 发布说明 / Release Notes

## 中文

本版本为维护更新，聚焦控制台日志降噪与内容审核体验优化。

### 修复与优化
- **控制台日志降噪**：`findMediaFile` 的成功命中（内存映射/缓存/basename）不再逐请求打印日志。浏览器对 mp4 的 Range 分段请求此前会让播放高峰刷屏（单秒可达几十条），现仅保留失败和慢路径日志，异常情况仍可正常排查。
- **点播未审核文件自动引导**：点播到待审核（uploads）文件时，服务端输出清晰的 `[点播拦截]` 日志，前端在播放队列标记"待审核"，并自动打开审核页面（/review）以便快速审核。
- **搜索/点播条目标注"待审核"**：来自 uploads 的文件在搜索结果与浏览分页网格上，以红字"待审核"标出，一眼可辨，无需点播后才知道。

### 技术说明
- 待审核判定依据来源：uploads 目录下的文件视为待审核（与内容审核的 `reviewFileAllowed` 规则一致），统一由 `markUploadReview` 在搜索与分页接口标注，避免逐文件探测的额外开销。

---

## English

Maintenance release focused on console log quieting and content-review UX.

### Fixes & Improvements
- **Console log quieting**: successful `findMediaFile` hits (memory-map / cache / basename) no longer log per request. Browser Range-segment requests to MP4s previously flooded the console during playback peaks (tens per second). Now only failure and slow paths are logged, so real issues remain diagnosable.
- **Auto-guide to review when playing a pending file**: playing an unreviewed (uploads) file logs a clear `[点播拦截]` entry server-side, flags the item "待审核" in the queue client-side, and auto-opens the review page (/review) for quick action.
- **Flag "待审核" on search/playlist items**: files from the uploads folder are shown with a red "待审核" tag in both search results and the browsing/grid view, so they are recognizable before playing.

### Technical Note
- Pending status is decided by source: anything under the uploads directory is treated as unreviewed (consistent with the review gate `reviewFileAllowed`), labelled uniformly by `markUploadReview` in both search and paged APIs, avoiding per-file probing overhead.