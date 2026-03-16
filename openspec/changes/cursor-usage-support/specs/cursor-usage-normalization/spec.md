## ADDED Requirements

### Requirement: Cursor dashboard CSV format compatibility
系统 SHALL 支持解析 Cursor dashboard 当前导出格式，以及缺少 `Kind` / `Max Mode` 列的旧版导出格式，只要 `codetok` 统计所需的日期、模型和 token 列存在。

#### Scenario: Parse current Cursor dashboard export
- **WHEN** 一个 CSV 文件包含 `Date`、`Kind`、`Model`、`Input (w/ Cache Write)`、`Input (w/o Cache Write)`、`Cache Read` 和 `Output Tokens` 列
- **THEN** 解析器必须将该文件识别为有效的 Cursor usage export 并产出对应 usage 记录

#### Scenario: Parse legacy Cursor export without kind column
- **WHEN** 一个 CSV 文件包含 `Date`、`Model`、`Input (w/ Cache Write)`、`Input (w/o Cache Write)`、`Cache Read` 和 `Output Tokens` 列，但不包含 `Kind`
- **THEN** 解析器必须仍然能够按旧格式解析该文件并产出 usage 记录

### Requirement: Cursor token categories are preserved as exported
系统 SHALL 按照 Cursor dashboard 导出中的原始类别保留 token 列，不得通过相减或推断把一个 input 列改写成另一个 input 列。`Input (w/ Cache Write)` SHALL 映射到 `codetok` 的 cache-create 类别，`Input (w/o Cache Write)` SHALL 映射到普通 input 类别，`Cache Read` 与 `Output Tokens` SHALL 分别映射到 cache-read 与 output 类别。

#### Scenario: Preserve both input categories directly
- **WHEN** 一行 Cursor CSV 同时包含 `Input (w/ Cache Write)` 和 `Input (w/o Cache Write)` 的非零值
- **THEN** 解析器必须把这两个值作为两个独立输入类别保留下来
- **AND** 不得在解析阶段通过相减重新推导其中任意一个值

#### Scenario: Ignore non-token fields for token accounting
- **WHEN** 一行 Cursor CSV 还包含 `Total Tokens` 或 `Cost` 列
- **THEN** 这些列不得覆盖或替代 `codetok` 自身的 token 汇总字段
- **AND** token 统计结果必须只来自已映射的 input、cache-read 和 output 列

#### Scenario: Treat blank token cells as zero-valued categories
- **WHEN** 一行有效 Cursor CSV 中某些 token 列为空字符串，但其他必需字段可解析
- **THEN** 解析器必须把这些空 token 列按零值处理
- **AND** 不得因为空 token 单元格而丢弃整行有效 usage 数据

### Requirement: Cursor CSV rows become deterministic session-like records
每一个有效的 Cursor CSV 行 SHALL 生成一条确定性的 `SessionInfo` 记录，包含固定的 provider 名称、来源文件相关的稳定 session 标识、来自 `Date` 的时间戳，以及可用于 `session` / `daily` 统计的稳定排序结果。

#### Scenario: Generate one session-like record per valid row
- **WHEN** 一个 Cursor CSV 文件中存在两行有效 usage 数据
- **THEN** 解析器必须生成两条独立的 `SessionInfo` 记录
- **AND** 每条记录的时间必须来自对应行的 `Date`

#### Scenario: Preserve deterministic ordering across multiple files
- **WHEN** 默认目录或自定义目录下存在多个 Cursor CSV 文件
- **THEN** `CollectSessions` 返回结果必须采用确定性的排序规则
- **AND** 同一批输入重复解析时，记录顺序与 session 标识必须保持稳定

### Requirement: Malformed Cursor data is skipped conservatively
系统 SHALL 对坏行和坏文件采取保守跳过策略：无效行不得污染有效统计，无效文件不得导致整个 Cursor provider 收集流程中断，除非目录本身不可访问。

#### Scenario: Skip malformed rows inside an otherwise valid file
- **WHEN** 一个有效的 Cursor CSV 文件中混有时间格式错误或数字字段无法解析的行
- **THEN** 解析器必须跳过坏行
- **AND** 同文件中的有效行仍然必须被正常统计

#### Scenario: Skip invalid CSV file while continuing directory scan
- **WHEN** 默认 Cursor 目录下同时存在一个无效 CSV 文件和一个有效 CSV 文件
- **THEN** provider 收集过程不得因为无效文件而失败
- **AND** 有效 CSV 文件中的 usage 记录仍然必须出现在结果中
