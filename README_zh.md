# codetok

[![CI](https://github.com/miss-you/codetok/actions/workflows/ci.yml/badge.svg)](https://github.com/miss-you/codetok/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go&logoColor=white)](https://go.dev)

[English](README.md)

一个用于追踪和汇总 AI 编程 CLI 工具 token 用量的命令行工具。

已支持的 Provider：

- **Kimi CLI** — 解析 `~/.kimi/sessions/**/wire.jsonl`
- **Claude Code** — 解析 `~/.claude/projects/**/*.jsonl`（含流式去重）
- **Codex CLI** — 解析 `~/.codex/sessions/**/*.jsonl`

计划支持：

- OpenCode
- Cursor

## 安装

### 从源码安装

```bash
go install github.com/miss-you/codetok@latest
```

注意：Go module 路径区分大小写，必须使用全小写 `github.com/miss-you/codetok`。

### 本地构建

```bash
git clone https://github.com/miss-you/codetok.git
cd codetok
make build
# 二进制文件在 ./bin/codetok
```

### 下载预编译版本

前往 [Releases](https://github.com/miss-you/codetok/releases) 页面下载。

## 快速开始

```bash
# 按日查看 token 用量（默认最近 7 天，单位=k）
codetok daily

# 按会话查看 token 用量
codetok session

# 输出 JSON 格式
codetok daily --json

# 查看全量历史数据
codetok daily --all

# 使用自定义滚动窗口
codetok daily --days 30

# 使用原始整数（不做单位缩放）
codetok daily --unit raw

# 强制用百万单位展示
codetok daily --unit m

# 按日期范围筛选
codetok daily --since 2026-02-01 --until 2026-02-15

# 按 Provider 筛选
codetok daily --provider claude
codetok session --provider kimi
```

提示：如果你改了代码后直接运行 `./bin/codetok`，请先执行 `make build` 刷新二进制。

## 使用说明

### `codetok daily`

按日展示 token 用量汇总。
默认仅展示最近 7 天。
如需全量历史数据，使用 `--all`；如需精确时间范围，使用 `--since`/`--until`。
表格默认按 `k` 单位展示 token 列（`--unit k`）。
可通过 `--unit raw`/`k`/`m`/`g` 控制展示单位。
JSON 输出始终保留原始整数 token 值。

```
Date        Provider  Sessions  Input(k)  Output(k)  Cache Read(k)  Cache Create(k)  Total(k)
2026-02-07  kimi      5         109.82k  15.36k  632.98k     0.00k         758.16k
2026-02-08  claude    2         95.05k   7.01k   274.23k     0.00k         376.29k
2026-02-15  codex     21        938.57k  149.29k 7869.70k    0.00k         8957.55k
TOTAL                 49        2965.04k 369.85k 24638.67k   0.00k         27973.57k
```

参数：

| 参数 | 说明 |
|------|------|
| `--json` | 以 JSON 格式输出 |
| `--days` | 未设置 `--since`/`--until` 时的最近天数窗口（默认：`7`） |
| `--all` | 包含全部历史会话（不能与 `--days`、`--since`、`--until` 同时使用） |
| `--unit` | 表格 token 展示单位：`raw`、`k`、`m`、`g`（默认：`k`） |
| `--since` | 起始日期（格式：`2006-01-02`） |
| `--until` | 截止日期（格式：`2006-01-02`） |
| `--provider` | 按 Provider 筛选（如 `kimi`、`claude`、`codex`） |
| `--base-dir` | 自定义数据目录（所有 Provider 生效） |
| `--kimi-dir` | 自定义 Kimi CLI 数据目录 |
| `--claude-dir` | 自定义 Claude Code 数据目录 |
| `--codex-dir` | 自定义 Codex CLI 数据目录 |

常用组合：
- `codetok daily` — 最近 7 天，表格单位 `k`
- `codetok daily --unit raw` — 最近 7 天，显示原始整数 token 值
- `codetok daily --days 30 --unit m` — 最近 30 天，按百万单位展示
- `codetok daily --all --unit g` — 全量历史，按十亿单位展示

### `codetok session`

按会话展示 token 用量。

```
Date        Provider  Session                               Title                      Input     Output  Total
2026-02-13  kimi      75c64dba-5c10-4717-83cd-f3d33abc39bc  翻译文章...                 72405     6080    78485
2026-02-15  claude    01f3c3c6-a4df-4e2b-8249-ea045ab13f11  写文档...                   381667    28258   409925
TOTAL                                                                                  2965044   369854  27973571
```

参数：`--json`、`--since`、`--until`、`--provider`、`--base-dir`、`--kimi-dir`、`--claude-dir`、`--codex-dir`。

### `codetok version`

输出版本信息；当 commit hash 与构建时间可用时会一并显示。

## 工作原理

codetok 读取 AI 编程 CLI 工具存储在本地磁盘的会话数据。每个 Provider 有独立的解析器，理解对应工具的数据格式。所有会话文件通过有界 goroutine 并行解析（默认：`min(NumCPU, 8)`，可通过 `CODETOK_WORKERS` 环境变量配置）。

统计口径：
- token 用量通过聚合本地已有会话日志中的 token 计数字段得到。
- codetok 不调用各 Provider 的远程 API。
- 只有当前本地仍存在日志文件的会话才会被统计。

**Kimi CLI** — `~/.kimi/sessions/<工作目录hash>/<会话UUID>/wire.jsonl`
- 解析 `StatusUpdate` 事件中的 `token_usage` 字段

**Claude Code** — `~/.claude/projects/<项目slug>/<会话UUID>.jsonl`
- 解析 `assistant` 事件中的 `message.usage` 字段
- 使用 `messageId:requestId` 复合键对流式事件去重（保留最后一条）

**Codex CLI** — `~/.codex/sessions/YYYY/MM/DD/rollout-*.jsonl`
- 解析 `event_msg` 事件中 `payload.type="token_count"` 的记录
- 取最后一条累计 token 计数

## 项目结构

```
codetok/
├── main.go                 # 入口，通过 ldflags 注入版本信息
├── cmd/
│   ├── root.go             # Cobra 根命令
│   ├── daily.go            # codetok daily（多 Provider）
│   └── session.go          # codetok session（多 Provider）
├── provider/
│   ├── provider.go         # Provider 接口和数据类型
│   ├── registry.go         # Provider 自动注册（init()）
│   ├── parallel.go         # 有界并行解析工具
│   ├── kimi/
│   │   └── parser.go       # Kimi CLI wire.jsonl 解析器
│   ├── claude/
│   │   └── parser.go       # Claude Code JSONL 解析器（含去重）
│   └── codex/
│       └── parser.go       # Codex CLI JSONL 解析器
├── stats/
│   └── aggregator.go       # 按日聚合和日期过滤
├── e2e/                    # 端到端测试
├── Makefile                # 构建、测试、lint 目标
└── .github/workflows/      # CI 和发布工作流
```

## 开发

```bash
# 构建
make build

# 运行测试
make test

# 代码检查（需要 golangci-lint）
make lint

# 格式化代码
make fmt

# 整理依赖
make tidy

# 查看所有目标
make help
```

## 添加新的 Provider

1. 在 `provider/` 下创建新的包（如 `provider/myprovider/`）
2. 实现 `provider.Provider` 接口并通过 `init()` 自动注册：

```go
package myprovider

import "github.com/miss-you/codetok/provider"

func init() {
    provider.Register(&Provider{})
}

type Provider struct{}

func (p *Provider) Name() string { return "myprovider" }

func (p *Provider) CollectSessions(baseDir string) ([]provider.SessionInfo, error) {
    // 解析会话文件，使用 provider.ParseParallel 进行并行解析
    // ...
}
```

3. 在 `cmd/daily.go` 和 `cmd/session.go` 中添加空白导入：
   ```go
   _ "github.com/miss-you/codetok/provider/myprovider"
   ```
4. 如需要，添加 `--myprovider-dir` 参数

## 许可证

[MIT](LICENSE)
