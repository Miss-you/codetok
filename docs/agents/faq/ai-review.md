# AI review FAQ

Known problems with `scripts/ai-review.sh`, acpx, and the Antigravity CLI (`agy`). Search here first when a review fails. After solving a new problem, add an entry in the same format, or update the existing one.

<a id="user-location"></a>
## `User location is not supported`

- **现象**：review 报 `FAILED_PRECONDITION: User location is not supported`。
- **原因**：代理变量已经设置，但代理当前出口节点所在地区不被 Gemini API 支持。
- **处理**：切换代理出口节点（例如美国）后重跑。脚本会识别这条错误并提示。
- **注意**：不要归因于「缺少代理变量」。缺代理时的现象是下一条，先核对失败时的 `env` 和代理日志再下结论。

<a id="model-not-recognized"></a>
## agy 等约 3.5 分钟后报 `model ... is not recognized`

- **现象**：agy 卡约 3.5 分钟后报模型不认识。
- **原因**：进程没拿到 `HTTPS_PROXY` / `HTTP_PROXY`（`env -i`、launchd、GUI 启动），直连拿不到模型列表。agy 不读 macOS 系统代理。
- **处理**：用 `scripts/ai-review.sh`，它总会自己设置代理；需要时设 `AI_REVIEW_PROXY`。

<a id="acpx-permission-denied"></a>
## acpx 中途以 `PERMISSION_DENIED` 中止

- **现象**：review 中途以 `PERMISSION_DENIED` 退出。
- **原因**：模型请求执行命令，被 `--non-interactive-permissions deny` 拒绝。
- **处理**：脚本已加 `--no-terminal` 并在 prompt 里禁止执行命令。仍复现就重跑；反复出现再收紧 prompt。

<a id="acpx-agy-acp-server"></a>
## `acpx antigravity exec` 报 `No runnable executable named "agy_acp_server.par"`

- **原因**：acpx 内置的 antigravity 适配器依赖的文件，agy 没有随包提供。
- **处理**：用 `--agent "agy acp"`，脚本已这样调用。

<a id="hydrogen-high-model-id"></a>
## acpx 不认 `gemini-4-hydrogen-high`

- **原因**：ACP 只公布不带档位后缀的模型 ID。
- **处理**：`--model gemini-4-hydrogen` 加 `--config-option thought_level=high`。`-high` 后缀只有 `agy --model` 认。

<a id="agy-p-flag"></a>
## `agy -p 'prompt' --model ...` 参数错乱

- **原因**：`-p` 和值之间用空格时会吞掉下一个参数。
- **处理**：写成 `-p='prompt'`。

<a id="api-error-retry"></a>
## 输出里有 `⚠ API error (attempt 1)`

- **现象**：acpx 输出 `⚠ API error (attempt N) ... EOF`，但末尾有完整 review 和 `[done] end_turn`。
- **原因**：agy 遇到瞬时网络错误后自动重试成功。
- **处理**：acpx 正常退出且输出最后一行是 `[done] end_turn` 时，脚本把 `⚠` 行只当警告打印，不判失败。在 `AI review:` 评论里注明发生过重试即可。最后一行不是 `[done] end_turn` 时按失败处理。

<a id="template-not-found"></a>
## `prompt template not found`

- **原因**：本机没有 superpowers 的 `requesting-code-review/code-reviewer.md`。
- **处理**：安装 superpowers 插件（Claude Code 或 Codex 均可），或用 `AI_REVIEW_TEMPLATE` 指向该文件。脚本依次查找 `~/.agents/skills/superpowers/`、`~/.codex/skills/` 和 Claude Code 插件缓存。

<a id="copilot-quota"></a>
## Copilot review 只返回 `insufficient remaining quota`

- **原因**：Copilot review 额度用完。
- **处理**：本仓库不再依赖 Copilot，按 `monitoring-pr-ai-reviews` 跑 `scripts/ai-review.sh --pr <number>`。
