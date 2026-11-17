# 2026-11-16 Reality ML-KEM 订阅兼容性修复

## 原因与改动

ZL-US（216.167.92.48）使用 legacy v2ray-agent 目录，但实际运行 Xray 26.11.9。平台此前依据 `Layout != "legacy"` 设置 Mihomo 的 `support-x25519mlkem768`，因而错误关闭此节点需要的密钥交换选项。

同一账号在本机 Mihomo v1.19.29 中对照测试：原订阅 TLS 握手失败；仅将 ML-KEM 开关设为 true 即可访问 Google，出口 IP 为 216.167.92.48。节点端账号、公钥、SNI、Short ID 与平台一致；公网 16132 由现有 dokodemo-door 转发到本机 Reality 45987，端口没有配错。

移除依赖安装目录的判断及重复配置字段，统一在共享 Reality 代理生成函数中显式保留 Chrome 的 ML-KEM key share。单账号导出和多节点订阅共用规则；Shadowrocket 仍移除 Mihomo 专用字段，VLESS 与 Sing-box 输出保持原样。本次未增加按版本探测或保存核心能力的机制，也未对所有历史 Xray 版本作兼容性承诺。

## 验证

- 后端全量测试、真实 SetupRoutes 与隔离 PostgreSQL schema 测试、go vet 通过。新增回归覆盖 legacy 布局下单账号及多节点两个实际导出接口。
- 生产数据库副本关闭自动同步演练，共核对 33 份订阅的四种格式，以及 5 个账号的三种导出格式，共 147 份输出。
- 38 份 Clash 输出中，34 份发生变化，共 66 个 ML-KEM 开关由 false 改为 true；所有其他代理字段、规则、策略组、凭据、节点命名及排序一致。其他格式内容哈希逐项一致。
- 发布后重新验证同一组 147 份输出及公网 HTTPS 订阅入口，差异与演练一致。
- 用正式订阅原始输出启动临时本机 Mihomo 实例，实测 DMIT-US、DMIT-YBF2、香港 → DMIT-US、香港 → DMIT-YBF2 和 ZL-US 五条路径，均成功访问 Google 并得到相应出口 IP。没有修改用户正在运行的 Clash Verge 配置。
- ZL-US 的 Xray 主进程仍为 PID 4618，Reality 配置哈希仍匹配原平台记录；未重启、改写或升级节点代理服务。

## 部署结果

- 仅更新控制中心后端，健康恢复约 5.48 秒；账号、订阅关联、路由、节点定义及前端保持不变。订阅地址无需更换，Clash Verge 更新订阅即可取得新字段。
- 后端 SHA256：`53f2c934868de251ccb9aed4ef672f99c9f5a7b11c9b9487b0d09d63ba03f280`。
- 发布目录（UTC 时间戳）：`/opt/v2ray-dash/releases/mlkem-layout-fix-20261117T070238Z`。
- 原后端、数据库备份和核对记录：`/var/backups/v2raydash/20261117T070238Z-mlkem-layout-fix`。切换脚本配置了失败自动恢复。
- 用户已在 Clash Verge 更新订阅后确认恢复。
