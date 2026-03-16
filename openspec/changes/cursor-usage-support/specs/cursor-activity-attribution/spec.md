## ADDED Requirements

### Requirement: Cursor activity attribution uses a dedicated local data source
系统 SHALL 通过专门的 Cursor 活动归因能力读取本地 `~/.cursor/ai-tracking/ai-code-tracking.db`，并把其中与 `tab` / `composer` 相关的 accepted-line 指标映射为独立的活动数据模型。

#### Scenario: Read activity attribution from local Cursor tracking database
- **WHEN** 本地存在可访问的 `ai-code-tracking.db`
- **THEN** 系统必须能够从中提取与 Cursor `tab` / `composer` 相关的活动指标
- **AND** 这些指标必须被建模为独立于 token usage 的活动数据

#### Scenario: Missing tracking database does not cause failure
- **WHEN** 本地不存在 `ai-code-tracking.db` 或该库当前不可读
- **THEN** 系统不得因此破坏其他 Cursor token 功能
- **AND** 活动归因能力必须返回空结果或明确的“无数据”状态

### Requirement: Activity attribution is never presented as token accounting
系统 MUST 把 Cursor 活动归因与 Cursor token usage 明确分离；`tab` / `composer` 活动指标不得被合并进 `provider.TokenUsage`、`daily` 总量或 `session` 总量中。

#### Scenario: Activity data does not alter token totals
- **WHEN** 同一时间范围内既存在 Cursor dashboard token 数据，也存在本地活动归因数据
- **THEN** token 报表中的 input、output、cache-read、cache-create 和 total 数值不得因为活动归因数据而变化

#### Scenario: Activity data is labeled as attribution rather than tokens
- **WHEN** 系统输出 Cursor 活动归因结果
- **THEN** 输出字段或展示文案必须明确表明该结果是 activity / attribution
- **AND** 不得复用 token 命名来表达这些指标

### Requirement: Cursor activity attribution separates composer and tab metrics
系统 SHALL 在活动归因输出中把 `composer` 与 `tab` 指标分开暴露，以便用户区分两类 Cursor 行为来源。

#### Scenario: Output both composer and tab activity categories
- **WHEN** 本地活动归因数据同时包含 `composer` 和 `tab` 相关指标
- **THEN** 输出结果必须分别提供这两类指标
- **AND** 不得把两者在没有标签的情况下混合成单一总数

#### Scenario: Output remains valid when only one category exists
- **WHEN** 本地活动归因数据只包含 `composer` 或只包含 `tab` 中的一类
- **THEN** 系统仍然必须返回有效结果
- **AND** 缺失的另一类必须以空值、零值或明确缺失状态表示
