# 服务器基础部署

本文整理 AetherGate 第一阶段的服务器准备和基础设施运行方式。执行前，应先按[迁入现有 Stack](./stack-import.zh-CN.md)把实际配置整理到 `deploy/compose/core`。

## 第一阶段服务

```text
公网或受信网络
      │
      ▼
LiteLLM Proxy :4000
├── PostgreSQL / litellm
└── Redis

AetherGate API
      │
      ▼
PgBouncer
      │
      ▼
PostgreSQL / aethergate
```

第一阶段只部署：

- LiteLLM：模型路由、Virtual Key、网关限流和上游故障切换；
- PostgreSQL：LiteLLM 与 AetherGate 的事务数据；
- PgBouncer：AetherGate 普通运行流量的连接池；
- Redis：LiteLLM 以及后续有限的缓存与协调用途。

ClickHouse 和 OpenMeter 分别留到使用分析与计费阶段，不让早期开发环境一次承担全部复杂度。

## 数据库隔离

同一个 PostgreSQL 实例中创建两个数据库和两个用户：

```text
litellm
└── litellm_user

aethergate
└── aethergate_user
```

AetherGate Go API 只能访问 `aethergate` 数据库。LiteLLM 的 Key、模型、预算等操作通过其支持的 API 完成，不直接修改 LiteLLM 内部表。

## 连接方式

生成给 AetherGate 后端的环境文件应表达以下含义：

```env
DATABASE_URL=postgresql://aethergate_user:<password>@pgbouncer:5432/aethergate
DIRECT_URL=postgresql://aethergate_user:<password>@postgres:5432/aethergate
LITELLM_BASE_URL=http://litellm:4000
LITELLM_MASTER_KEY=<server-side-master-key>
AETHERGATE_VAULT_KEK=<32字节随机值的标准Base64>
```

- `DATABASE_URL`：日常查询，走 PgBouncer transaction 池；
- `DIRECT_URL`：数据库迁移等需要直连或会话语义的操作；
- `LITELLM_BASE_URL`：Compose 内部地址；
- `LITELLM_MASTER_KEY`：仅后端使用，禁止发送给浏览器。
- `AETHERGATE_VAULT_KEK`：持久化 Vault 写入使用的密钥加密密钥，必须是恰好 32 字节随机值的标准 Base64，仅由 Go API 和获准内部 Worker 加载。

LiteLLM 首次启动和数据库结构操作默认直连 PostgreSQL。只有在固定版本完成兼容性测试后，才考虑切换到 PgBouncer session 池。迁移不要走 transaction 池。

## 端口和安全边界

之前的开发 Stack 使用：

| 服务 | 开发绑定 | 要求 |
|---|---|---|
| LiteLLM | `0.0.0.0:4000` | 仅限临时公网调试，安全组限制来源 IP |
| PostgreSQL | `127.0.0.1:5433` | 仅服务器本机 |
| PgBouncer | `127.0.0.1:6432` | 仅服务器本机 |
| Redis | 不映射宿主机端口 | 仅 Compose 网络 |

容器之间使用 `postgres:5432`、`pgbouncer:5432`、`redis:6379`、`litellm:4000`，不使用宿主机映射端口。

临时调试时，云安全组和主机防火墙中的 TCP 4000 只能允许开发人员当前公网 IP，不能长期开放 `0.0.0.0/0`。

生产环境必须：

- 使用反向代理和可信 HTTPS 证书；
- 尽量停止直接发布应用容器端口；
- 在入口增加认证、限流、请求大小限制和超时；
- PostgreSQL、PgBouncer、Redis 始终保持私网访问。

## 服务器准备

推荐运行目录：

```bash
sudo install -d -m 0750 /opt/aethergate
sudo chown -R "$USER":"$USER" /opt/aethergate
```

检查依赖：

```bash
docker --version
docker compose version
openssl version
```

把仓库 `deploy/compose/core` 中经过评审的内容复制到 `/opt/aethergate`。服务器自己的 `.env`、自动生成环境、密钥和备份只保留在 `/opt/aethergate`。

## 首次初始化与启动

```bash
cd /opt/aethergate
chmod +x init-env.sh backup.sh verify.sh
./init-env.sh
```

初始化脚本应生成：

- PostgreSQL 管理和应用用户密码；
- LiteLLM Master Key 与 Salt Key；
- LiteLLM UI 登录凭据；
- AetherGate 后端连接环境。
- 32 字节 AetherGate Vault KEK，使用标准 Base64 编码并保存到服务器密钥边界。

LiteLLM Salt Key 用于保护已保存的供应商凭据，开始保存模型凭据后不要随意重新生成。
当前 AetherGate Vault 使用单一 `env-v1` 包封密钥。必须通过获准的密钥管理系统备份；Vault 已有数据后不得直接替换，否则在实现并验证多密钥重包流程前，已有数据密钥将无法解密。

启动前检查：

```bash
docker compose config --quiet
docker compose pull
docker compose up -d
```

查看状态和日志：

```bash
docker compose ps
docker compose logs --tail=200 litellm
docker compose logs --tail=200 postgres
docker compose logs --tail=200 pgbouncer
docker compose logs --tail=200 redis
```

如果实际 `compose.yaml` 服务名不同，以导入后的文件为准。

## 完整验证

```bash
./verify.sh
```

同时人工确认：

1. 所有容器正常且健康检查通过；
2. LiteLLM readiness 正常；
3. LiteLLM UI 只有允许来源能够访问；
4. `litellm` 与 `aethergate` 数据库存在并使用不同所有者；
5. AetherGate 用户可通过 PgBouncer 访问自己的数据库；
6. 外部不可信主机无法连接 PostgreSQL 和 PgBouncer；
7. 测试 Virtual Key 只能访问被允许的模型。

之前使用的地址是：

```text
http://<服务器公网IP>:4000/ui
http://<服务器公网IP>:4000/health/readiness
```

具体健康端点仍需以导入版本的 LiteLLM 和 `verify.sh` 为准。UI 能打开不等于整套服务已经验证完成。

## Go API 接入

Go API 与基础服务位于同一 Compose 网络时，使用生成的内部连接值，并通过 Compose `env_file` 或后续密钥机制加载。不要把这些值复制到 Next.js 客户端配置中。

LiteLLM 集成变量只配置在 Go API 服务端环境中：

```dotenv
LITELLM_BASE_URL=http://litellm:4000
LITELLM_MASTER_KEY=<仅服务端使用的-master-key>
```

`LITELLM_BASE_URL` 必须是绝对 HTTP(S) 地址，不能包含内嵌凭据、查询串或 fragment。若健康端点不要求认证，可以不设置 `LITELLM_MASTER_KEY`；一旦设置，它也只能保留在服务端，绝不能返回 Console。两个值都不得写入 `NEXT_PUBLIC_*` 变量。

Go API 启动后，先检查脱敏配置状态，再从 API 所在网络执行真实探测：

```bash
curl -fsS http://aethergate-api:8080/api/v1/integrations/litellm/status
curl -fsS -X POST http://aethergate-api:8080/api/v1/integrations/litellm/verify
```

验证端点只调用 LiteLLM 的 `/health/liveliness` 与 `/health/readiness`，拒绝重定向，限制并丢弃响应正文，只返回状态、延迟和标准化错误证据，不会访问 LiteLLM 数据库内部表。`overall: ready` 只代表服务健康门禁通过；上线前仍需测试真实流式请求、取消、Virtual Key 策略、路由、用量归属和 Provider 故障行为。

Go API 在开发电脑运行时，优先选择：

1. 把 API 放进服务器 Compose 网络；
2. 通过 SSH 隧道连接服务器本机 PgBouncer；
3. 使用只能通过 VPN 或可信网络访问的独立开发数据库。

Provider Health 执行需要额外的信任边界：

- Go 控制面 API 只持久化探测任务并接收聚合后的观测结果；
- 独立的 Provider Health Worker 从服务端密钥边界解析凭据，只对允许列表中的提供商发起探测，并记录结果；
- Console 永远不接收提供商密钥，也不直接发起提供商探测；
- 只有在配置 Worker 出站允许列表、超时、凭据访问、审计记录和重试上限后，才能启用自动探测调度。

Scheduled Reports 执行使用独立的数据与交付边界：

- Go 控制面 API 保存计划、按 IANA 时区计算下次运行时间，并写入运行队列记录；
- 独立的 Reports Worker 领取定时、手动或重试任务，读取已授权的分析数据，生成 CSV/XLSX/PDF 文件，保存制品元数据，并交付给获准的邮件或 Slack 收件人；
- 对象存储、SMTP 和 Slack 凭据只存在于服务端，不进入 Console 配置；
- 只有在租户范围查询、对象保留策略、必要的恶意内容检查、签名下载、收件人授权、幂等性和重试限制配置完成后，才能启用 Reports Worker。

外部通知投递使用独立的身份与出站边界：

- Go 控制面 API 始终先创建接收人范围内的收件箱记录，并根据已校验的个人偏好，只写入排队、延后或抑制状态的外部投递记录；
- 独立的 Notifications Worker 领取可执行记录，解析服务端批准的邮件、Slack、Teams 或 Webhook 连接器引用，执行幂等与重试限制，并写回投递结果证据；
- 静默时段与摘要投递时间使用接收人的 IANA 时区计算，持久化时间仍统一为 UTC；
- 连接器凭据与原始 Webhook 密钥不得进入 Console、通知偏好载荷或控制面日志；
- 只有在接收人授权、目标允许列表、密钥轮换、模板转义、速率限制、退信/失败处理、重放保护、保留策略和审计记录配置完成后，才能启用外部通知投递。

Enterprise Vault 使用独立的加密与解析边界：

- 每个密钥版本生成新的 256 位随机数据密钥；密钥值和数据密钥分别使用 AES-256-GCM 保护，并把租户、密钥 ID 和版本作为认证附加数据；
- Console 与公共 HTTP 响应只接收掩码元数据、指纹、版本、引用、轮换状态和访问证据，永不接收明文、密文、Nonce 或包封数据密钥；
- 明文解析只存在于 Go 内部服务方法。Worker 必须提供操作者、工作负载、用途、请求 ID 和来源 IP，并把成功、拒绝或失败追加到不可变访问证据；
- PostgreSQL 持久化写入在 `AETHERGATE_VAULT_KEK` 缺失或非法时关闭失败。KEK 不得进入 Git 或数据库备份，但必须在独立权限控制的获准密钥系统中备份；
- 当前单一 `env-v1` 包封密钥不等于完整 KMS/密钥环。在重包、双密钥读取、回滚和恢复演练完成前，不得原地轮换；
- Provider Health、Gateway、Webhook、Reports 和 Notifications Worker 只能解析明确授权范围的引用，且不得记录或持久化返回的明文；
- 身份/RBAC、真实 PostgreSQL、真实 Worker 解析、外部 KMS/密钥环、备份恢复、撤销传播和泄露响应演练全部通过前，不得把 Vault 标记为已验证。

审计证据使用不可变存储与特权 Worker 边界：

- Go 控制面把操作者、动作、资源、结果、风险、请求/IP 上下文以及变更前后状态追加到租户独立的 SHA-256 前向哈希链；
- PostgreSQL 应用角色不得拥有绕过 `audit_events` 防修改触发器的路径；对修改被拒或哈希链校验失败必须产生运维告警；
- 只有 Audit Export Worker 可以领取已接受的导出任务、生成 CSV/JSONL 对象、计算 SHA-256 文件校验和，并写回行数、大小、对象键及成功/失败证据；
- 导出存储桶必须私有、加密、按租户前缀隔离、配置生命周期，并使用短期授权访问；对象存储凭据不得进入 Console；
- 保留期与法律保全是策略记录。物理清理必须由经过评审的特权分区保留 Worker 或分区删除流程执行，记录决策，并确保法律保全期间永不删除证据；
- 修改结构或保留策略前必须备份审计分区、策略和导出证据，并在隔离环境恢复后重新校验哈希链；
- 在身份/RBAC、租户范围读取、对象保留、幂等与重试、监控及恢复演练通过前，不得启用导出或物理清理。

不要为了调试把 PostgreSQL 直接开放给整个互联网。

## 备份与恢复准备

升级镜像、修改结构或调整配置前：

```bash
cd /opt/aethergate
./backup.sh
```

随后确认：

- 新备份确实生成；
- 记录其中包含的数据库和配置；
- 备份存储位置受保护，必要时加密；
- 重要备份按策略复制到服务器之外；
- 定期在隔离环境进行恢复演练。

仅生成文件而没有恢复演练，不算可靠备份。导入实际 `backup.sh` 后，应根据它的输出格式补充精确恢复命令。

## 更新流程

1. 在 `deploy/compose/core` 评审并提交配置变化；
2. 阅读所有待升级镜像的发布说明；
3. 备份当前服务器；
4. 同步配置，保留服务器 `.env`、密钥、备份和数据卷；
5. 运行 `docker compose config --quiet`；
6. 拉取镜像并只重建需要更新的服务；
7. 执行 `./verify.sh` 并检查日志；
8. 保留旧配置和镜像标签用于回滚。

使用固定镜像版本或明确的版本变量，不要在正常运行的 Stack 中无计划地改成 `latest`。

## 故障排查顺序

1. `docker compose config`：环境变量、引用文件和 YAML；
2. `docker compose ps`：容器状态与健康检查；
3. 服务日志：启动、认证、迁移和数据库连接；
4. 容器内部网络：服务名解析和内部端口；
5. 宿主机监听与防火墙；
6. 云安全组的 TCP 4000 来源限制；
7. 数据库 URL 中的数据库、用户、主机、端口和密码编码。

不要通过关闭认证或扩大数据库公网暴露来绕过问题。

