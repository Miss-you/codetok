---
name: breaking-design-into-tasks
description: Use when an approved codetok design needs to be split into a claimable task board.
---

# Breaking Design Into Tasks

## Overview

这是一个 alias 入口，直接转给 `deriving-task-board-from-design`。
不要在这里维护第二套任务板流程。

## When to Use

在这些场景使用：

- 用户明确说的是“拆任务 / break down the design / 先出 task board”
- 已有批准的 `docs/plans/*-design.md`
- 设计需要转成能在 `codetok` 里认领的任务板，方便并行实现 `cmd/`、`provider/`、`stats/`、`cursor/`、`e2e/`

这些场景不要使用：

- 设计还在漂移
- 你已经在交付某个具体任务
- 你已经确定应该直接使用 `deriving-task-board-from-design`

## SOP

1. 确认源设计已批准，且目标是 `codetok` 的 Go CLI 工作
2. 直接调用 `deriving-task-board-from-design`
3. 让任务板围绕可验证闭环拆分，而不是按目录机械拆分
4. 优先把 `cmd/`、`provider/`、`stats/`、`cursor/`、`e2e/`、文档和 Makefile gate 拆成可并行任务
5. 复核任务板是否能直接支持认领、并行和验收

## Acceptance Criteria

- 任务板对应一个已批准的 `docs/plans/*-design.md`
- 任务 ID 稳定，依赖关系显式，`Done When` 可检查
- 任务板能让后续任务安全认领，不依赖聊天记忆
- 输出里没有遗留的源仓库术语，验证点对齐 `make fmt`、`make vet`、`make test`、`make build`，必要时补 `make lint` 和 CLI smoke
- 若设计影响行为，任务板里能看出哪些任务会碰 `cmd/`、`provider/`、`stats/`、`cursor/` 或 `e2e/`

## Guardrails

- 不要发明第二套表结构
- 不要复制另一份状态机
- 保持 stable ID、claim-before-work、workspace 约定
- 不要把抽象目标写得比当前仓库证据更宽

## Next Step

使用 `deriving-task-board-from-design`。
