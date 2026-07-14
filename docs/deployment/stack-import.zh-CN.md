# 迁入现有 LiteLLM Stack

## 放置位置

将服务器上现有 `aethergate-litellm-stack` 中适合版本控制的文件复制到：

```text
aethergate/deploy/compose/core/
```

最终应直接得到：

```text
deploy/compose/core/compose.yaml
```

不要多嵌套一层 `aethergate-litellm-stack/`。

服务器正在运行的副本可以继续留在：

```text
/opt/aethergate
```

或现有：

```text
/opt/aethergate-litellm-stack
```

仓库目录是“经过评审、可以提交的配置源”；`/opt` 下的是“带有服务器密钥和数据的运行副本”。二者职责不同。

## 可以复制的文件

根据之前生成的 Stack，建议复制：

```text
compose.yaml
.env.example
init-env.sh
litellm-config.yaml
backup.sh
verify.sh
postgres/init/01-create-databases.sh
pgbouncer/pgbouncer.ini
```

原 Stack 的 `README.md` 中如果有专属命令，应合并进 `deploy/compose/core/README.md`，避免长期保留两份互相冲突的操作说明。

## 禁止复制到 Git 的内容

```text
.env
aethergate-backend.env
secrets/*
backups/*
logs/*
postgres_data/*
redis_data/*
数据库导出文件
TLS 私钥
真实模型供应商 API Key
```

`.gitignore` 只是最后一道保护，提交前仍需人工检查。

## Stack 只存在于服务器时

如果仓库也已经克隆到服务器，可以在确认路径后使用：

```bash
STACK_DIR=/opt/aethergate-litellm-stack
REPO_DIR=/path/to/aethergate

rsync -av \
  --exclude='.env' \
  --exclude='*.env' \
  --exclude='secrets/' \
  --exclude='backups/' \
  --exclude='logs/' \
  --exclude='postgres_data/' \
  --exclude='redis_data/' \
  --exclude='README.md' \
  "$STACK_DIR/" "$REPO_DIR/deploy/compose/core/"
```

因为 `*.env` 被整体排除，需要在人工确认 `.env.example` 只有占位值之后单独复制：

```bash
cp "$STACK_DIR/.env.example" "$REPO_DIR/deploy/compose/core/.env.example"
```

如果服务器没有 `rsync`，请只逐个复制上面的允许清单，不要把整个运行目录打包提交。

## 迁入后检查

在仓库根目录运行：

```bash
git status --short
git diff -- deploy/compose/core
```

提交前确认：

- `compose.yaml` 不包含明文密码；
- `.env.example` 只有占位值；
- 没有 `.env`、备份、日志、数据库卷或客户数据；
- 镜像使用固定版本或明确的版本变量；
- `docker compose config --quiet` 能找到所有引用文件；
- 脚本不会无意输出密钥。

## 后续维护方式

1. 在 `deploy/compose/core` 修改配置；
2. 评审并提交；
3. 更新服务器前先备份；
4. 将配置同步到 `/opt` 运行目录，但保留服务器的 `.env`、密钥、备份和数据卷；
5. 执行 Compose 配置检查和 `verify.sh`。

不要只在 `/opt` 中长期修改而不回写仓库，否则部署会逐渐失去可复现性。

