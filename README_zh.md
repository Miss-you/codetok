# codetok

[![CI](https://github.com/miss-you/codetok/actions/workflows/ci.yml/badge.svg)](https://github.com/miss-you/codetok/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go&logoColor=white)](https://go.dev)

[English](README.md)

一个用于追踪和汇总 AI 编程 CLI 工具本地 token usage events 的命令行工具。

已支持的 Provider：

- **Kimi CLI** — 解析 `~/.kimi/sessions/**/wire.jsonl`
- **Claude Code** — 解析 `~/.claude/projects/**/*.jsonl`（含流式去重）
- **Codex CLI** — 设置 `CODEX_HOME` 时解析 `$CODEX_HOME/sessions/**/*.jsonl`，否则解析 `~/.codex/sessions/**/*.jsonl`
- **Cursor** — 解析 `~/.codetok/cursor/*.csv`、`~/.codetok/cursor/imports/**/*.csv` 和 `~/.codetok/cursor/synced/**/*.csv` 下的本地 Cursor 用量导出文件

计划支持：

- OpenCode

## 安装

### 从 npm 安装

```bash
npm install -g @y0usali/codetok
```

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

发布自动化：
- 推送 `v*` tag 后会发布 GitHub Release 二进制。
- 同一个 workflow 会继续自动发布 npm 包。
- 仓库维护者需要在 GitHub Actions Secrets 中配置 `NPM_TOKEN`。

## 快速开始

```bash
# 按日查看 token 用量（默认最近 7 天，按 CLI/Provider 分组，单位=m）
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

# 指定日期筛选和按日分桶使用的时区
codetok daily --since 2026-02-01 --until 2026-02-15 --timezone Asia/Shanghai

# 按 Provider 筛选
codetok daily --provider claude
codetok session --provider kimi

# 只从自定义本地目录读取 Cursor 导出的 CSV
codetok daily --all --cursor-dir ~/Downloads/cursor-usage

# 查看本地 Cursor 活动归因（accepted lines，不是 token）
codetok cursor activity

# 以 JSON 查看本地 Cursor 活动归因，并指定 SQLite 路径
codetok cursor activity --json --db-path ~/.cursor/ai-tracking/ai-code-tracking.db

# 切换到按模型聚合（需要显式开启）
codetok daily --group-by model

# Share 区域展示 Top 10 分组
codetok daily --top 10
```

提示：如果你改了代码后直接运行 `./bin/codetok`，请先执行 `make build` 刷新二进制。

Cursor 命令边界：
- `daily`、`session` 和 `cursor activity` 只读取本地文件。
- `cursor login`、`cursor status`、`cursor sync` 是唯一可能显式访问 Cursor 远程 API 的命令。
- 一旦设置 `--cursor-dir`，只会扫描你提供的目录。

## 使用说明

### `codetok daily`

按日展示 token 用量汇总。
默认展示最近 7 天，并按 CLI/Provider 聚合（`--group-by cli`）。
使用 `--group-by model` 可以切换到模型维度聚合。
`daily` 按每条 usage event 的时间戳归属日期，因此同一个长会话可以把 token 分摊到多个自然日。
如需全量历史数据，使用 `--all`；如需精确时间范围，使用 `--since`/`--until`。
日期筛选和按日分桶默认使用本地时区；可用 `--timezone IANA/Name` 指定其他时区。
表格默认按 `m` 单位展示 token 列（`--unit m`）。
可通过 `--unit raw`/`k`/`m`/`g` 控制展示单位。
JSON 输出始终保留原始整数 token 值。
JSON 输出中，当前聚合维度和值由 `group_by` 与 `group` 描述。
`provider` 仅在该统计组可唯一对应到单个 Provider 时有值；当非 Provider 聚合维度横跨多个 Provider 时，`provider` 可能为空，`providers` 会列出贡献的 Provider。
`Sessions` 表示当天/该分组中贡献 usage events 的不同会话数。
可用 `--top N` 控制 share 区域展示多少个分组。
Cursor 在这个命令里仍然是本地读取：默认会扫描 `~/.codetok/cursor/` 下的根目录历史 CSV，以及 `imports/`、`synced/` 子目录；不会隐式触发 sync。

```
Daily Total Trend
Date   02-15   02-16   02-17   ...
Total  20.32m  8.47m   66.43m  ...
Bar    ###...  #.....  ######  ...

CLI Total Ranking
Rank  CLI     Sessions  Total(m)
1     claude  23        102.12m
2     codex   31        100.83m
3     kimi    41        26.78m

Top 5 CLI Share
Rank  CLI     Share   Sessions  Total(m)  Input(m)  Output(m)  Cache Read(m)  Cache Create(m)
1     claude  43.81%  23        102.12m   0.02m     0.50m      98.21m         3.39m
```

参数：

| 参数 | 说明 |
|------|------|
| `--json` | 以 JSON 格式输出 |
| `--days` | 未设置 `--since`/`--until` 时的最近天数窗口（默认：`7`） |
| `--all` | 包含全部历史会话（不能与 `--days`、`--since`、`--until` 同时使用） |
| `--unit` | 表格 token 展示单位：`raw`、`k`、`m`、`g`（默认：`m`） |
| `--group-by` | `daily` 聚合维度：`cli`（默认，Provider/CLI 视图）或 `model`（显式开启） |
| `--top` | 当前聚合维度下 share 区域展示的分组数量（默认：`5`） |
| `--since` | 起始日期（格式：`2006-01-02`） |
| `--until` | 截止日期（格式：`2006-01-02`） |
| `--timezone` | 日期筛选和按日分桶使用的时区；接受 IANA 名称，默认使用本地时区 |
| `--provider` | 按 Provider 筛选（如 `kimi`、`claude`、`codex`） |
| `--base-dir` | 自定义数据目录（所有 Provider 生效） |
| `--kimi-dir` | 自定义 Kimi CLI 数据目录 |
| `--claude-dir` | 自定义 Claude Code 数据目录 |
| `--codex-dir` | 自定义 Codex CLI 数据目录 |
| `--cursor-dir` | 自定义 Cursor CSV 目录；只扫描你提供的本地路径 |

常用组合：
- `codetok daily` — 最近 7 天，按 CLI/Provider 分组，表格单位 `m`
- `codetok daily --unit raw` — 最近 7 天，显示原始整数 token 值
- `codetok daily --days 30 --unit m` — 最近 30 天，按百万单位展示
- `codetok daily --all --unit g` — 全量历史，按十亿单位展示
- `codetok daily --group-by model` — 切换到模型维度聚合（显式开启）
- `codetok daily --top 10` — share 区域展示 Top 10 分组
- `codetok daily --timezone Asia/Shanghai` — 使用 Asia/Shanghai 解释事件日期

### `codetok session`

按会话展示 token 用量。
`session` 会先用 `--since`/`--until` 按 usage event 日期筛选，再把命中的 events 按会话聚合；因此只要会话在范围内有 token usage，即使它更早开始也会出现在结果中。
表格中的 `Date` 是该会话在所选时区内第一条命中的 usage event 日期。
Cursor 在这个命令里仍然是本地读取：默认会扫描 `~/.codetok/cursor/` 下的根目录历史 CSV，以及 `imports/`、`synced/` 子目录；不会隐式触发 sync。

```
Date        Provider  Session                               Title                      Input     Output  Total
2026-02-13  kimi      75c64dba-5c10-4717-83cd-f3d33abc39bc  翻译文章...                 72405     6080    78485
2026-02-15  claude    01f3c3c6-a4df-4e2b-8249-ea045ab13f11  写文档...                   381667    28258   409925
TOTAL                                                                                  2965044   369854  27973571
```

参数：`--json`、`--since`、`--until`、`--timezone`、`--provider`、`--base-dir`、`--kimi-dir`、`--claude-dir`、`--codex-dir`、`--cursor-dir`。
`--timezone` 接受 IANA 时区名称，默认使用本地时区。
设置 `--cursor-dir` 后，只会扫描该本地目录。

### `codetok version`

输出版本信息；当 commit hash 与构建时间可用时会一并显示。

### `codetok cursor activity`

读取本地 `~/.cursor/ai-tracking/ai-code-tracking.db`，展示 Cursor 的活动归因。
该命令会分别输出 `composer` 与 `tab` 的 accepted-line 指标。
它是独立的本地 activity 视图，不属于 token 统计。

参数：`--json`、`--db-path`。

## 工作原理

codetok 读取本地磁盘上的会话数据和用量导出文件。Provider 会先把这些本地记录转换为带时间戳的 usage events，再由命令层做聚合。JSONL 会话文件通过有界 goroutine 并行解析（默认：`min(NumCPU, 8)`，可通过 `CODETOK_WORKERS` 环境变量配置）；Cursor CSV 文件从本地目录发现后按文件顺序解析。

统计口径：
- token 用量通过聚合本地会话日志和本地 CSV 导出中的 token events 得到。
- `daily` 在所选时区下按 usage event 日期聚合。
- `session` 先按 usage event 日期筛选，再把命中的 events 按 Provider/会话聚合。
- `daily` 与 `session` 不会调用各 Provider 的远程 API。
- `codetok cursor login`、`status`、`sync` 是唯一会显式访问 Cursor 远程 API 的命令。
- 只有当前本地仍存在日志文件的会话才会被统计。

**Kimi CLI** — `~/.kimi/sessions/<工作目录hash>/<会话UUID>/wire.jsonl`
- 解析 `StatusUpdate` 事件中的 `token_usage` 字段

**Claude Code** — `~/.claude/projects/<项目slug>/<会话UUID>.jsonl`
- 解析 `assistant` 事件中的 `message.usage` 字段
- 使用 `messageId:requestId` 复合键对流式事件去重（保留最后一条）

**Codex CLI** — `$CODEX_HOME/sessions/YYYY/MM/DD/rollout-*.jsonl`；未设置 `CODEX_HOME` 时回退到 `~/.codex/sessions/YYYY/MM/DD/rollout-*.jsonl`
- 解析 `event_msg` 事件中 `payload.type="token_count"` 的记录
- 优先使用 `last_token_usage`
- 将累计的 `total_token_usage` 转换为每条 event 的增量 token

**Cursor** — `~/.codetok/cursor/*.csv`、`~/.codetok/cursor/imports/**/*.csv`、`~/.codetok/cursor/synced/**/*.csv`
- 解析本地保存的 Cursor Dashboard CSV 文件
- 默认会合并历史平铺 CSV、手工导入 CSV 和 sync 缓存 CSV
- `daily` 与 `session` 不会隐式触发 Cursor sync 或远程 API 访问
- 将 `Input (w/o Cache Write)`、`Input (w/ Cache Write)`、`Cache Read`、`Output Tokens` 映射到 `codetok` 的 token 字段
- 每一行 CSV 视为一条本地 usage 记录，用于 session/day 视图
- 暂不支持 Cursor Tab token 统计，因为导出数据没有提供可信的 Tab token 拆分
- 可通过 `codetok cursor activity` 读取 `~/.cursor/ai-tracking/ai-code-tracking.db` 中独立的本地 activity 归因数据

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
│   ├── cursor/
│   │   └── parser.go       # Cursor 用量 CSV 解析器
│   └── codex/
│       └── parser.go       # Codex CLI JSONL 解析器
├── stats/
│   ├── aggregator.go       # 旧 session 聚合辅助逻辑
│   └── events.go           # 基于 usage events 的按日聚合和日期过滤
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
