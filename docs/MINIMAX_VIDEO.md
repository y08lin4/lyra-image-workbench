# MiniMax 文生视频试验版

`video` 分支新增 MiniMax 文生视频入口，用来验证“用户登录 → 按额度提交视频任务 → 查询状态 → 获取下载地址”的闭环。

## 官方接口

- 文生视频：`POST https://api.minimaxi.com/v1/video_generation`
- 查询任务：`GET https://api.minimaxi.com/v1/query/video_generation?task_id=...`
- 获取文件：`GET https://api.minimaxi.com/v1/files/retrieve?file_id=...`

参考文档：<https://platform.minimaxi.com/docs/api-reference/video-generation-t2v>

## 当前实现

### 管理员侧

1. 进入 `/admin`，设置或登录 Admin。
2. 在管理配置里填写 `MiniMax API Key`。
3. 在“用户视频额度”区域选择用户名并增加额度。

注意：MiniMax Key 会保存到服务器运行时配置 `data/config.local.json`，前端只展示是否已设置和脱敏预览，不返回原始 Key。

### 用户侧

1. 用户登录后进入工作台“视频”标签。
2. 页面会读取当前剩余额度和 Admin 是否已配置 MiniMax Key。
3. 每次提交文生视频任务消耗 `1` 点额度。
4. 如果 MiniMax 创建任务失败，后端会自动退回本次消耗的额度。
5. 使用 `task_id` 查询任务状态，成功后用 `file_id` 获取 `download_url`，前端可直接预览/下载。

## 后端接口

```text
# 用户接口，需要用户登录 Cookie / Session
GET  /api/minimax/video-quota
POST /api/minimax/videos
GET  /api/minimax/videos/{taskID}
GET  /api/minimax/files/{fileID}

# Admin 接口，需要 X-Admin-Token
GET  /api/admin/config
POST /api/admin/config
GET  /api/admin/users
POST /api/admin/users/video-quota
```

`POST /api/admin/config` 可设置或清除 MiniMax Key：

```json
{
  "minimaxApiKey": "your-minimax-api-key"
}
```

```json
{
  "clearMinimaxApiKey": true
}
```

`POST /api/admin/users/video-quota` 用于给用户增加视频额度：

```json
{
  "username": "Alice_01",
  "delta": 5
}
```

`GET /api/minimax/video-quota` 返回：

```json
{
  "ok": true,
  "quota": {
    "remaining": 5,
    "costPerVideo": 1,
    "minimaxApiKeySet": true
  }
}
```

## 文生视频请求体示例

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

- 把视频任务接入统一任务队列和历史记录，记录任务归属，避免不同用户通过 `task_id` 交叉查询。
- 把生成视频落盘到 `outputs/`，避免 MiniMax 远程下载链接过期。
- 增加额度流水，记录 Admin 给谁加了多少、用户何时消耗和失败退款。
- 支持图生视频、首帧图、主体参考图等 MiniMax 其他视频接口。
