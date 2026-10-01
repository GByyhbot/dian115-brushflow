# 来源与发布边界

- DIAN115 公开协议、Schema、OpenAPI、检查工具、构建/签名脚本、Vite 配置及图标来自：
  https://github.com/madbrolab/dian115/tree/7637a392e9512bd0f7965b99445768969eb075fc/docs/plugin-platform
- `runtime/main.go` 的 ABI 与调用封套参考该目录的完整 Go WASM 示例，业务实现独立编写。
- 业务需求与规则含义参考：
  https://github.com/jxxghp/MoviePilot-Plugins/tree/9b9b14bc0f0bb3890b0f699a84f9bd0df063b2e3/plugins.v3/brushflow
- 没有复制 MoviePilot Python 插件源码，也没有复制 DIAN115 主程序源码。

上游文件的权利与使用条件归原作者所有。本架构不擅自为上游材料重新授予 MIT 等许可；manifest 暂以 `NOASSERTION` 标记，正式分发前需由维护者确认适用许可。开发私钥与生成包不进入源码仓库。
