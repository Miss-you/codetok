---
name: deriving-task-board-from-design
description: Use when an approved codetok design needs a claimable task board for parallel work.
---

# 从设计推导任务板

## 概述

把已批准设计整理成可执行任务板，而不是再写一份第二设计文档。

**核心原则：** 任务 ID、依赖边、状态历史和 `Done When` 是执行期真相源，不能依赖聊天记忆。

这个 skill 负责创建或刷新 `docs/plans/*-design-task.md`，不负责真正实现某个任务。

## 何时使用

在这些场景使用：

- `docs/plans/*-design.md` 已批准，准备进入实现
- 对应的 `*-design-task.md` 不存在
- task 文档已存在，但太粗、依赖缺失，或无法安全并行认领
- 准备让多个 agent 基于一个统一板子持续开发 `cmd/`、`provider/`、`stats/`、`cursor/`、`e2e/` 或文档
- 需要把 `make fmt`、`make vet`、`make test`、`make build`、可选 `make lint` 和 CLI smoke 落到任务级别

这些场景不要使用：

- 设计本身还在漂移，此时先用 `compatibility-first-planning`
- 你已经认领了某个任务，准备交付，此时改用 `delivering-go-task-end-to-end`

## SOP

每次创建或刷新任务板，都按同一循环执行：

1. 先定义当前任务板必须澄清什么
2. 基于设计和仓库证据更新表格
3. 审核 ID、依赖、状态和 `Done When`
4. 修复漂移、缺失证据或不可认领状态
5. 重新阅读，直到“下一个安全可做任务”是明确的

不要把任务板当成一次性文书工作。

## 必做检查

在落笔前先做这些检查：

1. 确认源设计文件真实存在
2. 用相同 stem 推导任务文档路径：`*-design.md -> *-design-task.md`
3. 检查任务文档是否已存在
4. 如果已存在，保留稳定 ID、已完成状态、已认领历史，除非仓库证据明确反驳它
5. 确认任务拆分能覆盖 codetok 的真实边界，而不是把 `cmd/`、`provider/`、`stats/`、`cursor/`、`e2e/` 混成一个大任务

不要凭记忆重排任务 ID，也不要先开工再补任务板。

## 输出形状

任务文档固定包含这些 section：

1. `Source Design`
2. `Status Legend`
3. `Dependency Rules`
4. `Task Table`
5. `Claiming Rules`
6. `Change Log`

`Task Table` 保持执行导向：

| ID | Title | Goal | Depends On | Parallel | Status | Owner | Claimed At | Workspace | Change | Done When | Notes |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |

`Notes` 里默认写入第一个预期验证命令或 gate。

## 流程

### 1. 读取源设计

抽出这些内容：

- goal 与 non-goals
- 阶段闭环
- 粗粒度任务和稳定 ID
- `cmd/`、`provider/`、`stats/`、`cursor/`、`e2e/` 的边界约束
- 任何必须保留的 compatibility contract

如果设计里已有稳定任务 ID，优先复用。

### 2. 初始化或刷新任务文档

如果任务文档不存在：

- 创建它
- 总任务数尽量控制在 `8-15`
- 按可交付行为闭环拆，而不是按目录名机械拆
- 典型任务应能独立覆盖解析、聚合、命令、Cursor 本地/同步边界、e2e、文档或验证中的一块

如果任务文档已存在：

- 刷新，不整份重写
- 保留已有完成/认领历史
- 任务合并或拆分必须写进 `Change Log`

### 3. 定义依赖与并行组

每个任务都要有：

- 明确硬依赖
- 可选并行标记
- 客观 `Done When`
- 在 `Notes` 中写出首个验证命令或 gate
- 如果任务改动行为，优先把 `make test`、`make build`、`make fmt`、`make vet` 或 CLI smoke 之一写进 `Notes`

`ready` 必须是推导结果，不要作为独立状态存储。

### 4. 使用显式状态流转

状态集固定为：

- `todo`
- `claimed`
- `research`
- `spec`
- `implementing`
- `verifying`
- `review`
- `blocked`
- `done`

`blocked` 可以中断任何活动状态。解除阻塞后，要在 `Notes` 里写出 `resume_to=<state>`。

### 5. 定义认领规则

认领必须遵守：

- 只能认领 `todo` 且所有硬依赖已 `done` 的任务
- 先更新任务板
- 再记录 `Owner` 与 `Claimed At`
- 然后创建 `workspace/<task-id>/`
- 把 workspace 路径写回任务表
- 在 `Change Log` 中追加认领记录
- 认领前要看清任务是否会碰到 `cmd/`、`provider/`、`stats/`、`cursor/`、`e2e/` 或 Makefile gate

如果要选“下一个任务”，优先选能解锁更多后续工作的高杠杆任务。

### 6. 用真实产物回收状态

刷新已有任务板时，以这些证据为准：

1. 源设计和稳定任务 ID
2. 活跃或已归档的 OpenSpec change
3. `workspace/<task-id>/` 产物
4. 已记录的验证输出
5. 当前任务文档文本

如果一个任务被标为 `done`，但证据不支持，就回退到仍有证据支持的最高状态。

## 快速参考

| 场景 | 动作 |
| --- | --- |
| task 文档不存在 | 从已批准设计创建 `*-design-task.md` |
| task 文档过时 | 刷新它，但保留稳定 ID 与历史 |
| 某任务依赖都完成且状态为 `todo` | 它是可认领的 |
| 某任务已认领 | `workspace/<task-id>/` 应真实存在 |
| 设计已经给出粗粒度任务 | 优先复用，不重新发明 ID |

## 常见错误

- 把任务板写成第二份设计说明
- 没写 `Done When`，导致完成条件主观化
- 还没更新任务板就先建 workspace 或先写代码
- 让并行关系只存在于聊天里，不落到表格中
- 把 `cmd/`、`provider/`、`stats/`、`cursor/`、`e2e/`、tooling 任务混成一个不可并行的大块

## 交接

当任务板已经初始化或刷新完成后，使用 `delivering-go-task-end-to-end` 去认领并交付其中一个 ready task。

## Acceptance Criteria

- 任务板和已批准设计一一对应，没有漂移
- 每个任务都有稳定 ID、清晰目标、依赖和可检查的 `Done When`
- 任务拆分能支持并行工作，且能直接映射到 `codetok` 的真实目录和验证 gate
- 任务板里没有遗留的源仓库术语
- 任何被标为已完成的任务，都能从设计、workspace 产物和验证输出里找到证据
