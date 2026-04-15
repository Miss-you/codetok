---
name: delivering-go-task-end-to-end
description: Use when a codetok task is ready to claim and needs end-to-end delivery through implementation, validation, and review.
---

# 端到端交付一个 Go 任务

## 概述

把一个已认领的 `codetok` 任务从调研推进到关闭，同时避免 task 状态、spec 状态、代码状态和验证状态彼此漂移。

**核心原则：** 只有当任务板、workspace 产物、OpenSpec 状态、代码变更和验证证据全部一致时，这个任务才算真正完成。

**隔离规则：** 非平凡任务默认在隔离 git worktree 中执行，避免与其他并行任务互相污染。

**DEFAULT SUB-SKILLS:** `using-git-worktrees`, `deriving-task-board-from-design`, `openspec-explore`, `openspec-ff-change`, `openspec-apply-change`, `openspec-sync-specs`, `openspec-archive-change`, `test-driven-development`, `requesting-code-review`, `receiving-code-review`, `verification-before-completion`

OpenSpec 相关 sub-skill 在任务改变产品行为或规格时启用；纯文档或纯工具任务必须在任务板中写清为何不需要它们。

每次阶段切换前，都先更新任务板，再进入下一阶段。

## 何时使用

在这些场景使用：

- 已存在批准通过的 `docs/plans/*-design.md`
- 已存在对应的 `*-design-task.md`
- 需要认领并交付一个具体任务
- 该任务需要经历 research/spec/implementation/verification/review/close
- 任务影响 `cmd/`、`provider/`、`stats/`、`cursor/` 或 `e2e/` 中的 Go 代码、CLI 行为或测试契约

这些场景不要使用：

- task board 不存在或过时，此时先用 `deriving-task-board-from-design`
- 工作还在宽泛设计阶段，此时先用 `compatibility-first-planning`

## SOP

1. 读取批准通过的设计文档和对应 task board，确认本次只做一个 task。
2. 用隔离 git worktree 开始工作，避免污染主工作区或并行任务。
3. 在 worktree 中把 task 标为 `claimed`，并创建 `workspace/<task-id>/`。
4. 先理解现有实现和测试，再决定是否需要 OpenSpec change、测试策略和代码改动。
5. 如果任务改变行为或契约，先产出 `original_impl.md`、`new_impl.md`、`final_impl_v1.md` 和 `test_strategy.md`，再进入实现。
6. 用 TDD 或最小可验证增量修改 `cmd/`、`provider/`、`stats/`、`cursor/`、`e2e/` 下的代码。
7. 依次跑 `make fmt`、`make test`、`make vet`、`make build`，`make lint` 在可用时补跑。
8. 处理 review 后再把 task 切到 `done`，并收口 workspace 产物和剩余风险。

## 默认 workspace 产物

对于中大型、边界不清晰或需要多阶段验收的任务，`workspace/<task-id>/` 默认至少包含：

- `original_impl.md`
- `new_impl.md`
- `final_impl_v1.md`
- `final_impl.md`
- `test_strategy.md`
- `todo.md`，仅当仍有遗留问题或延后项时需要

这些文件可以很短，但不能在本应存在时完全缺失后仍宣称阶段完成。

## Stage Loop

每个大阶段都必须按这个顺序执行：

1. 明确本阶段目标
2. 执行本阶段任务
3. 审核或校验本阶段结果
4. 修复发现的问题
5. 重新审核或复验，直到达到阶段验收阈值

不要只执行，不审核；也不要为了“省时间”跳过复验。

### 1. 认领任务并建立隔离上下文

目标：

- 选出一个真正可做的任务
- 在 worktree、任务板、workspace 三者之间建立稳定关联

执行：

1. 读取批准通过的设计文档
2. 读取对应的 `*-design-task.md`
3. 如果任务板不存在，停止并改用 `deriving-task-board-from-design`
4. 选择一个 `status=todo` 且所有硬依赖都已 `done` 的任务
5. 在开始 research/spec/code 之前，先用 `using-git-worktrees` 建立隔离 worktree
6. 在该 worktree 中把任务更新为 `claimed`
7. 在该 worktree 中创建 `workspace/<task-id>/`
8. 记录 `Owner`、`Claimed At`、`Workspace`
9. 正式开始调研时，把任务切到 `research`

验收阈值：

- worktree 已就绪
- 任务板已更新
- `workspace/<task-id>/` 真实存在
- 认领信息已落盘

### 2. 澄清实现方案并定稿

目标：

- 明确当前行为
- 明确更干净的 codetok 实现方式
- 在不过度设计的前提下定出最终方案

执行：

- 如果任务跨越 `cmd/`、`provider/`、`stats/`、`cursor/`、`e2e/` 多个面，或方案仍不清晰，默认采用 3-agent 模式：
  - `subagent 1`：基于当前仓库，输出 `workspace/<task-id>/original_impl.md`
  - `subagent 2`：提出更简洁的实现方案，输出 `workspace/<task-id>/new_impl.md`
  - `subagent 3`：综合前两者，产出 `workspace/<task-id>/final_impl_v1.md`
- 如果任务很小且边界清楚，可以由 owner 直接写出上述文档，但仍需要至少 `1` 个独立 reviewer 做复核

中大型任务默认再启动 `2+` 个 review subagent 评审 `final_impl_v1.md`；小任务至少保留 `1` 个独立 review pass。

建议评分维度：

| 维度 | 分值 |
| --- | --- |
| codetok CLI 契约忠实度 | 25 |
| provider/stats 语义忠实度 | 25 |
| Go-native 简洁性与可维护性 | 20 |
| 不过度设计 / 边界干净 | 15 |
| 验证清晰度与可测试性 | 15 |

验收阈值：

- 平均分 `>= 80`
- 没有高严重度问题

修复与复验：

- 未达标就修改 `final_impl_v1.md` 并重新评审
- 达标后产出 `workspace/<task-id>/final_impl.md`
- 然后把任务切到 `spec`

注意：

- 如果任务明确要求对齐某个外部参考实现，再引入该参考
- 否则当前仓库和已批准设计就是唯一真相，不要强行做外部 parity 调研

### 3. Spec 与测试策略

目标：

- 把最终方案变成可执行 change
- 明确“要证明什么”，而不是机械堆测试

执行：

1. 如果任务改变产品行为或规格，使用 `openspec-ff-change` 基于 `final_impl.md` 创建 change
2. 把 change 名写回任务板
3. 单独产出 `workspace/<task-id>/test_strategy.md`
4. 用 review subagent 评审 `final_impl.md`、spec 产物和 `test_strategy.md`

高严重度问题包括：

- 行为与任务目标不一致
- 破坏 `cmd/`、`provider/`、`stats/`、`cursor/` 或 `e2e/` 相关契约
- scope 超出已认领任务
- `test_strategy.md` 不能证明关键行为

验收阈值：

- 没有高严重度 spec 问题
- 任务板、spec、`final_impl.md` 和 `test_strategy.md` 一致

说明：

- 对纯文档或纯工具任务，可在任务板中明确 `Change=-`，但必须写清为何不需要 OpenSpec

当实现真正开始时，把任务切到 `implementing`。

### 4. 按 TDD 实现代码

目标：

- 以最小、可验证增量实现已批准行为

执行规则：

- 使用 `openspec-apply-change`
- 遵循 `test-driven-development`
- 只有写集不重叠时才并行
- 始终保留一个 owner 负责集成

不要把多个并行 agent 扔到同一批文件上。

### 5. 代码校验与验证

目标：

- 证明实现真的能跑、能编、能过关键验证

执行：

- 在进入验证前，把任务切到 `verifying`
- 至少运行与任务相关的 gate：
  - `make fmt`
  - `make vet`
  - `make test`
  - `make build`
  - `make lint`，仅当本机安装了 `golangci-lint`
- 如果任务影响 CLI 行为，补一个 focused CLI smoke test
- 如果任务影响 provider 解析、stats 聚合或 Cursor 本地/同步边界，补对应的 focused package test
- 如果任务影响 e2e 行为，补一个 focused e2e test

审核：

- 使用 `verification-before-completion`
- 对照 `test_strategy.md` 检查当前验证是否真的覆盖目标行为

验收阈值：

- 规定的验证项全部 fresh 通过
- 或显式跳过项已在任务板与 `todo.md` 说明理由

### 6. Code Review 与问题分流

目标：

- 在关闭前发现明显 bug、回归和低级正确性问题

执行：

- 进入 review 前，把任务切到 `review`
- 使用 `requesting-code-review`
- 再用 `receiving-code-review` 评估建议

问题分流：

- 明确 bug / regression / correctness 问题：现在修
- 更大范围的架构讨论或可接受延后项：记入 `todo.md`

验收阈值：

- 没有未处理的 must-fix 问题
- 修复后已重跑受影响验证

### 7. 关闭任务

目标：

- 把状态、证据和遗留风险全部收口

执行：

1. 如有 OpenSpec change，按实际需要做 `openspec-sync-specs` / `openspec-archive-change`
2. 更新任务板，把任务标成 `done`
3. 确认 workspace 产物完整
4. 把剩余风险、延后项写入 `todo.md` 或任务板 `Notes`

## Acceptance Criteria

- 任务板、workspace、OpenSpec、代码和验证证据彼此一致
- 相关 `make fmt`、`make test`、`make vet`、`make build` 已重新跑过，`make lint` 在可用时也已跑过
- 影响到的 `cmd/`、`provider/`、`stats/`、`cursor/`、`e2e/` 改动都有对应的 focused test 或 smoke test
- 任务状态已从 `claimed` 推进到 `done`

关闭阈值：

- 任务板状态为 `done`
- workspace、spec、代码和验证证据一致
- 遗留项已落盘，而不是只存在聊天里

## 常见错误

- 没有先认领任务就直接开工
- 任务板、workspace、OpenSpec、代码状态彼此不一致
- 所有任务都强制做重文档，导致过度流程化
- 没有 fresh verification 就声称任务完成
- 文件写集重叠时还强行多 agent 并行

## 交接

任务关闭后，如果已经有 PR 或准备开 PR，下一步使用 `monitoring-pr-ai-reviews`。
