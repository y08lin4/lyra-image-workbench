# 个人空间与第一版用户设计

本设计参考 `ai-image-generate-private` 的“空间密码”思路，但针对本机 Go 后端做调整：个人空间用于隔离任务、历史、上传参考图和统计；NewAPI Key 与管理配置仍由本机后端统一保存，不让前端直接请求 NewAPI。

## 1. 目标

- 用户进入工作台前先输入自己设置的空间密码。
- 相同空间密码进入同一个个人空间。
- 不同空间密码隔离任务、历史、参考图和统计。
- 浏览器只保存不可逆派生后的空间令牌，不保存明文空间密码。
- 后端按 `X-Space-Token` 定位空间。
- 个人空间不是公网账号系统，不做注册、找回密码、云同步。

## 2. 和参考项目的差异

参考项目：

```text
空间密码 -> 前端派生 identityToken -> Worker 用 owner_hash 隔离 D1 任务
```

本项目：

```text
空间密码 -> Go 后端校验并派生 space token -> data/spaces/{token}/ 隔离本机数据
```

差异点：

- 本项目是本机程序，空间用于本机多用户/多项目隔离，不是云端账号。
- NewAPI Key 不跟空间密码混在前端保存；Key 仍由后端管理配置保存。
- 后续任务、上传、历史、统计接口都要要求 `X-Space-Token`。

## 3. 当前接口

```text
POST   /api/spaces/session
GET    /api/spaces/session
DELETE /api/spaces/session
```

### 创建或进入空间

```http
POST /api/spaces/session
Content-Type: application/json

{
  "password": "用户自己设置的复杂空间密码"
}
```

响应：

```json
{
  "ok": true,
  "session": {
    "space": {
      "id": "64位不可逆 token",
      "displayName": "个人空间 abcd1234",
      "createdAt": "2026-05-09T...",
      "lastOpenedAt": "2026-05-09T..."
    },
    "token": "64位不可逆 token",
    "tokenPreview": "abcd1234…wxyz",
    "created": true
  }
}
```

前端保存 `session.token`，之后请求带：

```http
X-Space-Token: 64位不可逆 token
```

### 查询当前空间

```http
GET /api/spaces/session
X-Space-Token: ...
```

用于刷新页面后恢复空间状态。

### 退出空间

```http
DELETE /api/spaces/session
```

后端不删除空间，只提示前端清理本地 token。删除空间数据后续单独做危险操作确认。

## 4. 密码规则

沿用参考项目的复杂度思路：

- 至少 10 位。
- 不能全空格。
- 不能同一字符重复。
- 不能大量连续重复字符。
- 不能连续数字/字母。
- 不能重复片段。
- 不能键盘顺序。
- 不能常见弱密码词。
- 不能明显日期。
- 建议至少包含大小写字母、数字、符号中的三类。

## 5. 本机存储布局

```text
data/
  config.local.json              # 管理配置：NewAPI URL、600 秒超时、固定模型等
  spaces/
    {spaceToken}/
      space.json                 # 空间元信息
      jobs.json                  # 后续任务状态
      uploads/                   # 图生图参考图
      history.json               # 后续历史索引
      stats.json                 # 后续统计
outputs/
  {spaceToken}/YYYY-MM-DD/        # 后续空间隔离输出图
```

说明：

- `config.local.json` 是管理员级配置，不按空间隔离。
- 任务和图片按空间隔离。
- 后续如果开启 LAN 模式，必须增加访问口令或配对机制，否则任何局域网用户都可能进入/创建空间。

## 6. 第一版模型策略

第一版先固定参考项目原模型：

```text
gpt-image-2
```

`/api/admin/config` 暂时只允许设置：

- NewAPI 请求 URL。
- 超时时间，默认 600 秒。

接口返回 `modelLocked: true`，前端只展示模型，不提供编辑。后续再增加模型列表、模型测试和默认模型选择。

## 7. 图生图第一版纳入范围

第一版同时做文生图和图生图。图生图闭环：

```text
前端上传参考图
  -> POST /api/uploads/reference，带 X-Space-Token
     Content-Type: multipart/form-data
     字段名支持 image 或 image[]；单张最大 12MB，总请求最大 50MB，最多 8 张
  -> Go 保存到 data/spaces/{token}/uploads/
  -> 创建任务时传 uploadIds
  -> Go 读取本地参考图
  -> multipart 请求 {NewAPI}/images/edits
  -> 输出图保存到 outputs/{token}/...
```

不走 PiXhost，不让前端拿参考图远程 URL，不把参考图上传第三方。

## 8. 后续所有任务接口必须套空间

这些接口后续必须要求 `X-Space-Token`：

```text
POST /api/background-tasks
GET  /api/background-tasks
GET  /api/background-tasks/:id
GET  /api/background-tasks/:id/events
POST /api/background-tasks/:id/retry
POST /api/background-tasks/:id/cancel
GET  /api/background-tasks/:id/images/:index
POST /api/uploads/reference
GET  /api/stats
```

`/api/admin/config` 是否要求空间令牌后续再定。如果只监听 `127.0.0.1` 可以先不强制；如果开放 LAN，admin 必须加管理密码或配对 token。

