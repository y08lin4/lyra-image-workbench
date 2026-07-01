# Banana 功能拆分记录草案

> 状态：草案，供主控拆分、实现代理和验收代理对齐范围。本文只记录计划，不代表已经完成分支创建、代码删除或远程清理。
>
> 编写日期：2026-07-01
>
> 当前观察：工作区位于 `dev...origin/dev`，存在并行未提交改动；本地/远程分支列表中未发现已存在的 `banana` 分支名。后续任何分支操作都应先由主控确认基准提交。

## 1. 用户需求

用户希望把当前 `dev` 中的 Banana Nano / Banana provider 能力从主开发线拆出去：

- `dev` 回到更聚焦的 Image-2 主线，避免 Banana Key、Banana 模型规格、Banana provider 路由继续增加主线复杂度。
- Banana 相关能力不丢失，完整保存在独立 `banana` 分支，后续可以单独维护、验证或择机再合并。
- 拆分过程要有可验收边界：先建分支保留，再清理 `dev`，最后用搜索、测试、UI/API 冒烟确认主线没有残留可用 Banana 功能。
- 不混入无关重构，不覆盖并行代理的未提交改动。

## 2. 分支策略

### 2.1 基准确认

拆分前先由主控完成以下确认：

1. 确认所有并行代理的工作是否需要先提交、暂存或等待。
2. 用一次性 `safe.directory` 检查当前状态，避免写入全局 git 配置：

```powershell
git -c safe.directory=//fnos1/16T/github/lyra-image-workbench status --short --branch
```

3. 选定 Banana 保留基准提交，建议使用“清理前的 `dev` HEAD”，并在记录中写明 commit SHA。

### 2.2 创建 Banana 保留分支

在开始删除 `dev` 中任何 Banana 代码前，先从基准提交创建独立分支：

```powershell
git -c safe.directory=//fnos1/16T/github/lyra-image-workbench branch banana <split-base-sha>
git -c safe.directory=//fnos1/16T/github/lyra-image-workbench push origin banana
```

如需访问 GitHub，按仓库规则先确认代理环境变量或系统代理可用；不要裸连 GitHub，不使用强推。

### 2.3 `dev` 清理提交

`dev` 的清理应作为独立提交或一组小提交完成，提交信息使用简体中文。建议提交边界：

1. 后端删除 Banana provider、模型和 Key 路径。
2. 前端删除 Banana 选择器、设置项和展示入口。
3. 文档同步删除当前功能面中的 Banana 使用说明，只保留历史说明和本拆分记录。
4. 测试和验收补齐。

## 3. `dev` 删除范围

`dev` 的目标不是“搜索到 Banana 就机械删除”，而是删除可创建、可配置、可路由、可对外承诺的 Banana 功能面。

### 3.1 后端必须清理

- Provider/model 定义：`internal/config/config.go` 中 Banana provider、默认模型、模型白名单和校验函数。
- 任务执行链路：`internal/jobs/manager.go` 中 provider 归一化、Banana 模型规格、Banana Key 选择、Banana 参数跳过逻辑和错误文案。
- 任务类型与测试：`internal/jobs/types.go`、`internal/jobs/manager_test.go` 中 Banana provider、模型、ratio/resolution 特例。
- 用户配置：`internal/spaceconfig/store.go` 及测试中的 `BananaAPIKey`、`CloudBananaAPIKeyEnabled`、预览字段、清除字段。
- 系统配置：`internal/settings/settings.go`、`internal/settings/update.go` 及测试中的系统 Banana Key 字段、预览、清除逻辑。
- API 层：`internal/api/user_config.go`、`runtime_secrets.go`、`tasks.go`、`v1_image_tasks.go`、`developer_api_keys.go` 中接受或要求 Banana Key / provider 的路径。
- Agent 生成：`internal/agents/service.go` 中 Banana provider alias 和默认模型逻辑。

### 3.2 前端必须清理

- 模型工具：`web/src/lib/models.ts` 中 Banana provider、模型选项、规格匹配和展示标签。
- 本地 Key：`web/src/lib/localApiKeys.ts` 中 Banana Header、本地保存、清除、状态合并逻辑。
- API 契约：`web/src/types.ts`、`web/src/api/config.ts`、`web/src/api/contracts/admin.ts`、`web/src/api/contracts/tasks.ts` 中 Banana 字段和 provider 类型承诺。
- 设置与后台：`SettingsPanel.tsx`、`AdminPage.tsx`、`components/admin/SystemTab.tsx`、`components/admin/OverviewTab.tsx` 中 Banana Key 输入、清除、状态展示。
- 生图入口：`WorkbenchPage.tsx`、`GenerationPanel.tsx`、`BananaModelPicker.tsx` 中 provider 切换、Banana 规格选择、缺 Key 提示和 payload 组装。
- 结果展示：`ResultCanvas.tsx`、`TaskSidebar.tsx`、`TaskDetailModal.tsx` 中 Banana 模型标签和默认模型回退。
- 其他入口：`PromptAssistantModal.tsx`、`PromptResultPanel.tsx`、`PromptLibraryPage.tsx`、`NodeWorkflowPage.tsx`、`nodeWorkflowTemplates.ts`、`creativeCanvas/persistence.ts` 中可选择或回放 Banana provider 的路径。
- 样式：`web/src/styles.css` 中只服务于 Banana 选择器、按钮、弹窗、规格网格的样式块。

### 3.3 文档必须同步

`dev` 当前功能面文档应移除 Banana 作为可用功能的描述，包括但不限于：

- `README.md`
- `docs/API_DOCUMENTATION_SYNC.md`
- `docs/EXTERNAL_API_DESIGN.md`
- `docs/DEPLOY_BAOTA.md`
- `docs/DEPLOY_LINUX.md`
- `docs/CHANGES_FROM_AI_IMAGE_GENERATE.md`
- `docs/PROJECT_STRUCTURE.md`

历史归档文档可以保留 Banana 记录，但需要加注“历史说明，不代表当前 dev 功能面”，避免部署和 API 使用者误读。

### 3.4 允许短期兼容保留

以下内容不应作为“功能残留”直接删除，需由主控决定是否保留兼容层：

- 已有任务历史里 `provider=banana` 的只读展示。若直接删除展示映射，老任务列表可能出现空模型、异常标签或详情页崩溃。
- 已落盘配置中的 Banana Key 字段。建议在读取时忽略并在保存时不再写出，必要时提供一次性 scrub/migration，而不是在业务请求中继续使用。
- 历史文档、拆分记录、变更日志里的 Banana 字样。
- `banana` 分支自身的所有实现和文档。

## 4. `banana` 分支保留范围

`banana` 分支应保留当前清理前 `dev` 中可运行的 Banana 能力：

- 后端 Banana provider、模型白名单、模型规格到 ratio/resolution 的映射。
- 本地、云端、系统三层 Banana Key 配置和密钥预览逻辑。
- `/api/tasks`、`/v1/image-tasks`、Agent 生成中的 Banana provider 路由。
- 前端 Banana provider 选择、BananaModelPicker、规格按钮、缺 Key 提示、结果标签和任务详情展示。
- API 文档、部署文档、设置页说明中关于 Banana 的使用方式。
- 既有 Banana 相关测试，后续可以在该分支继续补齐真实 provider 的冒烟说明。

`banana` 分支后续如果继续开发，应避免从清理后的 `dev` 盲目合并删除提交；需要 cherry-pick 非 Banana 通用修复时，逐项检查冲突。

## 5. 验收清单

### 5.1 分支与提交

- [ ] 已记录 split base SHA。
- [ ] `origin/banana` 已从 split base 创建并可拉取。
- [ ] `dev` 的 Banana 清理提交不包含并行代理无关改动。
- [ ] 未执行 `git reset --hard`、`git checkout -- <file>`、`git clean -fd`、`git push --force`。

### 5.2 搜索验收

在 `dev` 清理后执行：

```powershell
rg -n -i "banana|nano-banana|banana-nano|CloudBanana|BananaAPI" .
```

期望结果只剩：

- 本文档。
- 明确标注为历史说明的归档文档。
- 必要的数据兼容/迁移注释或测试用例。

不应再出现：

- 可选 provider。
- 可填写 Banana Key 的设置项。
- 可提交 Banana 任务的 API 或前端 payload。
- 面向用户宣称 Banana 可用的部署/API 文档。

### 5.3 自动化验证

- [ ] 后端：`go test ./...` 通过。
- [ ] 前端类型检查：在 `web` 目录执行 `npx tsc --noEmit --pretty false --incremental false` 通过。
- [ ] 前端构建：在 `web` 目录执行 `npm run build` 通过。
- [ ] targeted diff 检查：`git diff --check` 通过。

### 5.4 UI/API 冒烟

- [ ] 工作台 provider 入口只显示 Image-2 或当前 dev 保留的非 Banana provider。
- [ ] 设置页不再出现 Banana Key 输入、清除按钮或状态提示。
- [ ] 后台系统配置不再出现系统 Banana Key。
- [ ] 使用 `provider=banana` 调用内部任务接口时返回明确的“不支持 provider”错误，而不是 panic、空任务或误走 Image-2。
- [ ] 使用 `/v1/image-tasks` 或 `/v1/images/generations` 时，文档和返回行为都不再承诺 Banana。
- [ ] 历史任务列表即使读到旧 `provider=banana` 记录，也不会导致页面崩溃。

## 6. 风险与注意事项

- 当前工作区已有多处未提交改动，拆分实现前必须先确认并行代理的文件所有权，避免在 `internal/jobs/manager.go`、`WorkbenchPage.tsx`、`NodeWorkflowPage.tsx` 等热点文件上互相覆盖。
- 如果先删 `dev` 再建 `banana` 分支，可能丢失完整保留点；必须先创建并推送 `banana`。
- Banana Key 属于敏感配置。清理时不能把历史 Key 打印到日志、测试输出或迁移报告中。
- 外部 API 文档当前把 `provider=banana` 写成可用能力；如果代码先删而文档未改，会造成外部调用方误用。
- 已有任务或配置文件可能包含 Banana 字段。完全删除类型字段会降低残留数据兼容性，需要在“读旧数据”和“禁止新任务”之间做清晰分层。
- `banana` 分支会和 `dev` 后续通用修复快速分叉，后续同步只能 cherry-pick 小补丁，不能无脑 merge。
- 搜索验收不能只看大小写 `Banana`，还要覆盖 `banana-nano`、`nano-banana`、`CloudBanana`、`BananaAPI` 等变体。

## 7. 建议执行顺序

1. 主控确认并冻结 split base。
2. 创建并推送 `banana` 分支。
3. 分派后端清理代理，独占 Banana 后端 provider/config/jobs/api/settings/spaceconfig 路径。
4. 分派前端清理代理，独占 Banana UI/API contract/model/styles 路径。
5. 分派文档同步代理，清理当前 dev 功能面文档中的 Banana 承诺。
6. 分派验收代理，执行搜索、Go 测试、前端类型检查/构建和 UI/API 冒烟。
7. 主控回收结果，确认 `dev` 清理提交和 `banana` 保留分支都满足验收后再推送。
