# 开发指南

本文按步骤说明如何在本地工作站运行 AetherGate 的 Console、API 及配套基础设施。这些部分如何组合，参见[仓库目录](../../README.zh-CN.md#仓库目录)。

本文档是公开的、受版本控制的内容（根目录 README 会直接链接到它）。请不要在其中写入真实的主机名、IP 地址或凭据——使用类似 `YOUR_SERVER_IP` 的占位符，真实值只保留在你自己、被 Git 忽略的 `.env` 文件里，具体做法见下文**『环境规则』**一节。

## 前置条件

| 工具 | 版本 | 检查方式 |
| --- | --- | --- |
| Node.js | 20 及以上 | `node -v` |
| npm | 11 及以上 | `npm -v` |
| Go | 1.26.4 及以上 | `go version` |
| Docker Desktop（含 Compose v2） | 最新版 | `docker compose version`（仅在需要下文**『可选本地基础设施 Stack』**时才用到） |

在 Windows 上，即便 Go 已经装好，直接执行 `go version` 也可能提示"无法识别"——如果 Go 的 `bin` 目录从未加入用户或系统 `PATH`（比如手动安装而非使用官方安装程序的默认选项，或者只在 IDE 里配置了 Go SDK 路径），就会出现这种情况。遇到这种情况：把 Go 的 `bin` 目录加入 `PATH`，或者每次都用 `go.exe` 的完整路径调用，或者直接依赖你 IDE 自己的 Go SDK 配置（见下文**『方式 B：GoLand Run Configuration』**），这种方式完全不依赖 `PATH`。

## 日常启动流程

如果你要接入的是共享的 PostgreSQL/LiteLLM Stack（比如[服务器基础部署](../deployment/server-foundation.zh-CN.md)中描述的那一套），就按这个流程操作。如果你想完全在自己电脑上运行，可以直接跳到**『不使用数据库的快速启动』**或**『可选本地基础设施 Stack』**。

### 1. 建立 SSH 隧道

这类部署默认不会把 PostgreSQL 和 PgBouncer 暴露到公网（见 [`deploy/compose/core/README.md`](../../deploy/compose/core/README.md)），所以需要通过隧道访问。开一个专门的 PowerShell 窗口，整个开发过程中保持运行：

```powershell
ssh -N `
  -p 22 `
  -o ExitOnForwardFailure=yes `
  -o ServerAliveInterval=30 `
  -L 6432:127.0.0.1:6432 `
  -L 5433:127.0.0.1:5433 `
  root@YOUR_SERVER_IP
```

把 `YOUR_SERVER_IP` 和登录用户换成你实际的目标服务器——这些信息只保存在你自己的笔记或密码管理器里，不要写进聊天记录或提交到仓库的文件中。`-N` 表示只建立隧道、不打开远程 Shell；`6432` 是 PgBouncer 的连接池端口，`5433` 是 PostgreSQL 的直连端口（见下一步）。这个窗口要一直开着，直到当天开发结束。

### 2. 执行数据库迁移（首次运行，以及每次新增迁移之后）

在仓库根目录执行：

```powershell
Get-Content apps/api/.env | ForEach-Object {
    if ($_ -match '^\s*([^#][^=]*)=(.*)$') {
        Set-Item "Env:$($matches[1].Trim())" $matches[2].Trim()
    }
}

$env:AETHERGATE_DATABASE_URL = $env:AETHERGATE_DIRECT_DATABASE_URL
go run ./apps/api/cmd/migrate
```

这段脚本先把 `apps/api/.env` 加载进当前 Shell，然后把迁移指向 PostgreSQL 的直连端口而不是连接池端口——迁移不应该走 PgBouncer 的 transaction 池。`AETHERGATE_DIRECT_DATABASE_URL` 本身不会被应用代码读取，它只是你在自己的 `.env` 里额外维护的一个便捷变量，专门配合这段脚本做替换用。成功时会打印：

```text
AetherGate database is up to date
```

只有新增迁移之后才需要重新执行这一步，不是每次启动都要跑。

### 3. 启动 Go API

#### 方式 A：命令行

在单独的终端里，从仓库根目录执行（先按上面的方式加载 `apps/api/.env`，但不要覆盖 `AETHERGATE_DATABASE_URL`）：

```powershell
go run ./apps/api/cmd/server
```

#### 方式 B：GoLand Run Configuration

1. 新建一个 **Go Build** 类型的 Run Configuration。
2. **Run kind** 选 Directory，**Directory** 填仓库根目录下的 `apps/api/cmd/server`。
3. **Working directory** 填仓库根目录。
4. 环境变量：如果你的 GoLand 版本有 `Environment files` 字段，直接指向 `apps/api/.env`；没有的话，就打开 `Environment variables` 编辑框，把该文件里非注释的内容粘贴进去。
5. 点击 Run 或 Debug。

不管用哪种方式，API 默认监听 `http://localhost:8080`。验证是否启动成功：

```powershell
Invoke-RestMethod http://localhost:8080/healthz
```

应该会返回 `status: ok`。这里要用连接池端口（`6432`），不要用迁移时的直连端口（`5433`）——正常运行的 API 应该始终走 PgBouncer。

### 4. 启动 Console

在另一个终端：

```powershell
npm install
npm run dev
```

（`npm install` 只在第一次执行，或者依赖变化之后才需要。）打开 `http://localhost:3000`——注意要用 `localhost`，不要用 `127.0.0.1`：API 目前的 CORS 策略只允许 `http://localhost:3000` 这一个来源。

### 当前运行状态一览

| 组件 | 运行方式 |
| --- | --- |
| SSH 隧道 | 保持第 1 步的 PowerShell 窗口运行 |
| Go API，`:8080` | GoLand Run/Debug，或 `go run` |
| Console，`:3000` | `npm run dev` |
| LiteLLM，`:4000` | 远程 Docker Stack |
| PostgreSQL / PgBouncer | 通过 SSH 隧道访问 |

## 不使用数据库的快速启动

如果只是想快速改改 Console 或 API、不需要持久化，可以完全跳过隧道和迁移：

```powershell
npm install
npm run dev
```

```powershell
go run ./apps/api/cmd/server
```

不设置 `AETHERGATE_DATABASE_URL` 时，服务器会打印 `using development memory repository`，所有状态都只保存在进程内存里——重启后不会保留。

## API 参考：接口与环境变量

核心接口：

- `GET /healthz`
- `GET /readyz`
- `GET /api/v1/overview`
- `GET /api/v1/requests`
- `GET /api/v1/requests/{requestID}`

API 还提供企业、API Key、工作空间、项目、成员、模型、提供商健康、路由、限流、预算、告警、Webhooks、报表、通知、审计和 Vault 相关接口——完整列表见 [`apps/api/README.md`](../../apps/api/README.md)。

| 变量 | 用途 |
| --- | --- |
| `AETHERGATE_API_ADDR` | HTTP 监听地址，默认 `:8080`。 |
| `AETHERGATE_DATABASE_URL` | PostgreSQL/PgBouncer 连接串。留空则使用内存开发存储。 |
| `AETHERGATE_AUTO_MIGRATE` | 只在可控的开发环境中设为 `true`，用于启动时自动迁移。优先使用显式的 `migrate` 命令。 |
| `AETHERGATE_VAULT_KEK` | 持久化 Vault 写入所需：标准 Base64 编码、恰好 32 字节随机值，每个环境独立生成。不要暴露给 Console，也不要提交到仓库。 |
| `LITELLM_BASE_URL` | LiteLLM 的内部绝对 HTTP(S) 地址，比如通过隧道或本地 Stack 访问时用 `http://127.0.0.1:4000`。 |
| `LITELLM_MASTER_KEY` | 可选的服务端专用 Bearer 凭据，用于 LiteLLM 健康探测。不要通过 `NEXT_PUBLIC_*`、日志或 Git 暴露。 |

Console 自己的 API 地址配置在 `apps/console/.env` 中：

```text
NEXT_PUBLIC_AETHERGATE_API_URL=http://localhost:8080/api/v1
```

这也是内置默认值，只有需要让 Console 指向别的 API 实例时才需要创建这个文件。

`apps/worker` 目前还没有可运行的入口；它现在只在 [`apps/worker/README.md`](../../apps/worker/README.md) 中记录了规划中的职责。

## 可选本地基础设施 Stack

[`deploy/compose/core`](../../deploy/compose/core/README.md) 是 LiteLLM、PostgreSQL 17、PgBouncer 和 Redis 这套 Stack 的评审后源文件，[服务器基础部署](../deployment/server-foundation.zh-CN.md)用的也是同一份。同一个 Compose 文件也可以直接在工作站上运行，而不是部署到远程服务器——这种情况下可以跳过该指南里 `/opt` 上传和 SSH 隧道相关的步骤，那些只适用于远程场景。

`init-env.sh`、`backup.sh`、`verify.sh` 都是 bash 脚本。在 Windows 上需要用 Git Bash（随 Git for Windows 安装）或 WSL 执行，不能直接在 PowerShell 里运行。

```bash
cd deploy/compose/core
./init-env.sh          # 生成 .env、aethergate-backend.env、secrets/pgbouncer_users.txt，全部使用随机凭据
docker compose config --quiet
docker compose pull
docker compose up -d
```

想自己填值而不是用 `init-env.sh` 生成？把 `.env.example` 复制成 `.env`，然后把每个 `CHANGE_ME` 替换掉即可。

本地端口（生成的 `.env` 默认值）：

- LiteLLM：`http://localhost:4000`（UI 在 `/ui`；`UI_USERNAME` / `UI_PASSWORD` 在生成的 `.env` 里）
- PostgreSQL（直连）：`127.0.0.1:5433`
- PgBouncer：`127.0.0.1:6432`

让 API 连接这套 Stack 时，要用这些宿主机映射端口，而不是生成的 `aethergate-backend.env` 里那些 Docker 内部主机名（`pgbouncer`、`postgres`、`litellm` 只有容器加入同一个 Compose 网络时才能解析）。

不要提交生成出来的 `.env`、`aethergate-backend.env` 或 `secrets/pgbouncer_users.txt`——这三个文件都已经在 `.gitignore` 里，必须始终保持这样。

## 提交前验证

打开 Pull Request 之前执行：

```powershell
npm run typecheck
npm run lint
npm run build
go test ./apps/api/...
go vet ./apps/api/...
```

## 环境规则

- 把本地值保存在各自应用旁边、被忽略的 `.env` 文件中（`apps/console/.env`、`apps/api/.env`、`deploy/compose/core/.env`）。
- 目前只有 `deploy/compose/core` 提供了已提交的 `.env.example`。`apps/console` 和 `apps/api` 还没有，所以在示例文件补齐之前，直接按上面的表格设置这两处的变量。
- 不要把生产环境的密钥、Master Key 或数据库导出复制进本地配置，也不要把真实的主机名、IP 地址或凭据贴进聊天记录、Issue 或提交到仓库的文件里。如果某个真实密钥曾经这样完整暴露过，就当它已经泄露，尽快安排轮换。
- 使用独立的开发数据库和 LiteLLM Master Key；本地开发环境不要指向生产 Stack。
- 优先使用 `deploy/compose/core` 中评审过的 Compose 源文件，不要临时拼凑容器。

## 前端

Console 使用 HeroUI v3、React 19+ 和 Tailwind CSS v4。交互与复杂表格的边界见 [`apps/console/README.md`](../../apps/console/README.md)。

## 后端

API 和 Worker 使用 Go，位于 `github.com/topoai/aethergate` module 下。领域边界见 [`apps/api/README.md`](../../apps/api/README.md) 和 [`apps/worker/README.md`](../../apps/worker/README.md)。

## 文档是开发流程的一部分

配置、端口、环境变量、迁移、服务归属或运行行为发生变化时，必须在同一个 Pull Request 中同步更新对应文档。
