## 1. Cursor usage 规范化

- [ ] 1.1 复核并统一 `provider/cursor` 的 CSV 字段语义，实现当前导出格式与旧版导出格式的兼容解析
- [ ] 1.2 为 Cursor CSV 解析补充稳定排序、坏行跳过、坏文件跳过与时间格式兼容测试
- [ ] 1.3 明确并验证 `Input (w/ Cache Write)`、`Input (w/o Cache Write)`、`Cache Read`、`Output Tokens` 到 `provider.TokenUsage` 的映射契约

## 2. Cursor 认证与 dashboard 同步

- [ ] 2.1 新增 `codetok cursor login`、`status`、`sync`、`logout` 子命令骨架与帮助文本
- [ ] 2.2 实现 Cursor session token 的本地凭据存储、权限控制与原子写入
- [ ] 2.3 实现 Cursor dashboard 状态校验与 usage CSV 拉取客户端，并支持可替换 base URL 以便测试
- [ ] 2.4 实现同步缓存目录写入、失败回滚与已有缓存保留逻辑
- [ ] 2.5 为认证与同步流程补充 HTTP mock / `httptest` 级别测试，覆盖成功、鉴权失败、非法响应和网络错误场景

## 3. 报表命令集成

- [ ] 3.1 抽取 `daily` / `session` 共享的 provider 收集编排逻辑，统一目录覆盖与错误处理语义
- [ ] 3.2 让默认 Cursor 报表同时发现手工导入 CSV 与同步缓存 CSV，并保持对旧平铺目录的兼容
- [ ] 3.3 保证 `--cursor-dir` 是绝对覆盖，不读取默认 Cursor 路径、同步缓存或本地凭据状态
- [ ] 3.4 为 `daily`、`session` 和 provider 过滤增加 Cursor 相关 e2e 测试，覆盖默认目录、覆盖目录、空目录和坏文件回退场景
- [ ] 3.5 更新 README、README_zh 与命令帮助文本，明确普通报表命令只读本地文件，不做隐式 sync

## 4. Cursor 活动归因能力

- [x] 4.1 设计并实现读取 `~/.cursor/ai-tracking/ai-code-tracking.db` 的独立 activity reader
- [x] 4.2 新增独立的 Cursor activity 输出模型或子命令，分别暴露 `composer` 与 `tab` 指标
- [x] 4.3 确保 activity attribution 不进入 `provider.TokenUsage`、`daily` 或 `session` 的 token 汇总路径
- [x] 4.4 为 activity attribution 补充 SQLite fixture 测试，覆盖数据库存在、缺失、单类别和双类别数据场景

## 5. 验收与回归验证

- [x] 5.1 运行格式化、静态检查和全量测试，确认 Cursor 新能力未破坏现有 provider 行为
- [ ] 5.2 补充一组端到端验收用例，覆盖“手工导入-only”“sync-only”“导入+sync 共存”“sync 失败但本地缓存仍可用”四类主路径
- [ ] 5.3 补充一组产品边界验收用例，确认 `daily` / `session` 不会隐式访问远程 Cursor API
- [x] 5.4 补充 activity attribution 边界验收用例，确认其输出不会污染 token 报表或 JSON token 字段
