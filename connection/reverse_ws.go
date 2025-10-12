package connection

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"island/config"

	"github.com/gorilla/websocket"
)

// ReverseWebSocket 实现反向WebSocket连接
type ReverseWebSocket struct {
	config     *config.Config
	conn       *websocket.Conn
	mu         sync.RWMutex
	quit       chan struct{}
	messageCh  chan []byte
	connected  bool
}

// NewReverseWebSocket 创建新的反向WebSocket连接
func NewReverseWebSocket(cfg *config.Config) *ReverseWebSocket {
	return &ReverseWebSocket{
		config:    cfg,
		quit:      make(chan struct{}),
		messageCh: make(chan []byte, 100),
		connected: false,
	}
}

// Start 启动反向WebSocket服务器
func (r *ReverseWebSocket) Start() error {
	if r.config.QQReverseWS == "" {
		return fmt.Errorf("未配置反向WebSocket地址")
	}

	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	http.HandleFunc(r.config.QQReverseWS, func(w http.ResponseWriter, req *http.Request) {
		conn, err := upgrader.Upgrade(w, req, nil)
		if err != nil {
			log.Printf("反向WebSocket升级失败: %v", err)
			return
		}

		r.mu.Lock()
		r.conn = conn
		r.connected = true
		r.mu.Unlock()

		log.Println("反向WebSocket连接已建立")

		// 处理消息
		go r.handleMessages(conn)

		// 等待连接关闭
		<-r.quit
	})

	log.Printf("反向WebSocket服务器已启动，路径: %s", r.config.QQReverseWS)
	return nil
}

// handleMessages 处理接收到的消息
func (r *ReverseWebSocket) handleMessages(conn *websocket.Conn) {
	defer func() {
		r.mu.Lock()
		r.connected = false
		r.conn = nil
		r.mu.Unlock()
		conn.Close()
	}()

	for {
		select {
		case <-r.quit:
			return
		default:
			_, message, err := conn.ReadMessage()
			if err != nil {
				if !websocket.IsCloseError(err, websocket.CloseNormalClosure) {
					log.Printf("反向WebSocket读取错误: %v", err)
				}
				return
			}

			// 将消息发送到消息通道
			select {
			case r.messageCh <- message:
			default:
				log.Println("反向WebSocket消息通道已满，丢弃消息")
			}
		}
	}
}

// GetMessage 获取消息
func (r *ReverseWebSocket) GetMessage() ([]byte, error) {
	select {
	case message := <-r.messageCh:
		return message, nil
	case <-time.After(1 * time.Second):
		return nil, fmt.Errorf("获取消息超时")
	case <-r.quit:
		return nil, fmt.Errorf("连接已关闭")
	}
}

// ReceiveMessage 接收消息（兼容manager.go接口）
func (r *ReverseWebSocket) ReceiveMessage() ([]byte, error) {
	return r.GetMessage()
}

// SendMessage 发送消息
func (r *ReverseWebSocket) SendMessage(request map[string]interface{}) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if !r.connected || r.conn == nil {
		return fmt.Errorf("反向WebSocket未连接")
	}

	return r.conn.WriteJSON(request)
}

// GetGroupList 获取群组列表
func (r *ReverseWebSocket) GetGroupList() ([]GroupInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if !r.connected || r.conn == nil {
		return nil, fmt.Errorf("反向WebSocket未连接")
	}

	// 发送获取群组列表请求
	request := map[string]interface{}{
		"action": "get_group_list",
		"params": map[string]interface{}{},
		"echo":   "group_list_request",
	}

	if err := r.conn.WriteJSON(request); err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}

	// 等待响应
	timeout := time.After(10 * time.Second)
	for {
		select {
		case <-timeout:
			return nil, fmt.Errorf("获取群组列表超时")
		case message := <-r.messageCh:
			var response map[string]interface{}
			if err := json.Unmarshal(message, &response); err != nil {
				continue
			}

			// 检查是否是群组列表响应
			if echo, ok := response["echo"].(string); ok && echo == "group_list_request" {
				return r.processGroupListResponse(response)
			}
		case <-r.quit:
			return nil, fmt.Errorf("连接已关闭")
		}
	}
}

// processGroupListResponse 处理群组列表响应
func (r *ReverseWebSocket) processGroupListResponse(response map[string]interface{}) ([]GroupInfo, error) {
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
func (r *ReverseWebSocket) LeaveGroup(groupID int64) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if !r.connected || r.conn == nil {
		return fmt.Errorf("反向WebSocket未连接")
	}

	request := map[string]interface{}{
		"action": "set_group_leave",
		"params": map[string]interface{}{
			"group_id": groupID,
		},
	}

	return r.conn.WriteJSON(request)
}

// IsConnected 检查是否连接
func (r *ReverseWebSocket) IsConnected() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.connected
}

// Close 关闭连接
func (r *ReverseWebSocket) Close() {
	close(r.quit)
	r.mu.Lock()
	if r.conn != nil {
		r.conn.Close()
		r.conn = nil
	}
	r.connected = false
	r.mu.Unlock()
}
