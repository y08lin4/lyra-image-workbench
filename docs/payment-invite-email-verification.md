# 支付、邀请和邮件配置最小验证说明

本说明用于本地验证易支付订单闭环、邀请首充奖励和 SMTP 配置脱敏。示例默认服务地址为 `http://127.0.0.1:8080`，需要先完成用户登录并带上用户 cookie，管理员配置接口需要 `X-Admin-Token`。

## 1. 配置易支付和 SMTP

管理员写入配置：

```bash
curl -X POST http://127.0.0.1:8080/api/admin/config \
  -H "Content-Type: application/json" \
  -H "X-Admin-Token: $ADMIN_TOKEN" \
  -d '{
    "publicBaseUrl":"http://127.0.0.1:8080",
    "epayEnabled":true,
    "epayApiUrl":"https://pay.example.com/submit.php",
    "epayPid":"1001",
    "epayKey":"secret-key",
    "epayMethods":["alipay","wxpay"],
    "creditPriceCents":10,
    "minTopUpCredits":10,
    "referralRewardCredits":4,
    "smtpEnabled":true,
    "smtpHost":"smtp.example.com",
    "smtpPort":587,
    "smtpUser":"noreply@example.com",
    "smtpPassword":"smtp-secret-1234567890",
    "smtpFrom":"noreply@example.com"
  }'
```

随后读取 `GET /api/admin/config`，响应中只能看到 `epayKeySet/epayKeyPreview` 和 `smtpPasswordSet/smtpPasswordPreview`，不应出现 `epayKey` 或 `smtpPassword` 明文字段。

## 2. 创建订单和查询状态

登录用户创建订单：

```bash
curl -X POST http://127.0.0.1:8080/api/billing/epay/orders \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d '{"credits":10,"method":"alipay"}'
```

响应包含 `tradeNo`、`order.status=pending` 和 `payUrl`。查询单笔订单：

```bash
curl -b cookies.txt http://127.0.0.1:8080/api/billing/epay/orders/LYRA202606270001
# 或
curl -b cookies.txt "http://127.0.0.1:8080/api/billing/epay/orders?tradeNo=LYRA202606270001"
```

用户只能查询自己的订单；其他用户查询同一 `tradeNo` 应返回 `TOPUP_ORDER_NOT_FOUND`。

## 3. 模拟易支付回调

回调参数按易支付规则签名：排除 `sign` 和 `sign_type`，参数名排序后拼成 `k=v&...`，末尾追加商户 Key，再做 MD5 小写。测试里可复用 `billing.SignParams` 生成签名。

最小成功回调字段：

```text
pid=1001
type=alipay
out_trade_no=<tradeNo>
trade_no=E202606270001
trade_status=TRADE_SUCCESS
money=1.00
sign=<md5>
sign_type=MD5
```

发送：

```bash
curl -X POST http://127.0.0.1:8080/api/billing/epay/notify \
  -H "Content-Type: application/x-www-form-urlencoded" \
  --data "pid=1001&type=alipay&out_trade_no=$TRADE_NO&trade_no=E202606270001&trade_status=TRADE_SUCCESS&money=1.00&sign=$SIGN&sign_type=MD5"
```

成功返回纯文本 `success`。重复发送同一回调仍应返回 `success`，但购买流水只新增一次。

## 4. 验证邀请奖励

1. 邀请人登录后调用 `POST /api/users/referral-code`，响应包含 `referralCode` 和完整 `referralLink`，例如 `http://127.0.0.1:8080/?ref=ABCD1234`。
2. 被邀请用户注册时，`referralCode` 字段可以传邀请码，也可以直接传完整 `referralLink`。
3. 只注册不会奖励邀请人。
4. 被邀请用户首个订单支付成功后，系统调用现有额度流水服务：买家获得 `purchase` 流水，邀请人获得一次 `referral_reward` 流水，来源为 `referral:<tradeNo>`。
5. 被邀请用户后续重复回调或再次充值不会再次触发同一首充邀请奖励。

## 5. 推荐本地自动验证

```bash
go test ./internal/users ./internal/billing ./internal/settings ./internal/api
```
