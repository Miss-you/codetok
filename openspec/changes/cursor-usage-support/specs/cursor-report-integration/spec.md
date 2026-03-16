## ADDED Requirements

### Requirement: Cursor reports merge imported and synced CSV sources by default
在未显式指定 `--cursor-dir` 时，系统 SHALL 同时扫描 `codetok` 默认 Cursor 根目录下的手工导入 CSV 与同步缓存 CSV，并把两类来源统一纳入 Cursor provider 的本地统计视图。

#### Scenario: Combine imported and synced Cursor CSV files
- **WHEN** 默认 Cursor 根目录下同时存在手工导入的 CSV 文件和 sync 生成的 CSV 文件
- **THEN** `daily` 和 `session` 必须都能消费这两类文件
- **AND** Cursor 统计结果必须包含两类来源中的全部有效 usage 记录

#### Scenario: Missing one source type does not fail reporting
- **WHEN** 默认 Cursor 根目录下只存在导入数据或只存在同步缓存中的一种
- **THEN** `daily` 和 `session` 仍然必须正常返回 Cursor 统计结果
- **AND** 不得因为另一类来源缺失而报错

#### Scenario: Preserve compatibility with legacy flat Cursor CSV layout
- **WHEN** 默认 Cursor 根目录下仍然存在历史遗留的平铺 CSV 文件，而不是新的分层子目录
- **THEN** `daily` 和 `session` 仍然必须发现并消费这些历史 CSV 文件
- **AND** 不得要求用户先迁移旧目录布局才能继续查看 Cursor 统计

### Requirement: Explicit `--cursor-dir` override is authoritative
当用户显式提供 `--cursor-dir` 时，系统 SHALL 只扫描该目录，不得再合并默认 Cursor 根目录中的其他导入路径、同步缓存路径或本地凭据状态。

#### Scenario: Use only the explicitly provided Cursor directory
- **WHEN** 用户执行 `daily --cursor-dir <custom-dir>` 或 `session --cursor-dir <custom-dir>`
- **THEN** 系统必须只从 `<custom-dir>` 读取 Cursor CSV 数据
- **AND** 不得自动读取默认 Cursor 根目录下的同步缓存

#### Scenario: Explicit directory override remains local-only
- **WHEN** 用户在传入 `--cursor-dir` 的同时本地还保存了 Cursor 凭据
- **THEN** 报表命令仍然不得访问远程 Cursor API
- **AND** 结果必须只由 `<custom-dir>` 中的本地文件决定

### Requirement: Cursor collection behavior is shared across reporting commands
`daily` 和 `session` SHALL 通过同一条共享收集路径解析 Cursor 输入，以保证 provider 过滤、目录覆盖、缺失目录处理和坏文件跳过行为保持一致。

#### Scenario: Daily and session use the same Cursor file-resolution rules
- **WHEN** 某个 Cursor CSV 集合在 `session` 视图中可见
- **THEN** 同一集合在相同 provider 过滤和目录覆盖条件下也必须被 `daily` 视图消费
- **AND** 两个命令不得对同一批 Cursor 文件采用不同的收集范围

#### Scenario: Missing default Cursor data does not break non-Cursor reporting
- **WHEN** 用户执行没有 `--provider cursor` 过滤的普通报表命令，而默认 Cursor 目录不存在
- **THEN** 命令必须继续正常处理其他 provider
- **AND** 不得因为 Cursor 默认目录缺失而整体失败

#### Scenario: Reporting continues to use existing local Cursor cache after sync failure
- **WHEN** 最近一次 Cursor sync 失败，但默认 Cursor 根目录中仍然保留旧的有效同步缓存或导入 CSV
- **THEN** `daily` 和 `session` 仍然必须使用这些已有本地数据生成 Cursor 报表
- **AND** 不得因为最近一次 sync 失败而拒绝返回已有的本地统计结果

### Requirement: Cursor reporting remains token-focused
现有 `daily` 与 `session` 输出 SHALL 继续只报告 token usage 结果，不得把 Cursor 归因活动指标混入这些命令的 token 统计字段。

#### Scenario: Cursor activity metrics are excluded from token reports
- **WHEN** 本地同时存在 Cursor token CSV 与活动归因数据库
- **THEN** `daily` 与 `session` 的 Cursor token 结果必须只基于 CSV usage 数据
- **AND** 不得把 `tab` / `composer` 活动指标写入 token usage 字段
