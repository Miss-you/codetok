# codetok

[![CI](https://github.com/Miss-you/codetok/actions/workflows/ci.yml/badge.svg)](https://github.com/Miss-you/codetok/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white)](https://go.dev)

[English](README.md)

一个用于追踪和汇总 AI 编程 CLI 工具 token 用量的命令行工具。

当前支持：

- **Kimi CLI** — 解析 `~/.kimi/sessions/**/wire.jsonl`

计划支持：

- Claude Code
- OpenCode
- Codex CLI
- Cursor

## 安装

### 从源码安装

```bash
go install github.com/Miss-you/codetok@latest
```

### 本地构建

```bash
git clone https://github.com/Miss-you/codetok.git
cd codetok
make build
# 二进制文件在 ./bin/codetok
```

### 下载预编译版本

前往 [Releases](https://github.com/Miss-you/codetok/releases) 页面下载。

## 快速开始

```bash
# 按日查看 token 用量
codetok daily

# 按会话查看 token 用量
codetok session

# 输出 JSON 格式
codetok daily --json

# 按日期范围筛选
codetok daily --since 2026-02-01 --until 2026-02-15
```

## 使用说明

### `codetok daily`

按日展示 token 用量汇总。

```
Date        Sessions  Input    Output  Cache Read  Cache Create  Total
2026-02-07  5         109822   15356   632985      0             758163
2026-02-08  2         95046    7010    274232      0             376288
2026-02-15  21        938566   149287  7869696     0             8957549
TOTAL       49        2965044  369854  24638673    0             27973571
```

参数：

| 参数 | 说明 |
|------|------|
| `--json` | 以 JSON 格式输出 |
| `--since` | 起始日期（格式：`2006-01-02`） |
| `--until` | 截止日期（格式：`2006-01-02`） |
| `--base-dir` | 自定义 Kimi 数据目录 |

### `codetok session`

按会话展示 token 用量。

```
Date        Session                               Title                      Input     Output  Total
2026-02-13  75c64dba-5c10-4717-83cd-f3d33abc39bc  翻译文章...                 72405     6080    78485
2026-02-15  01f3c3c6-a4df-4e2b-8249-ea045ab13f11  写文档...                   381667    28258   409925
TOTAL                                                                        2965044   369854  27973571
```

参数：与 `codetok daily` 相同。

### `codetok version`

输出构建版本号、commit hash 和构建时间。

## 工作原理

codetok 读取 AI 编程 CLI 工具存储在本地磁盘的会话数据：

**Kimi CLI** 的会话数据位于 `~/.kimi/sessions/<工作目录hash>/<会话UUID>/`：

- `wire.jsonl` — 事件流，其中 `StatusUpdate` 事件包含 `token_usage` 字段
- `metadata.json` — 会话标题和 ID

codetok 扫描所有会话目录，从 `StatusUpdate` 事件中提取 token 计数，按日或按会话进行汇总。

## 项目结构

```
codetok/
├── main.go                 # 入口，通过 ldflags 注入版本信息
├── cmd/
│   ├── root.go             # Cobra 根命令
│   ├── daily.go            # codetok daily
│   └── session.go          # codetok session
├── provider/
│   ├── provider.go         # Provider 接口和数据类型
│   └── kimi/
│       └── parser.go       # Kimi CLI wire.jsonl 解析器
├── stats/
│   └── aggregator.go       # 按日聚合和日期过滤
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

1. 在 `provider/` 下创建新的包（如 `provider/claude/`）
2. 实现 `provider.Provider` 接口：

```go
type Provider interface {
    Name() string
    CollectSessions(baseDir string) ([]SessionInfo, error)
}
```

3. 在 `cmd/daily.go` 和 `cmd/session.go` 中接入新 Provider

## 许可证

[MIT](LICENSE)
