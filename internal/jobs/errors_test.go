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
	}
	for _, tt := range cases {
		meta := ErrorMeta(tt.raw)
		if meta.Code != tt.code || meta.English != tt.english || meta.Chinese != tt.chinese {
			t.Fatalf("ErrorMeta(%q) = %+v", tt.raw, meta)
		}
	}
}
