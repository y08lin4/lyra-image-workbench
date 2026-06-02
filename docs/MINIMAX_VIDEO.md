# MiniMax 文生视频试验版

`video` 分支新增了 MiniMax 文生视频入口，用于验证视频生成闭环。

## 官方接口

- 文生视频：`POST https://api.minimaxi.com/v1/video_generation`
- 查询任务：`GET https://api.minimaxi.com/v1/query/video_generation?task_id=...`
- 获取文件：`GET https://api.minimaxi.com/v1/files/retrieve?file_id=...`

参考文档：

- <https://platform.minimaxi.com/docs/api-reference/video-generation-t2v>

## 当前实现

前端新增「视频」标签页：

1. 在浏览器本地保存 MiniMax API Key。
2. 输入视频提示词、模型、时长、分辨率。
3. 后端代理请求 MiniMax 创建任务。
4. 使用 `task_id` 查询任务状态。
5. 成功后使用 `file_id` 获取 `download_url`，前端直接预览/下载。

后端接口：

```text
POST /api/minimax/videos
GET  /api/minimax/videos/{taskID}
GET  /api/minimax/files/{fileID}
```

MiniMax Key 通过请求头临时传给后端：

```text
X-Image-Workbench-Minimax-API-Key
```

当前分支不把 MiniMax Key 保存到服务器，只保存在当前浏览器 `localStorage`。

## 请求体示例

```json
{
  "model": "MiniMax-Hailuo-02",
  "prompt": "一只白猫在雨夜霓虹街道中缓慢向镜头走来，电影感，浅景深",
  "duration": 6,
  "resolution": "1080P",
  "prompt_optimizer": true,
  "fast_pretreatment": false,
  "aigc_watermark": false
}
```

## 后续建议

- 把视频任务也接入统一任务队列和历史记录。
- 把生成的视频落盘到 `outputs/`，避免远程下载链接过期。
- 支持图生视频、首帧图、主体参考图等 MiniMax 其他视频接口。
- 在设置页加入 MiniMax Key 的本地/云端保存策略。

