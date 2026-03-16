## ADDED Requirements

### Requirement: Explicit Cursor authentication lifecycle
系统 SHALL 提供显式的 Cursor 认证命令，用于保存、检查和移除活动 Cursor 会话凭据。登录流程必须先验证用户提供的 `WorkosCursorSessionToken`，只有验证成功后才允许写入本地凭据存储。

#### Scenario: Save active Cursor credentials after successful validation
- **WHEN** 用户执行 Cursor 登录命令并提供一个有效的 `WorkosCursorSessionToken`
- **THEN** 系统必须先向 Cursor 校验该 token
- **AND** 只有校验成功后才允许把该 token 保存为当前活动凭据

#### Scenario: Reject invalid Cursor credentials
- **WHEN** 用户执行 Cursor 登录命令并提供一个无效或过期的 session token
- **THEN** 系统必须拒绝保存该 token
- **AND** 必须向用户返回可操作的失败原因

#### Scenario: Remove active credentials on logout
- **WHEN** 用户执行 Cursor 登出命令
- **THEN** 系统必须移除当前活动 Cursor 凭据
- **AND** 后续显式 Cursor 状态检查必须表现为“未登录”或等价状态

### Requirement: Cursor dashboard history sync is explicit and local-cache based
系统 SHALL 通过显式的 Cursor sync 命令从 Cursor dashboard 拉取 usage CSV，并把结果原子写入 `codetok` 自己管理的本地缓存目录。同步结果必须仍然是可被本地 parser 重复消费的 CSV 文件，而不是只存在内存中的临时结果。

#### Scenario: Sync Cursor usage CSV into local cache
- **WHEN** 用户已经保存有效的活动 Cursor 凭据并执行 sync 命令
- **THEN** 系统必须从 Cursor dashboard usage API 获取 CSV 数据
- **AND** 必须把返回内容原子写入本地 Cursor 缓存目录

#### Scenario: Preserve existing cache on sync failure
- **WHEN** 用户执行 sync 时遇到网络错误、鉴权失败或 API 返回非法响应
- **THEN** 系统不得破坏已有的本地同步缓存
- **AND** 必须把失败结果以明确的错误信息返回给用户

### Requirement: Remote Cursor API access is limited to explicit Cursor subcommands
系统 MUST 只允许在显式的 Cursor 子命令流程中访问远程 Cursor API，普通报表命令不得隐式触发登录校验或 CSV 同步。

#### Scenario: Reporting commands do not trigger implicit sync
- **WHEN** 用户执行 `daily` 或 `session`
- **THEN** 系统不得自动访问 Cursor dashboard API
- **AND** 报表结果必须完全基于本地已有的 Cursor 数据源

#### Scenario: Cursor status validates saved credentials explicitly
- **WHEN** 用户执行 Cursor 状态检查命令
- **THEN** 系统可以使用已保存凭据访问 Cursor 状态接口
- **AND** 返回结果必须明确区分“本地存在凭据”和“远程校验通过”这两个状态

### Requirement: Cursor credential and cache storage is tool-owned
系统 SHALL 将 Cursor 凭据和同步缓存保存到 `codetok` 自己管理的路径下，并采用适合本地 CLI 的文件权限与原子写入策略，避免把凭据或缓存散落到普通报表输入目录之外的未知位置。

#### Scenario: Persist credentials with restricted local access
- **WHEN** 系统保存 Cursor 活动凭据
- **THEN** 凭据文件必须写入 `codetok` 管理的本地配置路径
- **AND** 文件权限必须限制为本地用户可读写的安全级别

#### Scenario: Synced CSV remains discoverable by local parser
- **WHEN** 一次 Cursor sync 成功完成
- **THEN** 同步生成的 CSV 文件必须位于 Cursor 默认扫描树内或其明确约定的子目录内
- **AND** 无需额外转换步骤即可被 Cursor provider 重新发现和解析
