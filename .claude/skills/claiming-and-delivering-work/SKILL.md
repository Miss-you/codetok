---
name: claiming-and-delivering-work
description: Use when a codetok task board already exists and a single ready task needs to be claimed for delivery.
---

# Claiming And Delivering Work

## Overview

这是一个兼容命名 alias，用来承接“认领并交付任务”这一类表述。

它的 canonical workflow 是 `delivering-go-task-end-to-end`。不要在这里再维护第二套端到端交付流程。

如果任务板还不存在，先走 `deriving-task-board-from-design`，不要在 alias 里补造板子。

## When to Use

在这些场景使用：

- 用户明确说的是 “认领任务并交付 / claim and deliver”
- 已有任务板，且你准备推进一个具体 ready task
- 你想沿用口语化入口词，而不是直接点 canonical skill

这些场景不要使用：

- 设计或任务板仍未就绪
- 当前工作还停留在宽泛规划阶段
- 你已经确定应该直接使用 `delivering-go-task-end-to-end`

## What To Do

1. 确认已存在 `docs/plans/*-design-task.md`
2. 选择一个所有硬依赖都已满足的 ready task
3. 直接转到 `delivering-go-task-end-to-end`
4. 按该 skill 完成认领、spec、实现、验证、review 和关闭

## SOP

1. 确认当前工作确实是单个 ready task，而不是重新规划。
2. 读取 `docs/plans/*-design-task.md`，找到依赖已满足的 `status=todo` 任务。
3. 记录任务 ID、owner、claim 时间和 workspace 位置。
4. 把任务交给 `delivering-go-task-end-to-end`，继续推进实现与验证。
5. 如果任务涉及 `cmd/`、`provider/`、`stats/`、`cursor/` 或 `e2e/`，保持任务板、代码和测试记录同步。

## Guardrails

- 不要跳过任务板更新直接开工
- 不要把 alias 扩展成独立 workflow
- 保持 workspace、OpenSpec、代码和验证证据一致
- 任务不就绪时先返回上游设计或 task-board 工作流

## Acceptance Criteria

- 任务板中已明确本次认领的单个 task
- 该 task 的依赖已满足，且范围落在 codetok 的现有模块边界内
- 已把工作交给 canonical workflow 继续执行
- 不需要在这个 alias 中再定义新的交付路径
