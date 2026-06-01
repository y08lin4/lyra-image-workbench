# 提示词广场试验版

`dev` 分支先实现了一个最小闭环，用来验证后续“提示词广场 + GitHub 数据仓库”的方向。

## 当前已实现

- 工作台新增「广场」标签页。
- 登录用户可以提交：
  - 标题
  - 提示词
  - 模型
  - 标签
  - 比例 / 质量等基础参数
  - 图片文件或图片 URL
  - 来源链接和授权说明
- 后端会把数据保存到：

```text
data/prompt_square/items.json
data/prompt_square/images/
```

- 前端可以：
  - 搜索提示词、模型、标签；
  - 按标签筛选；
  - 复制提示词；
  - 一键把提示词填回生成页。

## API 草案

```text
GET  /api/prompt-square/items
POST /api/prompt-square/items
GET  /api/prompt-square/images/{file}
```

`POST /api/prompt-square/items` 使用 `multipart/form-data`：

| 字段 | 说明 |
| --- | --- |
| `title` | 标题，可空；为空时从提示词首行生成 |
| `prompt` | 必填，主提示词 |
| `negativePrompt` | 负面提示词，可空 |
| `model` | 模型 ID，例如 `gpt-image-2` |
| `tags` | 可重复字段；前端会按逗号/空格拆分 |
| `image` | 可选图片文件，支持 PNG/JPG/WEBP，最大 8MB |
| `imageUrl` | 可选外部图片 URL |
| `sourceUrl` | 来源链接 |
| `license` | 授权说明，例如 `user_submitted`、`CC-BY` |

## 后续接 GitHub 仓库的方向

当前 dev 版先落本地文件。后续可以把 `data/prompt_square` 同步到独立数据仓库，例如：

```text
lyra-prompt-gallery-data/
  data/prompts/YYYY/MM/{id}.json
  data/images/YYYY/MM/{id}.webp
  index/prompts.json
  index/tags.json
```

推荐流程：

```text
用户提交
  → 本服务压缩/校验/生成 JSON
  → GitHub App 或 PAT 提交到 data repo 的 pending 分支
  → 自动开 PR
  → 管理员审核合并
  → GitHub Actions 生成 index
  → 广场前端拉取 index/prompts.json
```

这样用户上传和公开展示之间有审核层，适合开源项目长期维护。

