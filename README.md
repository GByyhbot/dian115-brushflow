# DIAN115 BrushFlow

将 MoviePilot brushflow 的策略迁移到 DIAN115 Plugin API v2 的独立插件初步架构。

**当前版本 0.1.0 仅支持任务配置、规则预览和后台心跳，不会添加、暂停或删除真实种子。**

## 已实现

- Go WASM reactor 入口和 Host Call 封装；不直接使用网络 Socket 或宿主文件。
- Host Storage 配置持久化，更新使用 ETag / If-Match，并携带稳定幂等键。
- 任务配置校验、免费/H&R/正则/体积筛选和候选 ID 去重。
- 未知促销或 H&R 信息保守排除；预览无下载副作用。
- Vue 3 / Naive UI 联邦管理页面，本地 mock 预览入口。
- 独立 resident 心跳，页面与 resident 通过 Host Storage 交换状态。
- 外部站点、下载器接口边界；不可用适配器明确返回错误。
- 配置冲突、重启读取、规则边界、预览无副作用及 Broker 存储测试。
- 官方固定版本协议检查工具、WASM ABI 检查和 GitHub Actions。

## 构建与验证

需要 Node.js 22+、npm、Go 1.24+（需支持 `wasip1` reactor 与 `//go:wasmexport`；本项目验证使用 Go 1.27.1）。

```sh
npm ci
go test ./...
npm run build
npm run check
npm run test:wasm
npm run check:openapi
```

输出为 `build/runtime/plugin.wasm` 和 `build/frontend/dist/assets/remoteEntry.js`。

```sh
npm run dev
```

本地页面仅以内存模拟配置保存；刷新浏览器会丢失 mock 数据。规则预览必须通过真实插件 runtime 或 Go 测试执行，本地 mock 不复制 Go 策略。

## 本地签名包

```sh
npm run package -- --generate-key
```

开发私钥、构建目录和 `.d115p` 已忽略，不提交到 GitHub。正式发布前替换 manifest 的发布者、仓库地址和 market 的包地址，并使用长期固定签名密钥。模板市场地址不是可用下载链接。

生成包可在 DIAN115 插件中心本地导入。**尚未在真实宿主验收安装、resident 生命周期及下载器认证；通过静态检查不等于完整宿主兼容。**

## 使用

1. 创建任务并保存。初版不开放任务启用开关，后台只报告就绪情况。
2. 设置包含/排除正则、免费/H&R 条件与体积范围；体积单位为字节。
3. 保存配置，再提交候选 JSON 进行规则预览。
4. 查看匹配结果。`run`、`check`、`delete` 在此版本均返回 `skipped`。

正则使用 Go RE2，与 Python `re` 不完全相同，不支持后向引用及环视。上游配置不能未经转换直接导入。

## 下一阶段

1. 验证 Broker 与 qBittorrent 的认证会话；明确 Set-Cookie 过滤后的认证路径。
2. 实现单站点 RSS/API 适配与明确的字段缺失策略。
3. 实现读取下载器、任务所有权和添加结果核对，再开放执行。
4. 引入持久化命令队列与单一 resident 执行者；目前未实现自动调度执行。
5. 删除预览、H&R 删除保护、幂等恢复后，再实现自动删除。
6. 多任务配额、促销到期、订阅排除、Transmission、统计归档。

详见 [架构说明](docs/architecture.md) 与 [来源说明](THIRD_PARTY_NOTICES.md)。
