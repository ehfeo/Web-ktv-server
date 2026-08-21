# Debug Session: media-cache-deadlock

## 状态: [OPEN]

## Bug 描述
- 现象：点过的歌再次点歌（缓存命中场景），前端卡在"检查"，后台服务整个停掉
- 修改前正常：方案A 改动（合并 ffprobe + 给 checkMediaTracks 加 modTime 缓存）后出现
- 影响范围：页面服务整个停掉（不仅是单次请求超时）

## 假设（Falsifiable Hypotheses）

### H1: mediaTracksCache 死锁
- 描述：缓存写入路径 `mediaTracksCache.Lock()` 持锁时调用了 `fileInfo.ModTime()`，但 `fileInfo` 在缓存命中分支已使用，路径不应有死锁。需运行时验证。
- 观察点：并发请求同一文件时是否会卡在 Lock 等待

### H2: mediaInfoCache 死锁
- 描述：getMediaInfo 的 RLock 与 Lock 路径是否在某种条件下重入
- 观察点：getMediaInfo 调用栈是否卡在锁

### H3: HTTP handler 在缓存返回后忘记写响应，前端一直等
- 描述：CheckTracksHandler 在缓存命中时返回 warning 但 JSON 编码失败/返回 nil 导致前端死等
- 观察点：前端 fetch 是否完成、后端是否写完响应

### H4: ffprobe 子进程异常导致 process.Wait 阻塞整个 Go runtime
- 描述：合并后的 ffprobe 命令格式有问题，ffprobe 输出大量数据或 hang 住，CombinedOutput 阻塞
- 观察点：ffprobe 进程是否仍在运行、CombinedOutput 调用前后日志

### H5: 服务整个停掉暗示 panic，导致 http handler 没有恢复
- 描述：Go 默认 http server 在 handler panic 时会 recover 单个连接，但如果 panic 在 goroutine 中则会让进程退出
- 观察点：stdout/stderr 日志是否有 panic stack trace

## 进度日志

### Step 1: 假设已列出，等待插桩收集证据
