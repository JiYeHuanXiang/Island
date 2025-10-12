package connection

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"island/config"
	"github.com/gorilla/websocket"
)

// ConnectionManager 连接管理器
type ConnectionManager struct {
	config     *config.Config
	wsConn     *websocket.Conn
	httpClient *HTTPClient
	reverseWS  *ReverseWebSocket
	mode       string
	mu         sync.RWMutex
	retries    int
	maxRetry   int
	quit       chan struct{}
}

// MessageHandler 消息处理器接口
type MessageHandler interface {
	HandleMessage(data []byte) error
}

var (
	ErrMaxRetries  = errors.New("maximum retry attempts reached")
	ErrConnClosed  = errors.New("connection closed")
	ErrUnsupported = errors.New("unsupported connection mode")
)

// NewConnectionManager 创建连接管理器
func NewConnectionManager(cfg *config.Config, maxRetry int) *ConnectionManager {
	return &ConnectionManager{
		config:   cfg,
		mode:     cfg.ConnectionMode,
		maxRetry: maxRetry,
		quit:     make(chan struct{}),
	}
}

// Connect 建立连接
func (cm *ConnectionManager) Connect() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	switch cm.mode {
	case config.ModeWebSocket:
		return cm.connectWebSocket()
	case config.ModeHTTP:
		return cm.connectHTTP()
	case config.ModeReverseWebSocket:
		return cm.connectReverseWebSocket()
	default:
		return ErrUnsupported
	}
}

// connectWebSocket 连接WebSocket
func (cm *ConnectionManager) connectWebSocket() error {
	if cm.wsConn != nil {
		return nil
	}

	var err error
	for i := 0; i < cm.maxRetry; i++ {
		cm.wsConn, _, err = websocket.DefaultDialer.Dial(cm.config.QQWSURL, nil)
		if err == nil {
			cm.retries = 0
			log.Printf("WebSocket连接成功: %s", cm.config.QQWSURL)
			return nil
		}

		select {
		case <-cm.quit:
			return ErrConnClosed
		default:
			waitTime := time.Duration(i+1) * time.Second
			log.Printf("WebSocket连接失败 (尝试 %d/%d), %v秒后重试...", i+1, cm.maxRetry, waitTime)
			time.Sleep(waitTime)
		}
	}
	return fmt.Errorf("%w: %v", ErrMaxRetries, err)
}

// connectHTTP 连接HTTP API
func (cm *ConnectionManager) connectHTTP() error {
	if cm.httpClient != nil {
		return nil
	}

	cm.httpClient = NewHTTPClient(cm.config)
	log.Printf("HTTP API客户端已初始化: %s", cm.config.QQHTTPURL)
	return nil
}

// connectReverseWebSocket 连接反向WebSocket
func (cm *ConnectionManager) connectReverseWebSocket() error {
	if cm.reverseWS != nil {
		return nil
	}

	cm.reverseWS = NewReverseWebSocket(cm.config)
	log.Printf("反向WebSocket监听器已初始化: %s", cm.config.QQReverseWS)
	return nil
}

// SendMessage 发送消息
func (cm *ConnectionManager) SendMessage(action string, params interface{}) error {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	switch cm.mode {
	case config.ModeWebSocket:
		return cm.sendWebSocketMessage(action, params)
	case config.ModeHTTP:
		return cm.sendHTTPMessage(action, params)
	case config.ModeReverseWebSocket:
		return cm.sendReverseWebSocketMessage(action, params)
	default:
		return ErrUnsupported
	}
}

// sendWebSocketMessage 通过WebSocket发送消息
func (cm *ConnectionManager) sendWebSocketMessage(action string, params interface{}) error {
	if cm.wsConn == nil {
		return ErrConnClosed
	}

	request := map[string]interface{}{
		"action": action,
		"params": params,
	}

	return cm.wsConn.WriteJSON(request)
}

// sendHTTPMessage 通过HTTP API发送消息
func (cm *ConnectionManager) sendHTTPMessage(action string, params interface{}) error {
	if cm.httpClient == nil {
		return ErrConnClosed
	}

	// 根据action类型处理不同的消息发送
	switch action {
	case "send_group_msg":
		if paramsMap, ok := params.(map[string]interface{}); ok {
			groupID, _ := paramsMap["group_id"].(int64)
			message, _ := paramsMap["message"].(string)
			return cm.httpClient.SendMessage("group", groupID, message)
		}
	case "send_private_msg":
		if paramsMap, ok := params.(map[string]interface{}); ok {
			userID, _ := paramsMap["user_id"].(int64)
			message, _ := paramsMap["message"].(string)
			return cm.httpClient.SendMessage("private", userID, message)
		}
	default:
		return fmt.Errorf("不支持的HTTP API操作: %s", action)
	}

	return fmt.Errorf("参数格式错误")
}

// sendReverseWebSocketMessage 通过反向WebSocket发送消息
func (cm *ConnectionManager) sendReverseWebSocketMessage(action string, params interface{}) error {
	if cm.reverseWS == nil {
		return ErrConnClosed
	}

	// 反向WebSocket通常由客户端主动连接，这里需要等待连接
	if !cm.reverseWS.IsConnected() {
		return errors.New("反向WebSocket客户端未连接")
	}

	request := map[string]interface{}{
		"action": action,
		"params": params,
	}

	return cm.reverseWS.SendMessage(request)
}

// ReceiveMessage 接收消息
func (cm *ConnectionManager) ReceiveMessage() ([]byte, error) {
	switch cm.mode {
	case config.ModeWebSocket:
		return cm.receiveWebSocketMessage()
	case config.ModeReverseWebSocket:
		return cm.receiveReverseWebSocketMessage()
	default:
		return nil, ErrUnsupported
	}
}

// receiveWebSocketMessage 接收WebSocket消息
func (cm *ConnectionManager) receiveWebSocketMessage() ([]byte, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if cm.wsConn == nil {
		return nil, ErrConnClosed
	}

	_, message, err := cm.wsConn.ReadMessage()
	if err != nil {
		cm.mu.Lock()
		cm.wsConn = nil
		cm.mu.Unlock()
		return nil, err
	}

	return message, nil
}

// receiveReverseWebSocketMessage 接收反向WebSocket消息
func (cm *ConnectionManager) receiveReverseWebSocketMessage() ([]byte, error) {
	if cm.reverseWS == nil {
		return nil, ErrConnClosed
	}

	return cm.reverseWS.ReceiveMessage()
}

// GetConnectionMode 获取当前连接模式
func (cm *ConnectionManager) GetConnectionMode() string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.mode
}

// SwitchMode 切换连接模式
func (cm *ConnectionManager) SwitchMode(mode string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if mode == cm.mode {
		return nil
	}

	// 关闭当前连接
	cm.Close()

	// 更新模式
	cm.mode = mode
	cm.config.ConnectionMode = mode

	// 建立新连接
	return cm.Connect()
}

// IsConnected 检查连接状态
func (cm *ConnectionManager) IsConnected() bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	switch cm.mode {
	case config.ModeWebSocket:
		return cm.wsConn != nil
	case config.ModeHTTP:
		return cm.httpClient != nil
	case config.ModeReverseWebSocket:
		return cm.reverseWS != nil && cm.reverseWS.IsConnected()
	default:
		return false
	}
}

// GetGroupList 获取群组列表
func (cm *ConnectionManager) GetGroupList() ([]GroupInfo, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	switch cm.mode {
	case config.ModeHTTP:
		if cm.httpClient != nil {
			return cm.httpClient.GetGroupList()
		}
	case config.ModeWebSocket:
		// WebSocket模式下需要发送请求获取群组列表
		return cm.getWebSocketGroupList()
	case config.ModeReverseWebSocket:
		// 反向WebSocket模式下需要发送请求获取群组列表
		return cm.getReverseWebSocketGroupList()
	default:
		return nil, ErrUnsupported
	}

	return nil, ErrConnClosed
}

// getWebSocketGroupList 通过WebSocket获取群组列表
func (cm *ConnectionManager) getWebSocketGroupList() ([]GroupInfo, error) {
	if cm.wsConn == nil {
		return nil, ErrConnClosed
	}

	request := map[string]interface{}{
		"action": "get_group_list",
		"params": map[string]interface{}{},
	}

	if err := cm.wsConn.WriteJSON(request); err != nil {
		return nil, fmt.Errorf("发送群组列表请求失败: %w", err)
	}

	// 等待响应
	_, message, err := cm.wsConn.ReadMessage()
	if err != nil {
		return nil, fmt.Errorf("接收群组列表响应失败: %w", err)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(message, &response); err != nil {
		return nil, fmt.Errorf("解析群组列表响应失败: %w", err)
	}

	return cm.processGroupListResponse(response)
}

// getReverseWebSocketGroupList 通过反向WebSocket获取群组列表
func (cm *ConnectionManager) getReverseWebSocketGroupList() ([]GroupInfo, error) {
	if cm.reverseWS == nil || !cm.reverseWS.IsConnected() {
		return nil, ErrConnClosed
	}

	request := map[string]interface{}{
		"action": "get_group_list",
		"params": map[string]interface{}{},
	}

	if err := cm.reverseWS.SendMessage(request); err != nil {
		return nil, fmt.Errorf("发送群组列表请求失败: %w", err)
	}

	// 等待响应
	message, err := cm.reverseWS.ReceiveMessage()
	if err != nil {
		return nil, fmt.Errorf("接收群组列表响应失败: %w", err)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(message, &response); err != nil {
		return nil, fmt.Errorf("解析群组列表响应失败: %w", err)
	}

	return cm.processGroupListResponse(response)
}

// processGroupListResponse 处理群组列表响应
func (cm *ConnectionManager) processGroupListResponse(response map[string]interface{}) ([]GroupInfo, error) {
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
		return groups, nil
	}

	return nil, fmt.Errorf("无法解析群组列表响应: %v", response)
}

// LeaveGroup 退出群组
func (cm *ConnectionManager) LeaveGroup(groupID int64) error {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	switch cm.mode {
	case config.ModeHTTP:
		if cm.httpClient != nil {
			return cm.httpClient.LeaveGroup(groupID)
		}
	case config.ModeWebSocket:
		return cm.leaveWebSocketGroup(groupID)
	case config.ModeReverseWebSocket:
		return cm.leaveReverseWebSocketGroup(groupID)
	default:
		return ErrUnsupported
	}

	return ErrConnClosed
}

// leaveWebSocketGroup 通过WebSocket退出群组
func (cm *ConnectionManager) leaveWebSocketGroup(groupID int64) error {
	if cm.wsConn == nil {
		return ErrConnClosed
	}

	request := map[string]interface{}{
		"action": "set_group_leave",
		"params": map[string]interface{}{
			"group_id": groupID,
		},
	}

	return cm.wsConn.WriteJSON(request)
}

// leaveReverseWebSocketGroup 通过反向WebSocket退出群组
func (cm *ConnectionManager) leaveReverseWebSocketGroup(groupID int64) error {
	if cm.reverseWS == nil || !cm.reverseWS.IsConnected() {
		return ErrConnClosed
	}

	request := map[string]interface{}{
		"action": "set_group_leave",
		"params": map[string]interface{}{
			"group_id": groupID,
		},
	}

	return cm.reverseWS.SendMessage(request)
}

// Reinitialize 重新初始化连接管理器
func (cm *ConnectionManager) Reinitialize(newConfig *config.Config) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// 关闭当前连接
	close(cm.quit)
	
	if cm.wsConn != nil {
		cm.wsConn.Close()
		cm.wsConn = nil
	}

	if cm.httpClient != nil {
		cm.httpClient = nil
	}

	if cm.reverseWS != nil {
		cm.reverseWS.Close()
		cm.reverseWS = nil
	}

	// 更新配置
	cm.config = newConfig
	cm.mode = newConfig.ConnectionMode
	cm.retries = 0
	cm.quit = make(chan struct{})

	// 重新建立连接
	return cm.Connect()
}

// Close 关闭连接
func (cm *ConnectionManager) Close() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	close(cm.quit)

	if cm.wsConn != nil {
		cm.wsConn.Close()
		cm.wsConn = nil
	}

	if cm.httpClient != nil {
		cm.httpClient = nil
	}

	if cm.reverseWS != nil {
		cm.reverseWS.Close()
		cm.reverseWS = nil
	}
}
