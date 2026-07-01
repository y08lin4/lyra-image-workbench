# SQLite 迁移计划

## 目标和边界

本文档描述将当前 JSON store 逐步迁移到 SQLite 的执行计划。目标是在不引入额外数据库服务、不改变现有业务语义的前提下，先把高频读写、需要事务一致性、需要查询索引的数据域迁入 SQLite，同时保留可回滚到 JSON store 的路径。

迁移初期不追求一次性重构所有持久化逻辑。优先处理以下数据域：

- jobs：任务队列、生成记录、状态流转、重试和错误信息。
- users：用户身份、配额、偏好设置、权限或会话关联信息。
- billing：套餐、额度、消耗流水、支付回调幂等记录。
- canvas：画布、图层、节点、资源引用、编辑快照和版本信息。

非目标：

- 不在第一阶段引入 PostgreSQL、MySQL、Redis 等额外基础设施。
- 不把所有大对象直接塞入数据库；图片、缩略图、导出文件仍应保留在文件存储或对象存储，SQLite 仅保存元数据和引用。
- 不要求一次性删除 JSON store；迁移期间 JSON store 作为备份、回滚和对账来源保留。

## JSON Store 清单

当前迁移对象按数据职责归纳为以下 JSON store 类别。最终实施时应以现有仓库中的实际 store 文件、读写入口和调用链为准做映射表。

| 数据域 | 典型 JSON 内容 | SQLite 目标 | 迁移优先级 |
| --- | --- | --- | --- |
| jobs | 任务 ID、用户 ID、输入参数、状态、进度、结果路径、错误、时间戳 | 任务表、状态历史、幂等键、结果元数据 | 高 |
| users | 用户 ID、账号信息、偏好、角色、配额快照 | 用户表、偏好表、配额表 | 高 |
| billing | 套餐、订单、支付事件、额度变更、消费记录 | 订单表、支付事件表、额度流水表 | 高 |
| canvas | 画布配置、图层、节点、资源引用、版本快照 | 画布表、图层表、节点表、资源引用表、版本表 | 中 |
| settings | 全局配置、功能开关、本地运行配置 | 保留 JSON 或迁至 key-value 表 | 低 |
| cache | 可重建缓存、临时索引、派生数据 | 原则上不迁移，必要时建缓存表 | 低 |
| audit/log | 操作日志、调试输出、回调日志 | 关键审计入库，普通日志继续走日志系统 | 中 |

JSON store 到 SQLite 的映射建议保持一张迁移清单：

- store_name：原 JSON store 名称或逻辑名称。
- owner_domain：jobs、users、billing、canvas 等数据域。
- read_paths：主要读取入口。
- write_paths：主要写入入口。
- sqlite_tables：目标表。
- migration_status：pending、dual_write、backfilled、read_switched、json_retired。
- rollback_source：回滚时使用的 JSON 备份位置。

## 为什么 SQLite 暂时够用

SQLite 适合当前阶段的原因：

- 部署简单：不需要独立数据库进程，适合桌面、本地服务、小团队或单实例部署。
- 事务能力足够：能用 ACID 事务包住任务状态、额度扣减、支付回调幂等等关键写入。
- 查询能力明显强于 JSON：可以按用户、状态、时间、幂等键、画布更新时间等维度建索引。
- 运维成本低：备份、恢复、迁移文件都比引入远程数据库更轻。
- 迁移风险可控：可以和 JSON store 并行运行，支持双写、对账、读路径灰度切换。

需要承认的限制：

- 高并发多写场景有限。SQLite 适合读多写少或单实例串行化写入，不适合作为长期的高并发中心数据库。
- 不适合保存大型二进制图片。数据库中只保存路径、哈希、尺寸、MIME、生成参数摘要等元数据。
- 横向扩展能力有限。如果后续出现多实例部署、远程协作、高写入吞吐或复杂报表需求，应再评估 PostgreSQL。

因此本计划将 SQLite 定位为“下一阶段可靠本地持久化层”，不是最终形态的承诺。

## 迁移阶段

### 阶段 0：准备和约束

- 定义 SQLite 文件位置、备份目录、迁移版本表和 schema 初始化流程。
- 建立 `schema_migrations` 表记录已执行迁移版本。
- 给每个 JSON store 建立逻辑映射清单。
- 明确所有大对象继续外置存储，数据库只保存引用。
- 为 jobs、billing 设计幂等键，避免迁移期间重复创建任务或重复扣费。

交付物：

- 初始 schema。
- JSON store 映射清单。
- 数据库打开、迁移、备份、恢复的统一入口。

### 阶段 1：只读回填

- 从 JSON store 扫描历史数据，写入 SQLite。
- 回填过程只读 JSON，不改变线上读写路径。
- 每批数据记录迁移进度，支持断点续跑。
- 回填后按数据域输出对账结果，例如记录数、主键集合、关键字段哈希。

交付物：

- 回填脚本或迁移命令。
- 每个数据域的回填报告。
- 可重复执行且不会重复插入的 upsert 逻辑。

### 阶段 2：双写

- 写路径同时写 JSON store 和 SQLite。
- 以现有 JSON store 为主读，SQLite 作为影子库。
- 对 jobs、billing 这种关键域，写入 SQLite 应使用事务；JSON 写入失败或 SQLite 写入失败都要记录补偿任务。
- 建立后台对账，定期比较 JSON 和 SQLite 的关键字段。

交付物：

- 双写适配层。
- 对账报告。
- 补偿队列或人工修复流程。

### 阶段 3：读路径灰度切换

- 对低风险查询先切 SQLite 读取，例如任务列表、画布列表、历史记录查询。
- 单条关键详情读取可以先采取 SQLite 主读、JSON 兜底。
- 出现 SQLite 缺失、反序列化失败或 schema 不兼容时，自动回退 JSON 并记录告警。
- 灰度完成后，jobs、users、billing、canvas 分域切换主读。

交付物：

- 分域读开关。
- SQLite 主读路径。
- JSON 兜底和告警指标。

### 阶段 4：停止 JSON 主写

- SQLite 成为主写。
- JSON store 改为只读备份或导出快照。
- 保留一段时间的 JSON 导出能力，便于回滚和人工排查。
- 关键事件继续保留 append-only 审计记录，避免误删后无法追溯。

交付物：

- SQLite 主写路径。
- JSON 备份导出。
- 回滚演练结果。

### 阶段 5：清理和长期治理

- 删除已经稳定替代的 JSON 写入路径。
- 保留必要的导入、导出、备份、压缩和 vacuum 流程。
- 建立 schema 变更规范：向前兼容、可回滚、禁止无备份破坏性迁移。
- 观察 SQLite 文件大小、查询耗时、锁等待、备份耗时。

交付物：

- 清理后的持久化边界。
- 运维 runbook。
- 后续是否迁移到 PostgreSQL 的评估指标。

## Schema 边界

### 通用表

建议所有核心表包含以下字段：

- id：稳定主键，优先沿用现有业务 ID。
- created_at：创建时间。
- updated_at：更新时间。
- deleted_at：软删除时间，可选。
- version：乐观锁版本号，可选。
- metadata_json：少量不参与高频查询的扩展字段。

建议建立：

- `schema_migrations(version, applied_at, checksum)`
- `idempotency_keys(scope, key, target_type, target_id, created_at, expires_at)`
- `outbox_events(id, topic, payload_json, status, retry_count, created_at, processed_at)`

### Jobs 边界

核心职责：

- 保存任务主体和状态。
- 保存任务输入摘要、结果引用、错误信息。
- 支持按用户、状态、时间查询。
- 支持任务重试和幂等创建。

建议表：

- `jobs`
  - `id`
  - `user_id`
  - `type`
  - `status`
  - `priority`
  - `input_json`
  - `input_hash`
  - `result_json`
  - `error_code`
  - `error_message`
  - `attempt_count`
  - `created_at`
  - `started_at`
  - `finished_at`
  - `updated_at`
- `job_events`
  - `id`
  - `job_id`
  - `from_status`
  - `to_status`
  - `message`
  - `metadata_json`
  - `created_at`

边界规则：

- job 状态流转必须在事务中完成。
- 大图、生成结果文件、日志文件只保存引用，不直接入库。
- `job_events` 作为审计历史，不替代 `jobs.status` 的当前状态。

### Users 边界

核心职责：

- 保存用户身份和业务配置。
- 支持配额、角色、偏好和状态查询。
- 不保存明文敏感凭据。

建议表：

- `users`
  - `id`
  - `external_id`
  - `email`
  - `display_name`
  - `status`
  - `created_at`
  - `updated_at`
- `user_preferences`
  - `user_id`
  - `key`
  - `value_json`
  - `updated_at`
- `user_quotas`
  - `user_id`
  - `quota_type`
  - `balance`
  - `reserved`
  - `updated_at`

边界规则：

- 用户偏好可以是 key-value，但参与查询的字段应提升为显式列。
- 配额变更必须通过 billing 或 quota ledger 记录，不直接覆盖余额后丢失原因。
- 密钥、token、支付敏感信息不要进入普通 SQLite 明文字段；如必须保存，应使用系统凭据存储或加密方案。

### Billing 边界

核心职责：

- 保存订单、支付事件、额度流水。
- 确保支付回调幂等。
- 确保额度扣减和任务创建之间的一致性。

建议表：

- `billing_orders`
  - `id`
  - `user_id`
  - `provider`
  - `provider_order_id`
  - `status`
  - `amount`
  - `currency`
  - `plan_code`
  - `created_at`
  - `paid_at`
  - `updated_at`
- `billing_events`
  - `id`
  - `provider`
  - `event_id`
  - `event_type`
  - `payload_hash`
  - `payload_json`
  - `processed_at`
  - `created_at`
- `quota_ledger`
  - `id`
  - `user_id`
  - `source_type`
  - `source_id`
  - `delta`
  - `balance_after`
  - `reason`
  - `created_at`

边界规则：

- `provider + event_id` 必须唯一。
- 额度变更使用 append-only ledger，再更新余额快照。
- 支付回调处理必须先记录事件，再在同一事务中更新订单和额度。
- 退款、撤销、补偿都应以负向流水表达，不直接删除历史流水。

### Canvas 边界

核心职责：

- 保存画布主体、图层、节点、资源引用和版本。
- 支持最近编辑、按用户列出、按画布加载。
- 支持未来的版本恢复和局部编辑。

建议表：

- `canvases`
  - `id`
  - `user_id`
  - `title`
  - `status`
  - `width`
  - `height`
  - `background_json`
  - `created_at`
  - `updated_at`
- `canvas_layers`
  - `id`
  - `canvas_id`
  - `name`
  - `position`
  - `visible`
  - `locked`
  - `opacity`
  - `blend_mode`
  - `data_json`
  - `created_at`
  - `updated_at`
- `canvas_nodes`
  - `id`
  - `canvas_id`
  - `layer_id`
  - `type`
  - `position_x`
  - `position_y`
  - `width`
  - `height`
  - `rotation`
  - `data_json`
  - `created_at`
  - `updated_at`
- `canvas_assets`
  - `id`
  - `canvas_id`
  - `asset_type`
  - `uri`
  - `sha256`
  - `width`
  - `height`
  - `mime_type`
  - `metadata_json`
  - `created_at`
- `canvas_versions`
  - `id`
  - `canvas_id`
  - `version`
  - `snapshot_json`
  - `created_by`
  - `created_at`

边界规则：

- 高频查询字段使用显式列，复杂编辑状态可暂存 `data_json`。
- 完整画布快照放在 `canvas_versions`，当前可编辑状态放在规范化表。
- 资源文件不入库，只保存 URI、哈希和尺寸等元数据。
- 大型 snapshot 需要设置压缩或归档策略，避免 SQLite 文件快速膨胀。

## 索引建议

通用建议：

- 每个外键列建立索引。
- 所有唯一幂等键建立唯一索引。
- 高频列表页使用组合索引，按过滤条件在前、排序字段在后。
- 避免给低选择性的布尔字段单独建索引，必要时使用组合索引。
- 对 JSON 字段不要依赖运行时全文扫描；需要查询的属性应提升为列。

建议索引：

- jobs
  - `unique(id)`
  - `index(user_id, created_at desc)`
  - `index(status, priority, created_at)`
  - `index(user_id, status, created_at desc)`
  - `unique(input_hash)`：仅在业务确认同输入必须幂等时使用。
- job_events
  - `index(job_id, created_at)`
- users
  - `unique(id)`
  - `unique(external_id)`：如果存在外部身份。
  - `unique(email)`：如果邮箱是登录标识。
  - `index(status, created_at)`
- user_preferences
  - `unique(user_id, key)`
- user_quotas
  - `unique(user_id, quota_type)`
- billing_orders
  - `unique(provider, provider_order_id)`
  - `index(user_id, created_at desc)`
  - `index(status, updated_at)`
- billing_events
  - `unique(provider, event_id)`
  - `index(processed_at)`
- quota_ledger
  - `index(user_id, created_at desc)`
  - `index(source_type, source_id)`
- canvases
  - `index(user_id, updated_at desc)`
  - `index(status, updated_at desc)`
- canvas_layers
  - `index(canvas_id, position)`
- canvas_nodes
  - `index(canvas_id, layer_id)`
  - `index(canvas_id, type)`
- canvas_assets
  - `index(canvas_id)`
  - `index(sha256)`
- canvas_versions
  - `unique(canvas_id, version)`
  - `index(canvas_id, created_at desc)`

## 事务和幂等策略

### 事务边界

必须使用事务的场景：

- 创建 job，同时扣减或预留额度。
- job 状态从 queued 到 running、running 到 succeeded 或 failed。
- 支付回调处理：记录事件、更新订单、写额度流水、更新余额快照。
- 保存 canvas 当前状态和版本快照。
- 删除或归档 canvas，同时更新关联资源引用状态。

建议事务模式：

- 短事务：事务中只做数据库读写，不做图片生成、网络请求、文件上传等耗时操作。
- 先记录意图：需要异步副作用时，先在事务内写 `outbox_events`，事务提交后再处理外部动作。
- 明确失败状态：外部动作失败后回写 failed 或 retryable 状态，而不是让数据停在未知状态。

### 幂等键

建议幂等键范围：

- job 创建：`scope = job_create`，key 可以来自客户端请求 ID、输入哈希或业务 request id。
- 支付回调：`scope = billing_event`，key 使用 `provider + event_id`。
- 额度扣减：`scope = quota_debit`，key 使用 `source_type + source_id`。
- canvas 保存：`scope = canvas_save`，key 使用客户端保存操作 ID 或版本号。

幂等处理规则：

- 收到请求后先尝试插入幂等键。
- 插入成功才执行主体逻辑。
- 如果唯一约束冲突，读取已有 target 并返回相同结果。
- 幂等键应保存 target_type、target_id、创建时间和过期时间。

### Upsert 和断点续跑

回填脚本和双写补偿都必须支持重复执行：

- 使用稳定业务 ID 作为主键或唯一键。
- 对不可变历史事件使用 `insert or ignore`。
- 对当前状态快照使用带版本检查的 upsert。
- 每批回填记录 checkpoint，例如 store 名称、最后处理主键、处理数量、校验哈希。

## 风险和回滚

### 主要风险

- 数据不一致：双写期间 JSON 和 SQLite 有一边写入失败。
- schema 边界过早固化：把仍在快速变化的 JSON 结构拆得太细，增加迁移成本。
- 事务过大：一次保存大量 canvas snapshot 或批量回填导致锁等待和 UI 卡顿。
- 文件膨胀：频繁更新、大型 JSON snapshot、审计表增长导致 SQLite 文件变大。
- 并发写冲突：多个 worker 同时写任务状态或额度余额。
- 敏感信息落库：支付 payload、token、用户凭据未经筛选直接进入数据库。
- 回滚不可用：切读后才发现 JSON 备份缺字段或已停止更新。

### 缓解措施

- 分域迁移，不跨 jobs、users、billing、canvas 一次性切换。
- 双写阶段保留 JSON 主读，SQLite 只做影子对账。
- 为关键写入建立补偿队列和人工修复命令。
- 使用 WAL 模式、短事务、批量回填限速。
- 对 billing 和 quota 使用 append-only ledger，避免只保留最终余额。
- 对大 JSON 和版本快照设置保留策略。
- 切换前执行回滚演练，确认 JSON 仍能支撑主读。

### 回滚策略

阶段 0 和阶段 1：

- 直接删除或忽略 SQLite 文件即可。
- JSON store 仍是唯一主数据源。

阶段 2：

- 关闭双写开关，恢复 JSON-only。
- 保留 SQLite 文件用于排查，不作为主数据源。
- 对照双写失败日志，补齐 JSON 或 SQLite 中缺失的数据。

阶段 3：

- 关闭分域 SQLite 读开关，恢复 JSON 主读。
- 对已发现不一致的数据域暂停继续切换。
- 使用对账报告定位缺失记录，修复后再重新灰度。

阶段 4：

- 如果 JSON 备份仍在持续导出，可将读写切回 JSON，并从 SQLite 导出切换窗口内新增记录回补 JSON。
- 如果 JSON 已停止更新，必须先执行 SQLite 到 JSON 的反向导出，再切回 JSON。
- billing 回滚必须优先保护支付事件和额度流水，不允许用旧 JSON 快照覆盖新支付结果。

阶段 5：

- 若 JSON 写入路径已删除，回滚成本较高，应按事故恢复处理。
- 使用 SQLite 备份文件、迁移前快照和导出工具恢复。
- 只有在确认数据导出完整、关键业务可读写后，才允许切回旧版本。

## 验收标准

- 每个数据域都有明确 JSON store 到 SQLite 表的映射。
- 回填脚本可重复执行，重复执行不会产生重复记录。
- jobs、billing 的关键写入具备事务和幂等保护。
- 支持按开关分域切换读路径。
- 支持关闭 SQLite 读写并恢复 JSON 主路径。
- 至少完成一次从 SQLite 备份恢复的演练。
- 对账报告覆盖记录数、主键集合和关键字段摘要。

## 后续决策点

满足以下任一条件时，应重新评估是否升级到 PostgreSQL：

- 出现多实例同时写入同一个数据库文件。
- job、billing 或 canvas 的写入吞吐持续增长，锁等待成为瓶颈。
- 需要复杂权限、协作编辑、跨用户共享和强一致多人实时状态。
- 需要远程访问、集中备份、报表分析或更强的查询能力。
- SQLite 文件大小、备份耗时或恢复耗时超过可接受范围。
