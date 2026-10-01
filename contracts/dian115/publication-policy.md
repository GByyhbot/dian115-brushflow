# 公开源边界

本仓库只发布插件契约与第三方插件示例。主项目源码（`cmd/`、`internal/`、`frontend/src/`、构建设置、部署文件、生成的发布包与私钥）永不公开，也不应出现在本仓库。

## 允许发布

- `docs/plugin-platform/` 下的公开契约、Schema、OpenAPI、示例和黑盒联调工具。
- 插件作者基于公开契约独立编写的插件工程。
- 由公开契约生成的构建、测试和文档产物（不含主项目内部文件）。

## 禁止发布

- 主项目的 `cmd/`、`internal/`、`frontend/src/`、构建/部署脚本、CI 配置中的内部步骤、发布包（`.d115p` 等）、签名私钥和凭据。
- 从主项目复制或改写而来、且未被公开契约覆盖的任何源码。
- 未公开的 API 定义、内部 Schema 或内部文档。

## 检查

每次公开提交前运行：

```bash
node docs/plugin-platform/conformance/verify-public-surface.mjs
```

CI 会在公开仓库中拒绝违反本边界的提交。
