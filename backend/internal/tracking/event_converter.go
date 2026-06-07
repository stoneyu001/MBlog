package tracking

import (
	"bytes"
	"encoding/json"
	"log"
	"math"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// convertToUnpartitionedTrackEvent 将前端请求转换为埋点事件对象
func convertToUnpartitionedTrackEvent(req UnpartitionedTrackEventRequest, c *gin.Context) *UnpartitionedTrackEvent {
	// 处理时间戳
	var eventTime time.Time
	if req.Timestamp > 0 {
		clientTime := time.UnixMilli(req.Timestamp).In(chinaLocation)
		minTime := time.Now().Add(-24 * time.Hour)
		if clientTime.After(minTime) {
			eventTime = clientTime
		} else {
			eventTime = time.Now().In(chinaLocation)
		}
	} else {
		eventTime = time.Now().In(chinaLocation)
	}

	if req.EventType == "" {
		req.EventType = "UNKNOWN"
	}

	// 智能解析 platform：处理前端发送的 "OS/Browser" 格式
	var extractedBrowser string
	validPlatforms := map[string]bool{
		"Windows": true, "macOS": true, "Linux": true,
		"Android": true, "iOS": true, "Unknown": true,
	}

	if strings.Contains(req.Platform, "/") {
		parts := strings.Split(req.Platform, "/")
		if len(parts) >= 2 {
			req.Platform = parts[0]
			extractedBrowser = parts[1]
		}
	}

	if req.Platform == "" || req.Platform == "unknown" || !validPlatforms[req.Platform] {
		userAgent := c.Request.UserAgent()
		platform, browser := extractPlatformFromUA(userAgent)
		req.Platform = platform
		extractedBrowser = browser
	}

	if extractedBrowser != "" {
		if req.Metadata == nil {
			req.Metadata = make(map[string]interface{})
		}
		if _, exists := req.Metadata["browser"]; !exists {
			req.Metadata["browser"] = extractedBrowser
		}
		req.Metadata["user_agent"] = c.Request.UserAgent()
	}

	// 计算 event_duration：两次事件间的时间差
	if req.EventDuration <= 0 {
		lastEvent, exists := getLastEventInfo(req.SessionID)
		if exists && req.Timestamp > lastEvent.Timestamp {
			durationMs := req.Timestamp - lastEvent.Timestamp
			req.EventDuration = int(math.Min(float64(durationMs)/1000.0, 3600.0))
		}
	}

	updateLastEventInfo(req.SessionID, LastEventInfo{
		Timestamp: req.Timestamp,
		EventType: req.EventType,
		SessionID: req.SessionID,
	})

	// URL 字段解码
	pagePath := decodeURLField(req.PagePath, "/")
	elementPath := decodeURLField(req.ElementPath, "")
	referrer := decodeURLField(req.Referrer, "")
	customProps := decodeMapURLs(req.CustomProperties)
	metadataMap := decodeMapURLs(req.Metadata)

	// 清理 device_info 中的敏感字段
	deviceInfoMap := sanitizeDeviceInfo(req.DeviceInfo)

	// 清理所有字符串字段
	req.SessionID = cleanString(req.SessionID)
	req.UserID = cleanString(req.UserID)
	req.EventType = cleanString(req.EventType)
	pagePath = cleanString(pagePath)
	elementPath = cleanString(elementPath)
	referrer = cleanString(referrer)
	req.Platform = cleanString(req.Platform)

	// JSON 序列化
	metadata := convertMapToString(metadataMap)
	customPropsStr := convertMapToString(customProps)
	deviceInfo := convertMapToString(deviceInfoMap)

	return &UnpartitionedTrackEvent{
		SessionID:        req.SessionID,
		UserID:           req.UserID,
		EventType:        req.EventType,
		ElementPath:      elementPath,
		PagePath:         pagePath,
		Referrer:         referrer,
		Metadata:         metadata,
		UserAgent:        c.Request.UserAgent(),
		IPAddress:        c.ClientIP(),
		CreatedAt:        eventTime,
		CustomProperties: customPropsStr,
		Platform:         req.Platform,
		DeviceInfo:       deviceInfo,
		EventDuration:    req.EventDuration,
		DeviceType:       extractDeviceTypeFromUA(c.Request.UserAgent()),
	}
}

// convertMapToString 将 map 转为 JSON 字符串，确保中文不被转义
func convertMapToString(data map[string]interface{}) string {
	if len(data) == 0 {
		return `{}`
	}
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(data); err != nil {
		log.Printf("JSON序列化失败: %v", err)
		return `{}`
	}
	return strings.TrimSpace(buffer.String())
}

// decodeURLField 对单个 URL 字段进行解码
func decodeURLField(raw string, defaultVal string) string {
	if raw == "" {
		return defaultVal
	}
	decoded, err := url.QueryUnescape(raw)
	if err != nil {
		return raw
	}
	return decoded
}

// decodeMapURLs 解码 map 中 value 为 string 且 key 含 url/path/link 的字段
func decodeMapURLs(m map[string]interface{}) map[string]interface{} {
	for key, value := range m {
		if strValue, ok := value.(string); ok {
			lowerKey := strings.ToLower(key)
			if strings.Contains(lowerKey, "url") ||
				strings.Contains(lowerKey, "path") ||
				strings.Contains(lowerKey, "link") ||
				strings.Contains(lowerKey, "href") {
				if decoded, err := url.QueryUnescape(strValue); err == nil {
					m[key] = decoded
				}
			}
		}
	}
	return m
}

// sanitizeDeviceInfo 清理设备信息中的敏感字段
func sanitizeDeviceInfo(info map[string]interface{}) map[string]interface{} {
	if info == nil {
		return nil
	}
	sensitiveKeys := []string{"password", "token", "secret", "key", "auth"}
	for _, k := range sensitiveKeys {
		delete(info, k)
	}
	for key, value := range info {
		if strValue, ok := value.(string); ok {
			info[key] = cleanString(strValue)
		}
	}
	return info
}
