package spaces

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"regexp"
	"strings"
)

const (
	PasswordMinLength = 10
	HashPrefix        = "image-workbench-space:v1:"
)

var derivedTokenPattern = regexp.MustCompile(`^[a-f0-9]{64}$`)

func ValidatePassword(value string) error {
	password := strings.TrimSpace(value)
	if len(password) < PasswordMinLength {
		return NewValidationError("SPACE_PASSWORD_TOO_SHORT", "空间密码至少需要 10 位")
	}

	compact := regexp.MustCompile(`\s+`).ReplaceAllString(password, "")
	lower := strings.ToLower(compact)
	if compact == "" {
		return NewValidationError("SPACE_PASSWORD_BLANK", "空间密码不能只包含空格")
	}
	if allSame(compact) {
		return NewValidationError("SPACE_PASSWORD_REPEATED_CHAR", "空间密码过于简单：不能使用同一个字符重复")
	}
	if regexp.MustCompile(`(.)\1{5,}`).MatchString(compact) {
		return NewValidationError("SPACE_PASSWORD_MANY_REPEATS", "空间密码过于简单：不能包含大量连续重复字符")
	}
	if isSequential(lower) {
		return NewValidationError("SPACE_PASSWORD_SEQUENTIAL", "空间密码过于简单：不能使用连续数字或连续字母")
	}
	if hasRepeatedPattern(lower) {
		return NewValidationError("SPACE_PASSWORD_PATTERN", "空间密码过于简单：不能使用重复片段")
	}
	if containsKeyboardSequence(lower) {
		return NewValidationError("SPACE_PASSWORD_KEYBOARD", "空间密码过于简单：不能使用键盘顺序")
	}
	if containsWeakWord(lower) {
		return NewValidationError("SPACE_PASSWORD_WEAK_WORD", "空间密码过于简单：不能使用常见弱密码词")
	}
	if isDateLike(lower) {
		return NewValidationError("SPACE_PASSWORD_DATE", "空间密码过于简单：不能使用明显日期或年份重复")
	}

	categories := 0
	if regexp.MustCompile(`[a-z]`).MatchString(password) {
		categories++
	}
	if regexp.MustCompile(`[A-Z]`).MatchString(password) {
		categories++
	}
	if regexp.MustCompile(`\d`).MatchString(password) {
		categories++
	}
	if regexp.MustCompile(`[^a-zA-Z0-9]`).MatchString(password) {
		categories++
	}
	if categories < 3 {
		return NewValidationError("SPACE_PASSWORD_FEW_CATEGORIES", "空间密码过于简单：建议同时包含大小写字母、数字和符号中的至少三类")
	}

	return nil
}

func DeriveToken(password string) (string, error) {
	if err := ValidatePassword(password); err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(HashPrefix + strings.TrimSpace(password)))
	return hex.EncodeToString(sum[:]), nil
}

func NormalizeToken(value string) (string, error) {
	token := strings.ToLower(strings.TrimSpace(value))
	if !derivedTokenPattern.MatchString(token) {
		return "", NewValidationError("SPACE_TOKEN_INVALID", "个人空间令牌无效，请重新输入空间密码")
	}
	return token, nil
}

func IsValidationError(err error) bool {
	var validationErr ValidationError
	return errors.As(err, &validationErr)
}

func AsValidationError(err error, target *ValidationError) bool {
	return errors.As(err, target)
}

type ValidationError struct {
	Code    string
	Chinese string
}

func NewValidationError(code string, chinese string) ValidationError {
	return ValidationError{Code: code, Chinese: chinese}
}

func (e ValidationError) Error() string {
	return e.Chinese
}

func allSame(value string) bool {
	if value == "" {
		return false
	}
	first := []rune(value)[0]
	for _, item := range value {
		if item != first {
			return false
		}
	}
	return true
}

func isSequential(value string) bool {
	if len(value) < PasswordMinLength {
		return false
	}
	digits := "012345678901234567890"
	reverseDigits := "098765432109876543210"
	letters := "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz"
	reverseLetters := "zyxwvutsrqponmlkjihgfedcbazyxwvutsrqponmlkjihgfedcba"
	return strings.Contains(digits, value) || strings.Contains(reverseDigits, value) || strings.Contains(letters, value) || strings.Contains(reverseLetters, value)
}

func hasRepeatedPattern(value string) bool {
	for size := 1; size <= len(value)/2; size++ {
		if len(value)%size != 0 {
			continue
		}
		part := value[:size]
		if strings.Repeat(part, len(value)/size) == value {
			return true
		}
	}
	return false
}

func containsKeyboardSequence(value string) bool {
	rows := []string{
		"qwertyuiop",
		"poiuytrewq",
		"asdfghjkl",
		"lkjhgfdsa",
		"zxcvbnm",
		"mnbvcxz",
		"1qaz2wsx3edc4rfv5tgb",
		"0okm9ijn8uhb7ygv6tfc",
	}
	for _, row := range rows {
		length := max(6, min(len(row), len(value)))
		if strings.Contains(value, row[:length]) {
			return true
		}
	}
	common := []string{"qwerty", "asdfgh", "zxcvbn", "1qaz2wsx", "qwerty123", "qwertyuiop"}
	for _, item := range common {
		if strings.Contains(value, item) {
			return true
		}
	}
	return false
}

func containsWeakWord(value string) bool {
	normalized := regexp.MustCompile(`[^a-z0-9]`).ReplaceAllString(value, "")
	weakWords := []string{
		"password",
		"admin",
		"administrator",
		"letmein",
		"welcome",
		"iloveyou",
		"qwerty",
		"testtest",
		"aiimage",
		"aigenerate",
		"imagegenerate",
		"cloudtask",
		"myspace",
		"localspace",
	}
	for _, word := range weakWords {
		if strings.Contains(normalized, word) {
			return true
		}
	}
	return false
}

func isDateLike(value string) bool {
	if !regexp.MustCompile(`^\d+$`).MatchString(value) {
		return false
	}
	if regexp.MustCompile(`^(19|20)\d{2}\1`).MatchString(value) {
		return true
	}
	if len(value) >= 4 && regexp.MustCompile(`^(19|20)\d{2}$`).MatchString(value[:4]) && hasRepeatedPattern(value) {
		return true
	}
	return regexp.MustCompile(`^(19|20)\d{6,}$`).MatchString(value)
}

func min(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
