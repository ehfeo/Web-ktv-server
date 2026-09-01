# v1.8.0 更新说明

## 新增
- 应用图标 favicon.ico：饱和紫色渐变圆角底 + 高饱和黄色 KTV 字样，浏览器标签页可一眼识别
- 图标支持 128/64/48/32/16 多分辨率内嵌
- 新增 handler_favicon.go，改为主程序以 /favicon.ico 提供服务
- 新增 build.bat 构建脚本与 gen_favicon.ps1 图标生成脚本

## 修复
- 修复网页标签图标显示问题（原 favicon.svg 过大/不清晰），统一改用 /favicon.ico