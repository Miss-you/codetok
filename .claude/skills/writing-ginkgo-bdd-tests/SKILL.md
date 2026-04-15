---
name: writing-ginkgo-bdd-tests
description: Use when codetok needs behavior-focused Go tests, including BDD-style specs or Ginkgo if a suite already exists.
---

# Writing Behavior-Focused Go Tests

## Overview

把 task 和 spec 变成可执行的行为规格，而不是把现有 `testing` 用例机械翻译一遍。

这个 skill 保留 `writing-ginkgo-bdd-tests` 这个兼容入口名；在 codetok 中默认使用标准 Go `testing`、table-driven tests 和 e2e tests。只有仓库已经存在 Ginkgo suite，或用户明确要求继续使用 Ginkgo 时，才写 Ginkgo/Gomega specs。

**核心原则：** 行为测试必须追踪用户可观察行为或规格场景；测试名、subtest 名或 `Describe/When/It` 要能读成一句有意义的需求句子。

**覆盖原则：** 行为测试可以和 TDD 覆盖同一行为。长期希望沉淀可读规格时，应从 task/spec 重新表达核心行为，而不是只补现有测试没测到的边角。

**证据原则：** 如果没有看过测试失败，就不知道它是否真的能保护行为。

**REQUIRED SUB-SKILL:** 如果写行为测试时需要改生产代码，使用 `test-driven-development`，先让测试失败，再写最小实现。

## When to Use

使用在这些场景：

- 用户给出或指向 `docs/plans/*-design-task.md`、OpenSpec `spec.md`、`tasks.md`、proposal/design 等 artifact
- 已有 TDD/标准 Go 测试，但还需要一份更贴近 task/spec 的行为用例
- 需要把 task 验收条件、OpenSpec `Scenario`、CLI 契约或领域规则写成可读规格
- Go 包需要新增或扩展测试，尤其是 `cmd/`、`provider/`、`stats/` 或 `e2e/`
- 仓库里已经有 Ginkgo suite，或者团队明确要在现有 package 中继续使用它

不要使用在这些场景：

- 只是解释 Ginkgo/testify 区别，没有要求落测试
- 纯前端、非 Go 测试
- 缺少 task/spec 且无法从仓库推断验收行为；如果用户只给任务编号，先从 `docs/plans`、`openspec`、`workspace` 中定位 artifact
- 仓库没有 Ginkgo suite，却打算为了这个任务强行引入新依赖；默认先用标准 Go 测试、table-driven 测试或 e2e 测试

## Inputs and Outputs

输入应明确到文件或任务；如果不明确，先在仓库 artifact 中定位：

| 输入 | 用途 |
| --- | --- |
| task 列表 | 确定要覆盖的验收项和完成边界 |
| spec/proposal/design | 确定真实行为、错误分支、兼容约束 |
| 现有 TDD 测试 | 理解已有行为证据和测试 helpers；允许 BDD 重复覆盖核心行为 |
| 生产代码入口 | 确定应从包 API、CLI helper、fixture 还是 e2e 命令测 |

输出通常包括：

- 新增或更新的 `*_test.go` 文件
- 如果目标 package 已经在用 Ginkgo，才新增 `*_ginkgo_test.go`、`*_suite_test.go`
- 更新 task checklist 或验证记录，说明哪些 spec 场景已有 BDD 覆盖

## SOP

1. 读取 task 和 spec 文件；列出必须被 BDD 覆盖的行为句子。
2. 读取相关生产代码和现有 `*_test.go`；理解已有测试意图和 helper，但用 task/spec 重新选择 BDD 行为覆盖面。
3. 为每个行为选择测试层级：纯包 API、CLI command helper、fixture/golden、或端到端命令。
4. 先选择当前仓库最自然的测试形态：
   - 标准 Go：`TestXxx` + table-driven subtests，默认首选
   - e2e：命令输出、flags、fixture 或二进制行为需要端到端证明时使用
   - Ginkgo：仅当目标 package 已有 suite 或用户明确要求时使用
5. 先写一个最小可失败的行为测试，运行目标包测试看 RED。
6. 验证 RED 是有意义的：
   - 新行为缺失：失败应指向缺失行为。
   - 已实现行为补 BDD：对每个行为组做 negative control；高风险核心场景逐条验证；恢复后再继续。
   - suite/dependency failure 只能证明测试 harness，不能替代行为断言失败。
7. 跑到 GREEN；如果生产行为缺失，遵循 `test-driven-development` 修生产代码。
8. GREEN 后才重构测试结构、抽 helper、整理 fixtures；重构期间保持测试全绿。
9. 继续补齐 task/spec 场景；优先覆盖正常路径、错误路径、边界、兼容契约。
10. 跑 focused package test 和仓库 gate：
   - `go test ./path/to/pkg -run TestName -count=1`
   - 如果仓库已用 Ginkgo，再跑对应 verbose 命令
   - `make test`

## BDD Red-Green-Refactor

### RED - Write a Failing Spec

每次只写一个最小行为测试。测试名要清楚，断言真实代码，mock 只在不可避免时使用。

```go
func TestResolveDailyDateRangeRejectsInvalidDays(t *testing.T) {
	req := DailyRequest{Days: -1}

	result, err := ResolveDailyDateRange(req)

	if err == nil || !strings.Contains(err.Error(), "days must be positive") {
		t.Fatalf("expected positive-days error, got %v", err)
	}
	if result != (DateRange{}) {
		t.Fatalf("expected zero range, got %#v", result)
	}
}
```

运行目标包测试，确认失败原因正确。测试直接通过时，不要继续堆场景；先证明断言能抓住错误。

### GREEN - Make the Spec Pass

如果生产行为已经存在，恢复 negative control，让测试回到真实期望并跑绿。
如果行为缺失，只写让当前测试通过的最小生产代码，不顺手实现额外 task。

### REFACTOR - Clean Up While Green

只在 GREEN 后清理：

- 抽真正共享的 setup helper
- 抽 fixture helper
- 合并同一行为的多个输入为 table-driven cases
- 改善 test/subtest 名称；如果使用 Ginkgo，再改善 `Describe/When/It` 命名

不要在 refactor 阶段新增行为。

## Scenario Shape

每个测试主体应能看出 Given / When / Then：

```go
func TestResolveDailyDateRangeRejectsInvalidDays(t *testing.T) {
	req := DailyRequest{Days: -1} // Given

	result, err := ResolveDailyDateRange(req) // When

	if err == nil || !strings.Contains(err.Error(), "days must be positive") { // Then
		t.Fatalf("expected positive-days error, got %v", err)
	}
	if result != (DateRange{}) {
		t.Fatalf("expected zero range, got %#v", result)
	}
}
```

不要求固定写注释；要求 setup、动作、断言三段在阅读上清楚。

## Scenario Priority

优先选择这些 BDD 场景：

1. 用户最关心的主流程
2. spec 明确写出的 `Scenario`
3. task 验收条件和兼容契约
4. 错误路径、边界值、历史 bug
5. 低价值实现细节不写 BDD

## BDD Mapping

| Artifact 内容 | 标准 Go 表达 | Ginkgo 表达，仅限已有 suite |
| --- | --- | --- |
| capability / command / package behavior | `TestDailyCommand...` | `Describe("Daily command", ...)` |
| OpenSpec `Scenario` 或 task 验收条件 | `t.Run("invalid window", ...)` | `When("the requested window is invalid", ...)` |
| 可观察结果 | assertion on stdout/stderr/error/result | `It("prints validation errors without stdout", ...)` |
| 同一行为的多个输入例子 | table-driven cases | `DescribeTable` + `Entry` |
| 共享 fixture/setup | helper function or `t.TempDir()` | `BeforeEach`，保持短小 |
| 清理临时文件、mock controller | `t.Cleanup` | `DeferCleanup` |

好的测试标题描述行为，不描述实现：

```go
func TestResolveDailyDateRangeRejectsInvalidDays(t *testing.T) {
	result, err := ResolveDailyDateRange(DailyRequest{Days: -1})

	if err == nil || !strings.Contains(err.Error(), "days must be positive") {
		t.Fatalf("expected positive-days error, got %v", err)
	}
	if result != (DateRange{}) {
		t.Fatalf("expected zero range, got %#v", result)
	}
}
```

## Ginkgo Suite Pattern

如果目标 package 已经在用 Ginkgo，且还没有 suite 文件，再新增一个：

```go
package daily_test

import (
    "testing"

    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
)

func TestDaily(t *testing.T) {
    RegisterFailHandler(Fail)
    RunSpecs(t, "Daily Suite")
}
```

命名规则：

- suite 文件：`<package>_suite_test.go` 或已有项目约定名称
- specs 文件：`<feature>_ginkgo_test.go`
- suite 名称：`Daily Suite`、`Provider Parser Suite`，能定位 package 或 capability

## Test Design Rules

- 不要逐行翻译旧测试；从 task/spec 重建行为场景。
- 不要为了写 BDD 先引入新依赖；codetok 默认优先标准 Go 测试、table-driven 测试和 e2e 测试。
- 可以重复覆盖 TDD 已测行为；重复的理由应是“这是 task/spec 的核心行为”，不是“把旧测试换个语法”。
- 每个 test/subtest 只验证一个用户可观察结果或一个紧密结果组。
- 标题里出现多个无关的 “and” 时，优先拆成多个 tests 或 subtests。
- 行为测试通过得太快时，对行为组做 negative control，证明它不是空保护。
- 嵌套过深时，优先拆测试函数、拆文件、或把条件下沉到 helper/fixture 名称。
- 复用现有 fixtures/golden，但不要为了方便修改 canonical fixture。
- mock 不是 Ginkgo 内置能力；优先手写 fake，只有需要验证调用次数、参数或顺序时才用 gomock/testify mock。
- mock 不能成为被测主体；断言用户可观察行为，少量断言调用参数只用于证明协作边界。
- 不要为写 BDD 测试改生产 API；如果现有代码不可测，先说明耦合点，再用 TDD 做最小重构。
- 如果 BDD 暴露 spec 和实现不一致，停止扩大测试面，先报告不一致并修正 artifact 或实现。

## Quick Checklist

- [ ] task/spec 中每个关键 `Scenario` 或验收项都有对应行为测试
- [ ] 测试名或 subtest 名能读成业务句子；如果使用 Ginkgo，`Describe/When/It` 也要可读
- [ ] 已避免机械复制现有 TDD 测试，即使覆盖行为有意重复
- [ ] 新增行为 spec 看过 meaningful RED；已实现行为组看过 negative control failure
- [ ] RED 失败原因符合预期，不是 typo、import、fixture 路径错误
- [ ] 如果改了生产代码，已遵循 `test-driven-development`
- [ ] package 测试通过
- [ ] `make test` 通过
- [ ] 没有为了这个任务额外引入 Ginkgo 依赖，除非仓库已有 suite
- [ ] task checklist 或验证记录说明 BDD 覆盖范围

## Rationalizations to Reject

| Excuse | Reality |
| --- | --- |
| “现有 TDD 已经过了，行为测试直接补上就行” | 直接通过的补测不证明能抓回归；至少按行为组做 negative control。 |
| “TDD 已经覆盖，所以不用写行为测试” | 如果这是 task/spec 核心行为，可以重复覆盖，作为长期规格入口。 |
| “只是换成 Ginkgo 写法” | BDD 不是换皮；必须从 task/spec 写行为句子。 |
| “先把所有 tests/specs 写完再跑” | 一次只写一个行为，先看失败，再继续。 |
| “suite/dependency 失败已经算 RED” | 那只证明 harness；行为 spec 仍要证明断言有效。 |
| “mock 调用都对，所以行为对” | mock 行为不是产品行为；优先断言真实输出、错误、状态或文件内容。 |
| “为了测试方便改个 API” | 测试困难说明设计耦合；先说明问题，用 TDD 做最小重构。 |

## Red Flags

- 先改生产代码，再补行为测试
- `TestWorks`、`It("works")`、`It("returns error")`
- 大量复制现有 case，但没有 task/spec 映射
- 因为 TDD 已覆盖就跳过 task/spec 核心行为场景
- 新测试第一次运行就全绿，且没有 negative control
- 不能解释某条测试对应哪个 task/spec 场景
- mock 断言多于行为断言
- 仓库没有 Ginkgo，却为单个任务引入 Ginkgo/Gomega

遇到这些信号，停止扩大测试面，回到 task/spec 重新写一个最小可失败的行为规格。

## Common Mistakes

| Mistake | Fix |
| --- | --- |
| 把 Ginkgo 当普通测试换皮 | 从 task/spec 写行为句子，再落代码 |
| `TestWorks`、`It("works")`、`It("returns error")` | 写清条件下的业务结果 |
| 一个 helper 做完所有世界构造 | 只放共享、稳定、必要 setup |
| 所有 case 都用复杂 table | 只有同一行为多组数据才用 table |
| 为 mock 而 mock | 能用真实代码或手写 fake 就不用 mock |
| 只跑 `go test ./...` | 优先跑 focused package test，再跑 `make test` |

## Acceptance Criteria

- 关键 task/spec 场景已被行为测试覆盖，且能对应到 codetok 的真实命令、解析或聚合行为
- 默认仍兼容标准 Go 测试栈，没有强行把 Ginkgo 变成新依赖
- 如果已有 Ginkgo suite，则新增的 spec 能正常运行并保持可读
- 相关 package 测试和 `make test` 已通过
