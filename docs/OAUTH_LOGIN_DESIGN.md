# OAuth/OIDC 登录后续接入方案

本文只完善方案文档，不实现代码、不修改现有登录流程。本阶段明确不接入原生 OAuth/OIDC 登录，仍保留现有用户名/邮箱 + 密码 + 2FA 作为唯一可用登录路径；后续如果要做快捷登录，再按本文拆分为可选能力上线。

## 1. 阶段边界

- 本阶段不新增 OAuth/OIDC 后端路由、前端按钮、数据库字段、配置项或迁移脚本。
- 本阶段不接入 Google、GitHub、通用 OIDC，也不引入 Tinyauth、forward-auth 或外部认证 sidecar。
- 当前 Admin 登录、普通用户密码登录、2FA、Bearer API Key、用户空间隔离、任务和额度模型保持不变。
- 本文作为后续实现前的验收口径：实现前需要再次确认 provider 范围、自动绑定策略、自动注册策略和部署回调地址。

## 2. 后续可选范围

后续可以按业务优先级选择一个或多个 provider，不要求一次性全部上线：

| Provider | 定位 | 建议阶段 | 主身份键 |
| --- | --- | --- | --- |
| Google | OIDC Authorization Code Flow | 首批可选 | `google:<sub>` |
| GitHub | OAuth App Web Application Flow，可包装成统一外部身份 | 首批可选 | `github:<id>` |
| 通用 OIDC | 企业 IdP / 自建 IdP | 二期可选，需更严格配置审核 | `<issuer>:<sub>` |

推荐实现顺序：

1. 先落统一的外部身份抽象、state/nonce/PKCE、安全回调和用户绑定模型。
2. 再接 Google OIDC，验证 OIDC 主路径。
3. 如需要开发者生态登录，再接 GitHub OAuth，并通过 adapter 归一化为外部身份。
4. 最后再评估通用 OIDC，开放前必须补齐 issuer 固定校验、discovery URL 审核、JWKS 缓存、claim 映射、SSRF 防护和管理员回滚能力。

## 3. 推荐抽象

推荐把后续登录统一建模为 `ExternalIdentityProvider`，业务层只处理归一化后的 `ExternalIdentity`，不要让 Google/GitHub 的差异散落在用户、会话和前端逻辑里。

```ts
type ExternalIdentity = {
  provider: "google" | "github" | "oidc";
  issuer?: string;
  subject: string;
  email?: string;
  emailVerified: boolean;
  username?: string;
  displayName?: string;
  avatarUrl?: string;
};
```

Provider adapter 职责：

- 生成授权 URL，包含固定 redirect URI、state、PKCE；OIDC provider 还要包含 nonce。
- 在服务端用 code 换 token，不把 client secret、code、access token、ID token 暴露给前端。
- 校验 provider 返回的身份材料，并归一化为 `ExternalIdentity`。
- 对 Google 校验 ID Token 签名、`iss`、`aud`、`exp`、`iat`、`nonce`、`email_verified`，可选校验 `hd`。
- 对 GitHub 调用 `/user` 和 `/user/emails`，选择 verified primary email；没有可靠邮箱时按策略拒绝自动绑定或自动注册。
- 对通用 OIDC 固定 issuer，限制 discovery/JWKS 来源，不允许管理员随意配置任意 token/userinfo endpoint。

## 4. 用户表 external identities

用户表建议新增 `externalIdentities` 列表或等价关联表，用来绑定外部身份，不把 email 当作唯一登录键。

```json
{
  "externalIdentities": [
    {
      "provider": "google",
      "issuer": "https://accounts.google.com",
      "subject": "10769150350006150715113082367",
      "email": "user@example.com",
      "emailVerified": true,
      "username": "user",
      "displayName": "User Name",
      "avatarUrl": "https://...",
      "boundAt": "2026-06-29T20:00:00+08:00",
      "lastLoginAt": "2026-06-29T20:10:00+08:00"
    }
  ]
}
```

约束：

- `provider + issuer + subject` 全局唯一；Google/GitHub 可用固定 issuer 或 provider 作为 issuer。
- `email` 只用于展示、预填、人工确认和受控自动绑定，不能作为最终登录主键。
- 一个 Lyra 用户可以绑定多个外部身份；同一个外部身份只能绑定一个 Lyra 用户。
- 更新 provider profile 时只更新外部身份快照，不覆盖用户手动设置的昵称、头像或本地安全设置，除非产品另行确认。
- 自动注册出的用户仍生成独立用户空间、storage token、额度和 API Key 上下文，不能共享 provider token。

## 5. 配置项

配置建议归入现有 Admin 系统配置，默认全部关闭。Admin 读取配置时必须脱敏 secret，保存时允许只更新非 secret 字段。

```json
{
  "oauthLoginEnabled": false,
  "oauthAutoRegisterMode": "disabled",
  "oauthAutoBindVerifiedEmail": false,
  "oauthRequireLocal2FAForBoundUsers": true,
  "oauthProviders": {
    "google": {
      "enabled": false,
      "clientId": "",
      "clientSecret": "",
      "redirectUri": "",
      "scopes": ["openid", "email", "profile"],
      "hostedDomainAllowlist": [],
      "requireVerifiedEmail": true,
      "displayName": "Google"
    },
    "github": {
      "enabled": false,
      "clientId": "",
      "clientSecret": "",
      "redirectUri": "",
      "scopes": ["read:user", "user:email"],
      "requireVerifiedEmail": true,
      "displayName": "GitHub"
    },
    "oidc": {
      "enabled": false,
      "issuer": "",
      "clientId": "",
      "clientSecret": "",
      "redirectUri": "",
      "scopes": ["openid", "email", "profile"],
      "allowedEmailDomains": [],
      "requireVerifiedEmail": true,
      "displayName": "OIDC"
    }
  }
}
```

关键默认值：

| 字段 | 默认值 | 说明 |
| --- | --- | --- |
| `oauthLoginEnabled` | `false` | 全局开关；关闭后登录页不展示入口，start/callback 也拒绝。 |
| `oauthAutoRegisterMode` | `disabled` | 自动注册策略：`disabled`、`invite_required`、`enabled`。 |
| `oauthAutoBindVerifiedEmail` | `false` | 是否允许 verified email 自动绑定唯一匹配账号，默认关闭。 |
| `oauthRequireLocal2FAForBoundUsers` | `true` | 已绑定本地账号且开启 2FA 时，OAuth 后仍要求 TOTP。 |
| `clientSecret` | 空 | 只写入、不明文读取；Admin GET 只返回 `clientSecretSet` 和可选 preview。 |
| `redirectUri` | 空 | 为空时由 `publicBaseUrl` 推导；生产环境必须使用 HTTPS。 |

## 6. 回调 URL

推荐固定回调路径，provider 控制台和后端配置必须完全一致：

```text
GET /api/auth/oauth/{provider}/start?next=/workbench
GET /api/auth/oauth/{provider}/callback?code=...&state=...
```

回调 URL 规则：

- `{provider}` 只能是后端 allowlist 中的固定 ID，例如 `google`、`github`、`oidc`，不能来自任意管理员输入。
- `redirect_uri` 必须在发起登录和换 token 时保持一致。
- 生产环境必须是 HTTPS；本地开发可以允许 localhost HTTP。
- `publicBaseUrl` 需要明确反向代理后的外部地址，避免生成内网地址或错误 scheme。
- `next` 只接受同源相对路径，例如 `/`、`/workbench`、`/settings`；拒绝 `https://evil.example`、`//evil.example` 和反斜杠变体。

## 7. state、nonce 和 PKCE

- `state` 使用至少 128 bit 随机数，推荐 256 bit base64url；服务端只保存 hash，TTL 建议 10 分钟。
- `state` 绑定 provider、redirect URI、next、用途 login/bind、PKCE code verifier hash、创建时间和客户端 cookie。
- 回调开始时必须先校验并一次性消费 state；缺失、过期、不匹配或重复使用都失败。
- OIDC provider 使用独立 `nonce`，保存 hash，并在 ID Token 中校验 `nonce` claim。
- GitHub 没有 OIDC nonce，依赖 state + PKCE + 后端 token exchange。
- 所有 provider 都使用 PKCE S256，即使是后端 confidential client。
- state cookie 使用 `HttpOnly`、`SameSite=Lax`；生产 HTTPS 下必须加 `Secure`。

## 8. 登录、绑定和账户合并

### 8.1 登录判定顺序

1. 用 `provider + issuer + subject` 查找已绑定用户，找到则进入该用户的 Lyra 会话。
2. 如果用户已登录并从设置页发起绑定，则把外部身份绑定到当前用户。
3. 如果开启 `oauthAutoBindVerifiedEmail`，且 provider 返回 verified email，并且 Lyra 中只有一个同邮箱账号，则绑定到该账号。
4. 如果 `oauthAutoRegisterMode` 允许自动注册，则创建新用户并绑定外部身份。
5. 否则拒绝登录，提示先用现有账号登录后手动绑定。

### 8.2 账户合并边界

- 不做静默多账号合并；邮箱相同只能作为候选信号，不能直接覆盖或迁移另一个账号的数据。
- 多个 Lyra 账号共享同一邮箱、邮箱未验证、provider 未返回邮箱时，必须要求人工选择或先本地登录绑定。
- 合并用户空间、任务、额度、API Key、历史记录属于单独高风险迁移，不随 OAuth 登录自动执行。
- 如果后续产品需要“合并账号”功能，应独立设计审核、数据归属、回滚和审计流程。

### 8.3 解绑规则

- 设置页可展示 `已绑定登录方式`，允许绑定和解绑外部身份。
- 绑定/解绑敏感操作建议要求当前会话重新通过密码或 TOTP。
- 解绑最后一种可登录方式前，必须确认账号已有本地密码，或仍绑定另一个可用 provider。
- 同一个外部身份重复绑定到不同用户时必须失败并写入审计日志。

## 9. 安全边界

- client secret、authorization code、provider access token、ID Token、refresh token 不进入前端、URL、普通日志或错误提示。
- 默认不请求业务无关 scope，例如 Google Drive、GitHub repo。
- 默认不保存 provider refresh token，不做离线访问；如未来需要保存，必须加密、记录用途并提供撤销路径。
- OAuth/OIDC 只负责证明外部身份，不替代 Lyra 用户、权限、额度、任务、空间隔离和 Bearer API Key 模型。
- 绑定到已开启本地 2FA 的账号时，OAuth 成功后仍进入本地 TOTP 二阶段。
- provider token exchange、identity fetch、JWKS/discovery 请求需要超时、限流和错误熔断。
- 通用 OIDC 禁止任意 URL 直连；必须限制 issuer/discovery/JWKS host，避免 SSRF。
- 登录尝试、state 失败、token exchange 失败、绑定/解绑、自动注册和自动绑定都应进入审计日志。

## 10. 迁移步骤

后续真正实现时建议按以下步骤推进，每一步都可独立验收和回滚：

1. 新增配置结构但默认关闭，不改变登录页展示和现有登录行为。
2. 新增 `externalIdentities` 存储和唯一索引，执行空迁移或兼容旧用户的懒加载迁移。
3. 实现 state/nonce/PKCE store、TTL、一次性消费、限流和审计。
4. 实现统一 provider adapter 接口，先接 Google OIDC 或单个目标 provider。
5. 实现 start/callback 路由，但保持全局开关关闭，并补齐路由测试。
6. 实现绑定/解绑 API 和设置页入口，先允许已登录用户手动绑定。
7. 再按策略开放登录页 provider 按钮、自动绑定和自动注册。
8. 最后补部署文档，说明 provider 控制台配置、回调 URL、HTTPS、反向代理 header、secret 轮换和回滚步骤。

## 11. 验收清单

- 本阶段代码零改动：未新增 OAuth/OIDC 路由、UI、配置迁移或登录实现。
- 全局默认关闭时，登录页不展示 provider，start/callback 返回禁用错误。
- Google/OIDC 校验 `iss`、`aud`、签名、`exp`、`iat`、`nonce`、`email_verified`，可选校验域限制。
- GitHub 换 token 后重新读取 `/user` 和 verified primary email，不信任前端提交的邮箱。
- state 缺失、过期、不匹配、重复使用均失败；PKCE code verifier 不匹配失败。
- `next=https://evil.example`、`//evil.example`、反斜杠和编码变体被拒绝或降级到 `/`。
- 已绑定外部身份登录到同一个 Lyra 用户，不创建新用户。
- `oauthAutoBindVerifiedEmail=false` 时，即使邮箱匹配也不自动绑定。
- 自动绑定只在 verified email 唯一匹配时发生；多账号同邮箱或邮箱未验证均失败。
- 自动注册关闭时，未绑定外部身份不能创建用户。
- `invite_required` 模式下，OAuth 不绕过邀请码或 referral 门槛。
- 已开启本地 2FA 的用户，OAuth 后仍需 TOTP 才能进入工作台。
- 解绑最后登录方式被阻止。
- 日志中不出现 access token、ID Token、authorization code、client secret。
- 通用 OIDC 不允许任意 endpoint，issuer/discovery/JWKS 均受 allowlist 或固定配置约束。

## 12. 实现前待确认

- 首批是否只做 Google OIDC，还是同时做 GitHub。
- 是否需要通用 OIDC；如需要，允许的 issuer 和企业域白名单是什么。
- 是否允许自动注册；默认建议 `disabled` 或 `invite_required`。
- 是否允许 verified email 自动绑定；默认建议关闭，只开放用户手动绑定。
- 自部署环境的 `publicBaseUrl`、HTTPS、反向代理 header 和最终回调 URL 口径。

## 13. 参考资料

- Google OpenID Connect: https://developers.google.com/identity/openid-connect/openid-connect
- GitHub OAuth App web flow: https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/authorizing-oauth-apps
- GitHub REST API emails: https://docs.github.com/en/rest/users/emails
- OpenID Connect Core 1.0: https://openid.net/specs/openid-connect-core-1_0.html

