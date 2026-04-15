---
name: compatibility-first-planning
description: Use when a broad codetok change needs design work before implementation or task breakdown.
---

# Compatibility-First Planning

## Overview

把一个宽泛的 `codetok` 目标收敛成可实施设计，同时先冻结用户可见契约，再讨论抽象。

这个 skill 只负责设计，不负责实现。设计批准后，下一步是 `deriving-task-board-from-design`。

## When to Use

在这些场景使用：

- 用户说的是“做一个新功能 / 重构 / 重写 / 调整输出”
- 变更会影响 `cmd/`、`provider/`、`stats/`、`cursor/`、`e2e/`、CLI flags、stdout/stderr 或退出码
- 你还不确定哪些 token 统计、文件读取、分组或输出行为必须稳定
- 你已经看到自己快开始抽象，但还没冻结边界

这些场景不要使用：

- 小型 bugfix
- 单文件改动且验收标准已经非常明确
- 设计已经批准，当前只差生成任务板或落代码

## SOP

对每个 planning artifact，都按同一循环执行：

1. 先定义这个 section 要解决什么问题
2. 基于仓库证据写出当前版本
3. 冻结用户可见契约，特别是 CLI 命令、flags、输出形状、错误语义和 token 统计语义
4. 写清 `cmd/`、`provider/`、`stats/`、`cursor/` 和 `e2e/` 的边界
5. 用 `make fmt`、`make vet`、`make test`、`make build` 作为默认验证骨架，`make lint` 仅在可用时补充
6. 重新通读，直到它足以指导实现

不要在关键设计缺口仍然存在时进入实现。

## Required Outputs

设计结果必须显式包含：

1. Goal and non-goals
2. Current project inventory
3. Compatibility contract
4. Terminology mapping
5. Core vs shell boundary
6. Phase plan
7. Rough task breakdown
8. Verification plan
9. Evidence references

缺一项都不算 implementation-ready。

设计必须落盘到 `docs/plans/YYYY-MM-DD-<topic>-design.md`。
下游的任务板 skill 依赖这份文件；只有聊天里的草稿或口头确认不算完成。

## The Flow

### 1. Clarify the target

先把“成功”定义成用户可见行为，而不是包结构。

优先冻结这些面：

- 本次是否只读取本地磁盘上的现有 session/log 文件，不调用任何远程 API
- token 统计、分组维度和汇总口径是否保持语义稳定
- CLI 命令、flags、stdout/stderr、exit code 是否要保持稳定
- 输出展示、筛选、范围选择、单位切换和 e2e 覆盖哪些是 v1 范围

### 2. Inventory the current system

在提架构前，先确认当前仓库事实：

- `cmd/` 里的 Cobra 命令面，通常包括 `daily`、`session`、`cursor`、`version`
- `provider/` 的解析器、注册和并发读取逻辑
- `stats/` 的日期过滤、汇总和日级聚合逻辑
- `cursor/` 的本地 activity、auth、sync 和 storage 边界
- `e2e/` 里已有的 CLI 端到端验证
- `docs/plans/` 里的已有设计约束
- `Makefile` 里的 `make fmt`、`make vet`、`make test`、`make build`、`make lint` 到底证明了什么

结论必须尽量绑定到具体文件，而不是凭聊天记忆。

### 3. Freeze compatibility contracts

把外部契约写死，常见包括：

- session 发现规则与文件存在性约束
- provider 解析输出的字段和单位语义
- daily/session/version 的命令和 flags
- `daily`、`session`、`cursor activity` 只读本地文件；只有 `cursor login/status/sync` 可以接触远程 Cursor API
- token 统计的分组、排序、滚动窗口和范围语义
- 若存在 JSON 或其他导出能力，其 payload 结构

不要把兼容性写成 “后面实现时再看”。

### 4. Write terminology mapping

显式写清这些术语：

- 什么是统计真相源
- 什么是 CLI shell
- 哪些词指 provider，哪些词指 session，哪些词指 day-level stats
- 哪些词不能为了“听起来通用”而被过度泛化
- 哪些词必须和仓库现有命令名一致

### 5. Draw the architectural boundary

`codetok` 的默认边界是：

- core: `provider` 解析、`stats` 聚合、token 口径和数据模型
- shell: Cobra 命令、参数解析、文件扫描、Cursor 显式同步命令、终端输出、e2e 入口

只抽象当前项目已经证明共通的部分。

### 6. Phase the work around closed loops

第一阶段必须闭合真实用户路径，推荐优先级：

1. 读取一个本地 session 文件或日志文件
2. 解析出可验证的 token 统计
3. 在 `daily` 或 `session` 命令中输出结果
4. 用 `make test` 和最小 CLI smoke 证明结果稳定
5. 正常退出

不要把第一阶段写成 “先做通用框架”。

### 7. Define checks before implementation

每个设计至少要回答：

- `Scope`: v1 到底覆盖哪条完整路径
- `Contract`: 哪些用户面必须稳定
- `Boundary`: 哪些抽象是现在必须的
- `Operational`: 是否能跑通一个真实闭环
- `Data`: session/provider/stats contract 如何被证明
- `Maintenance`: 维护者能否快速理解

### 8. Hand off

只有当设计已经写入 `docs/plans/YYYY-MM-DD-<topic>-design.md`，并且被明确批准后，才进入 `deriving-task-board-from-design` 生成执行任务板。

## Quick Reference

| Artifact | Question it answers |
| --- | --- |
| Goal and non-goals | 这次到底做什么，不做什么？ |
| Inventory | 当前仓库已经有什么？ |
| Compatibility contract | 哪些用户行为不能被偷偷改掉？ |
| Terminology mapping | 哪些词必须稳定，哪些词只是内部实现？ |
| Core vs shell | 统计真相和 CLI 适配边界怎么分？ |
| Phase plan | 先闭合哪条真实路径？ |
| Verification plan | 用什么证据证明设计成立？ |

## Default Checks

- `Scope`: 是否只覆盖一个清晰的用户路径
- `Session`: 是否只依赖本地磁盘上现有文件
- `Provider`: 是否冻结了解析和分组语义
- `CLI`: 是否覆盖命令、flags、输出、错误语义
- `Boundary`: 是否把 shell 和统计核心分开
- `Maintenance`: 是否避免为了未来场景过度抽象

## Red Flags

- “先做通用 provider/runtime 抽象，以后方便扩展”
- “token 口径后面再补”
- “本地 session 结构很简单，不用先冻结”
- “先按感觉拆 package，契约边走边看”
- “CLI 输出只是皮肤，不算兼容面”

## Common Mistakes

- 从宽泛目标直接跳到包结构
- 把 session/provider/stats 语义当成实现细节
- 为未来运行时、插件、provider 提前抽象
- 没有证据引用，只写主观判断
- 写完设计却没有清楚下一步任务拆解入口

## Output Standard

好的规划结果应该让读者立刻知道：

- 当前 v1 目标是什么
- 哪些内容明确不做
- 哪些 session/provider/stats/cursor/CLI 面必须稳定
- core 和 shell 怎么分
- 第一条闭环要怎么交付
- 设计文档最终落在哪个 `docs/plans/*-design.md` 路径
- 下一步如何拆成可认领任务

## Acceptance Criteria

- 设计文档已落到 `docs/plans/YYYY-MM-DD-<topic>-design.md`
- 设计显式覆盖 goal/non-goals、仓库现状、兼容契约、术语、core/shell 边界、阶段计划、粗任务拆分、验证计划和证据引用
- 外部契约覆盖本地 session/provider/stats/cursor/CLI 行为，且没有把远程 API 调用混入 `daily`、`session` 或 `cursor activity`
- 验证计划包含 `make fmt`、`make vet`、`make test`、`make build`，并说明 `make lint` 是否可用
- 设计已经足以交给 `deriving-task-board-from-design` 创建任务板

## Next Step

设计批准后，使用 `deriving-task-board-from-design` 创建或刷新对应的 `docs/plans/*-design-task.md`。
