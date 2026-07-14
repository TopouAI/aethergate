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
```

- `DATABASE_URL`：日常查询，走 PgBouncer transaction 池；
- `DIRECT_URL`：数据库迁移等需要直连或会话语义的操作；
- `LITELLM_BASE_URL`：Compose 内部地址；
- `LITELLM_MASTER_KEY`：仅后端使用，禁止发送给浏览器。

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

LiteLLM Salt Key 用于保护已保存的供应商凭据，开始保存模型凭据后不要随意重新生成。

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

Go API 在开发电脑运行时，优先选择：

1. 把 API 放进服务器 Compose 网络；
2. 通过 SSH 隧道连接服务器本机 PgBouncer；
3. 使用只能通过 VPN 或可信网络访问的独立开发数据库。

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

