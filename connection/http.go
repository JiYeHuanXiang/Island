package connection

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"island/config"
)

// HTTPClient 实现HTTP API客户端
type HTTPClient struct {
	config     *config.Config
	httpClient *http.Client
}

// NewHTTPClient 创建新的HTTP客户端
func NewHTTPClient(cfg *config.Config) *HTTPClient {
	return &HTTPClient{
		config: cfg,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// SendMessage 发送消息到QQ
func (h *HTTPClient) SendMessage(messageType string, targetID int64, message string) error {
	apiURL := h.config.QQHTTPURL
	if !strings.HasSuffix(apiURL, "/") {
		apiURL += "/"
	}

	var action string
	var params map[string]interface{}

	switch messageType {
	case "group":
		action = "send_group_msg"
		params = map[string]interface{}{
			"group_id": targetID,
			"message":  message,
		}
	case "private":
		action = "send_private_msg"
		params = map[string]interface{}{
			"user_id": targetID,
			"message": message,
		}
	default:
		action = "send_msg"
		params = map[string]interface{}{
			"message_type": messageType,
			"user_id":      targetID,
			"group_id":     targetID,
			"message":      message,
		}
	}

	requestBody := map[string]interface{}{
		"action": action,
		"params": params,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("构建请求失败: %w", err)
	}

	resp, err := h.httpClient.Post(apiURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP API错误: 状态码 %d, 响应: %s", resp.StatusCode, string(body))
	}

	// 解析响应检查是否成功
	var response map[string]interface{}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %w", err)
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("解析响应失败: %w", err)
	}

	if status, ok := response["status"].(string); ok && status != "ok" {
		return fmt.Errorf("API返回错误: %v", response)
	}

	return nil
}

// GetGroupList 获取群组列表
func (h *HTTPClient) GetGroupList() ([]GroupInfo, error) {
	apiURL := h.config.QQHTTPURL
	if !strings.HasSuffix(apiURL, "/") {
		apiURL += "/"
	}
	apiURL += "get_group_list"

	requestBody := map[string]interface{}{
		"action": "get_group_list",
		"params": map[string]interface{}{},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("构建请求失败: %w", err)
	}

	log.Printf("发送HTTP API请求到: %s", apiURL)
	resp, err := h.httpClient.Post(apiURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP API错误: 状态码 %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	log.Printf("收到HTTP群组列表响应: %+v", response)

	// 处理响应
	return h.processGroupListResponse(response)
}

// processGroupListResponse 处理群组列表响应
func (h *HTTPClient) processGroupListResponse(response map[string]interface{}) ([]GroupInfo, error) {
	// 检查是否是API响应格式
	if status, ok := response["status"].(string); ok && status == "ok" {
		if data, ok := response["data"].([]interface{}); ok {
			groups := make([]GroupInfo, 0)
			for _, item := range data {
				if groupData, ok := item.(map[string]interface{}); ok {
					groupID, _ := groupData["group_id"].(float64)
					groupName, _ := groupData["group_name"].(string)
					
					groups = append(groups, GroupInfo{
						ID:     int64(groupID),
						Name:   groupName,
						Active: true,
					})
				}
			}
			log.Printf("成功获取到 %d 个群组", len(groups))
			return groups, nil
		}
	}

	// 检查是否是直接数据格式
	if data, ok := response["data"].([]interface{}); ok {
		groups := make([]GroupInfo, 0)
		for _, item := range data {
			if groupData, ok := item.(map[string]interface{}); ok {
				groupID, _ := groupData["group_id"].(float64)
				groupName, _ := groupData["group_name"].(string)
				
				groups = append(groups, GroupInfo{
					ID:     int64(groupID),
					Name:   groupName,
					Active: true,
				})
			}
		}
		log.Printf("成功获取到 %d 个群组", len(groups))
		return groups, nil
	}

	return nil, fmt.Errorf("无法解析群组列表响应: %v", response)
}

// LeaveGroup 退出群组
func (h *HTTPClient) LeaveGroup(groupID int64) error {
	apiURL := h.config.QQHTTPURL
	if !strings.HasSuffix(apiURL, "/") {
		apiURL += "/"
	}
	apiURL += "set_group_leave"

	requestBody := map[string]interface{}{
		"action": "set_group_leave",
		"params": map[string]interface{}{
			"group_id": groupID,
		},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("构建请求失败: %w", err)
	}

	resp, err := h.httpClient.Post(apiURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP API错误: 状态码 %d", resp.StatusCode)
	}

	// 解析响应检查是否成功
	var response map[string]interface{}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %w", err)
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("解析响应失败: %w", err)
	}

	if status, ok := response["status"].(string); ok && status != "ok" {
		return fmt.Errorf("API返回错误: %v", response)
	}

	return nil
}

// TestConnection 测试连接
func (h *HTTPClient) TestConnection() error {
	apiURL := h.config.QQHTTPURL
	if !strings.HasSuffix(apiURL, "/") {
		apiURL += "/"
	}
	apiURL += "get_status"

	requestBody := map[string]interface{}{
		"action": "get_status",
		"params": map[string]interface{}{},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("构建请求失败: %w", err)
	}

	resp, err := h.httpClient.Post(apiURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP API错误: 状态码 %d", resp.StatusCode)
	}

	return nil
}
