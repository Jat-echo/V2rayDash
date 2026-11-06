# 2026-11-03 客户端专用订阅与 TLS 指纹

同一订阅的两台 iPhone 使用 Shadowrocket，一台的 TLS 指纹为旧 Safari 配置，另一台为 `Safari26_3`。故障手机能够连接香港入口 TCP 端口，但 VLESS 随即读取失败；用户手动改为 `Safari26_3` 后确认网页恢复。日志本身不足以确定所有读取失败的原因。

平台新增 Shadowrocket 节点专用订阅，仅包含 `proxies`，每个节点下发 `client-fingerprint: Safari26_3`。Clash Verge / Mihomo 保留 `chrome` 和原有 ML-KEM 开关；继续使用完整分流模板。通用地址按 User-Agent 识别客户端，也可使用 `?format=shadowrocket`、`?format=clash_meta` 明确指定，显式格式优先。订阅链接弹窗增加两个专用地址及复制按钮。

## 验证

- 后端全部测试及 `go vet` 通过；真实 SetupRoutes 和隔离 PostgreSQL schema 测试覆盖客户端识别、显式格式优先、禁用账号及原有模板参数。
- 前端 4 项测试、TypeScript 类型检查、正式构建通过。构建仍有已有的体积提示。
- 在生产数据库副本中关闭自动同步演练。33 份原有订阅的 VLESS、Clash Meta、Sing-box 共 99 份配置，内容 SHA256 逐项一致。
- 33 份新 Shadowrocket 配置中的 86 个可发布节点逐项与 Clash 格式对照；仅客户端指纹和不适用的 Mihomo ML-KEM 开关有差异，账号、连接参数、名称、落地旗帜及排序一致。
- 本机 Clash Verge 2.5.2 附带 Mihomo v1.19.29；用同一内核启动临时回环测试实例，使用现有 CYM 订阅的香港中转 → DMIT-US 节点成功访问 Google（约 87 KB），HTTPS 查询出口 IP 为 `179.255.98.196`。没有修改正在运行的 Clash Verge 配置。
- 正式页面的订阅弹窗已检查，两种客户端地址的格式参数、只读属性、复制按钮及桌面宽度下的布局正常。

## 上线结果

- 更新正式后端和前端；账号、订阅、关联、路由及受管节点定义保持不变，没有调用节点配置发布或升级代理内核。
- 切换及平台健康恢复约 7.25 秒。上线后重新验证原有 99 份配置哈希、客户端格式选择与页面入口。
- 正式订阅仍为 33 份，待同步和发布中节点均为 0。
- 后端 SHA256：`07e54357c16f40c97f8e7b96abe3c35653bcf9bdc46d3a5005e92af5d0e92cb5`。
- 前端 index SHA256：`7c9fcf65c3dd552c021bfde5323aaa6123816e51ade5e2daeb7b13d28448277d`。
- 发布目录：`/opt/v2ray-dash/releases/tls-fingerprint-20261103T051749Z`。
- 旧前后端、数据库备份及演练记录：`/var/backups/v2raydash/20261103T051749Z-tls-fingerprint`。
- 工作区保留此前已经上线但尚未提交的依赖补装改动；本次没有提交或推送 Git。

## 手机验证边界

用户已验证手动切换指纹恢复。平台下发的字段和导出结构已经验证，但尚未在真实 Shadowrocket 中验证导入后的实际采用情况。旧版客户端可能忽略节点指纹，本地全局强制设置也可能覆盖节点值；不能仅凭导出字段正确就声称所有手机自动修复。更新订阅后仍需核对故障手机是否实际使用 `Safari26_3`。

参考：[Mihomo TLS 配置](https://wiki.metacubex.one/en/config/proxies/tls/)、[Shadowrocket 官方更新记录](https://t.me/s/ShadowrocketNews?before=1654)、[Sub-Store Shadowrocket 导出实现](https://github.com/sub-store-org/Sub-Store/blob/master/backend/src/core/proxy-utils/producers/shadowrocket.js)。
