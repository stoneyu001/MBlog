package tracking

import "strings"

// extractPlatformFromUA 从 User-Agent 中解析操作系统和浏览器
func extractPlatformFromUA(ua string) (platform, browser string) {
	ua = strings.ToLower(ua)

	// 检测操作系统
	switch {
	case strings.Contains(ua, "iphone") || strings.Contains(ua, "ipad") || strings.Contains(ua, "ipod") ||
		(strings.Contains(ua, "macintosh") && strings.Contains(ua, "mobile")):
		platform = "iOS"
	case strings.Contains(ua, "android"):
		platform = "Android"
	case strings.Contains(ua, "windows"):
		platform = "Windows"
	case strings.Contains(ua, "macintosh") || strings.Contains(ua, "mac os x"):
		platform = "macOS"
	case strings.Contains(ua, "linux"):
		platform = "Linux"
	default:
		platform = "Unknown"
	}

	// 检测浏览器（按优先级排序）
	switch {
	case strings.Contains(ua, "edg/") || strings.Contains(ua, "edge/") || strings.Contains(ua, "edgios/"):
		browser = "Edge"
	case strings.Contains(ua, "opera") || strings.Contains(ua, "opr/") || strings.Contains(ua, "opios/"):
		browser = "Opera"
	case strings.Contains(ua, "chrome/") || strings.Contains(ua, "crios/"):
		browser = "Chrome"
	case strings.Contains(ua, "firefox/") || strings.Contains(ua, "fxios/"):
		browser = "Firefox"
	case strings.Contains(ua, "safari/") && !strings.Contains(ua, "chrome/"):
		browser = "Safari"
	default:
		browser = "Unknown"
	}

	return platform, browser
}

// extractDeviceTypeFromUA 从 User-Agent 中解析设备类型
func extractDeviceTypeFromUA(ua string) string {
	ua = strings.ToLower(ua)

	// 检测平板设备（先于手机，因为某些平板UA也包含mobile）
	if strings.Contains(ua, "ipad") ||
		(strings.Contains(ua, "android") && !strings.Contains(ua, "mobile")) ||
		strings.Contains(ua, "tablet") {
		return "Tablet"
	}

	// 检测手机设备
	if strings.Contains(ua, "iphone") ||
		strings.Contains(ua, "ipod") ||
		strings.Contains(ua, "android") && strings.Contains(ua, "mobile") ||
		strings.Contains(ua, "windows phone") ||
		strings.Contains(ua, "blackberry") ||
		strings.Contains(ua, "mobile") {
		return "Mobile"
	}

	// 默认为桌面设备
	return "Desktop"
}

// cleanString 清理字符串，去除首尾空格、换行符，压缩连续空格
func cleanString(s string) string {
	if s == "" {
		return s
	}
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\r\n", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}
	return strings.TrimSpace(s)
}
