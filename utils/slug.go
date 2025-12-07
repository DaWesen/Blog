package utils

import (
	"regexp"
	"strings"
)

// GenerateSlug 从中文/英文生成URL友好的slug
func GenerateSlug(input string) string {
	// 1. 去除首尾空格
	trimmed := strings.TrimSpace(input)

	// 2. 转换为小写
	lower := strings.ToLower(trimmed)

	// 3. 替换空格为连字符
	withHyphens := strings.ReplaceAll(lower, " ", "-")

	// 4. 移除特殊字符，只保留字母、数字、连字符
	reg := regexp.MustCompile("[^a-z0-9-]+")
	cleaned := reg.ReplaceAllString(withHyphens, "")

	// 5. 移除连续的连字符
	reg = regexp.MustCompile("-+")
	final := reg.ReplaceAllString(cleaned, "-")

	// 6. 移除首尾的连字符
	final = strings.Trim(final, "-")

	return final
}

// SanitizeSlug 清理用户输入的slug
func SanitizeSlug(input string) string {
	return GenerateSlug(input)
}
