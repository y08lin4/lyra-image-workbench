package jobs

import "testing"

func TestErrorMetaMapsUnexpectedEOF(t *testing.T) {
	meta := ErrorMeta("unexpected EOF")
	if meta.Code != "E_UPSTREAM_EOF" || meta.English != "upstream_response_truncated" || meta.Chinese != "上游响应提前结束" {
		t.Fatalf("unexpected meta: %+v", meta)
	}
	result := NewResult(0, StatusFailed, "unexpected EOF")
	if result.ErrorCode != meta.Code || result.ErrorText != meta.Chinese || result.ErrorEnglish != meta.English {
		t.Fatalf("result error fields not populated: %+v", result)
	}
}

func TestErrorMetaMapsHTTPStatus(t *testing.T) {
	meta := ErrorMeta("上游请求失败：HTTP 524：timeout")
	if meta.Code != "E_UPSTREAM_GATEWAY_TIMEOUT" || meta.English != "upstream_gateway_timeout" || meta.Chinese != "上游网关等待超时" {
		t.Fatalf("unexpected meta: %+v", meta)
	}
}

func TestErrorMetaMapsCommonHTTPStatus(t *testing.T) {
	cases := []struct {
		raw     string
		code    string
		english string
		chinese string
	}{
		{"上游请求失败：HTTP 401：invalid api key", "E_UPSTREAM_AUTH", "upstream_auth_failed", "上游鉴权失败"},
		{"提示词模型请求失败：HTTP 405：method not allowed", "E_UPSTREAM_METHOD_NOT_ALLOWED", "upstream_method_not_allowed", "上游接口不支持当前请求方法或路径"},
		{"上游请求失败：HTTP 429：rate limit", "E_UPSTREAM_RATE_LIMIT", "upstream_rate_limited", "上游请求限流"},
		{"上游请求失败：HTTP 413：payload too large", "E_IMAGE_TOO_LARGE", "image_or_payload_too_large", "图片或请求体过大"},
		{"上游请求失败：HTTP 415：unsupported media type", "E_OUTPUT_FORMAT_UNSUPPORTED", "output_format_or_media_type_unsupported", "上游不支持当前图片或输出格式"},
		{"上游请求失败：HTTP 502：bad gateway", "E_UPSTREAM_GATEWAY", "upstream_gateway_error", "上游请求失败，可能触发敏感词或上游服务暂不可用"},
	}
	for _, tt := range cases {
		meta := ErrorMeta(tt.raw)
		if meta.Code != tt.code || meta.English != tt.english || meta.Chinese != tt.chinese {
			t.Fatalf("ErrorMeta(%q) = %+v", tt.raw, meta)
		}
	}
}
func TestErrorMetaMapsGIFBackendUnavailable(t *testing.T) {
	cases := []string{
		"gif backend unavailable",
		"GIF 动图生成" + "后端" + "尚未" + "接入",
		"legacy GIF backend error: GIF 动图生成" + "后端" + "尚未" + "接入",
	}
	for _, raw := range cases {
		meta := ErrorMeta(raw)
		if meta.Code != "E_GIF_LEGACY_PLACEHOLDER" || meta.English != "legacy_gif_task" || meta.Chinese != "历史 GIF 任务，请重新创建 GIF 动图任务" {
			t.Fatalf("ErrorMeta(%q) = %+v", raw, meta)
		}
		result := NewResult(0, StatusFailed, raw)
		if result.ErrorCode != meta.Code || result.ErrorEnglish != meta.English || result.ErrorText != meta.Chinese {
			t.Fatalf("NewResult(%q) did not expose GIF error display fields: %+v", raw, result)
		}
	}
}
