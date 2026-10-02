# DIAN115 BrushFlow

DIAN115 Plugin API v2 刷流插件。1.0 版本通过标准 JSON Feed 获取候选资源，通过 qBittorrent Web API 下载、检查和清理种子，不依赖 DIAN115 未公开的内部 PT 接口。

## 安装

在 DIAN115 的“插件中心 → 仓库与开发”中添加以下任一地址：

```text
https://raw.githubusercontent.com/GByyhbot/dian115-brushflow/main/plugin-market/index.json
```

如果当前网络无法访问 GitHub Raw，可使用 jsDelivr 索引地址：

```text
https://cdn.jsdelivr.net/gh/GByyhbot/dian115-brushflow@main/plugin-market/index.json
```

不要填写 GitHub 网页的 `.../blob/.../plugin-market/index.json` 地址。如果 DIAN115 的 GitHub 加速地址不可用，应在系统网络设置中关闭或更换该加速地址，然后删除旧仓库并用上述地址重新添加。插件包仍从 GitHub Release 下载，因此安装时也需要容器能够访问 GitHub Release，或配置一个可用的 GitHub 加速地址。

也可以从 [GitHub Release](https://github.com/GByyhbot/dian115-brushflow/releases/latest) 下载 `.d115p`，然后在同一页面使用“本地导入”。本地导入会执行与市场安装相同的签名、完整性和权限检查。

## 功能

- 多站点、多 qBittorrent、多独立任务和常驻调度。
- 免费、H&R、标题正则、体积、做种人数和发布时间筛选。
- 每个任务限制活动种子数。
- 按做种时间、分享率、上传量、下载超时和免费到期清理。
- 使用任务标签和候选唯一标签双重确认所有权；不会清理无法确认归属的种子。
- Host Storage 持久化配置、种子记录、最近执行历史和后台心跳。
- 手动选种、手动检查、规则预览、Telegram 通知。
- qBittorrent 和站点凭据只保存为 dian115 托管凭据引用，配置中不保存 Cookie 或密码。

## 站点 JSON Feed

插件使用稳定、可测试的 JSON 边界。可以用站点 API、RSS 转换器或代理生成：

```json
{
  "items": [
    {
      "id": "site-torrent-123",
      "title": "Example.Movie.1080p",
      "size_bytes": 10737418240,
      "download_url": "magnet:?xt=urn:btih:...",
      "free": true,
      "hr": false,
      "published_at": "2026-10-02T08:00:00Z",
      "seeders": 3,
      "free_until": "2026-10-03T08:00:00Z"
    }
  ]
}
```

`id` 必须在站点内稳定且唯一。需要“仅免费”“排除 H&R”或最大做种人数时，相应字段缺失会被保守排除。启用最大发布时间后，时间缺失或格式错误也会被排除。`download_url` 只接受磁力链接或 HTTP(S) 种子 URL；HTTP(S) 种子会先由 Broker 使用站点托管凭据下载，再上传给 qBittorrent，单个文件上限 4 MiB。

## 凭据和网络

在 dian115 插件安装实例的托管凭据界面创建绑定，然后把返回的 `credential_ref` 填入站点或下载器：

- JSON Feed 通常绑定 `GET` 和 Feed 路径，可注入 Cookie、Authorization 或 query token。
- qBittorrent 绑定应覆盖 `GET/POST` 的 `/api/v2/` 路径。可注入已有 `SID=...` Cookie，或在可信反向代理中注入 Authorization。
- Broker 会删除登录响应的 `Set-Cookie`，插件无法自行保存 qB 登录会话。不要在插件配置中填写明文密码。
- qBittorrent 地址必须是 DIAN115 容器能够访问的地址；容器内的 `127.0.0.1` 通常不是物理宿主机。

## 安全启用顺序

1. 安装插件，配置站点 Feed 和 qBittorrent，任务保持停用。
2. 保存后用规则预览核对选种条件。
3. 点击“立即选种”，确认 qB 中出现 `brushflow-<task-id>` 与 `bfid-...` 标签。
4. 点击“立即检查”，确认状态同步正常。
5. 配置删除条件。所有删除条件为“任一满足”；数值 `0` 表示关闭该条件。
6. 明确决定是否勾选“删种时同时删除文件”，最后启用自动调度。

免费到期删除只处理尚未完成的任务。做种时间、分享率和上传量只处理已完成种子。下载超时只处理未完成种子。

## 构建、测试和打包

需要 Node.js 22+、npm 和支持 `wasip1` reactor/`//go:wasmexport` 的 Go；CI 使用 Go 1.27.1。

```sh
npm ci
go test ./...
npm run build
npm run check
npm run test:wasm
npm run check:openapi
npm run package
```

开发环境首次打包可运行 `npm run package -- --generate-key`。正式发布必须固定使用同一 Ed25519 发布密钥。私钥、`build/`、`releases/` 和 `.d115p` 均被 Git 忽略。

## 当前兼容范围

- DIAN115 `>=3.8.51 <5.0.0`（包括 4.0.64），Plugin API v2。
- qBittorrent Web API v2。
- 暂不包含 Transmission、MoviePilot 站点解析器、站点分享率控制、订阅排除和全局跨任务动态容量删除。

完整架构和安全边界见 [docs/architecture.md](docs/architecture.md)，上游来源见 [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md)。
