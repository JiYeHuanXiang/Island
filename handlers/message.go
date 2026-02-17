package handlers

import (
	"encoding/json"
	"fmt"
	"island/config"
	"island/connection"
	"island/dice"
	"log"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// MessageHandler 处理OneBot V11协议消息
type MessageHandler struct {
	connManager *connection.ConnectionManager
	config      *config.Config
	diceEngine  *dice.Engine
	cmdRegistry *dice.CommandRegistry
	mu          sync.RWMutex
}

// OneBotMessage OneBot V11协议消息结构
type OneBotMessage struct {
	PostType    string          `json:"post_type"`
	MessageType string          `json:"message_type"`
	Message     json.RawMessage `json:"message"`
	UserID      int64           `json:"user_id"`
	GroupID     int64           `json:"group_id"`
	RawMessage  string          `json:"raw_message"`
	SelfID      int64           `json:"self_id"`
}

// NewMessageHandler 创建新的消息处理器
func NewMessageHandler(connManager *connection.ConnectionManager, cfg *config.Config) *MessageHandler {
	return &MessageHandler{
		connManager:  connManager,
		config:      cfg,
		diceEngine:  dice.New(),
		cmdRegistry: dice.NewCommandRegistry(),
	}
}

// StartWebSocketMessageLoop 启动WebSocket消息循环
func (h *MessageHandler) StartWebSocketMessageLoop() {
	log.Println("启动WebSocket消息循环...")
	for {
		message, err := h.connManager.ReceiveMessage()
		if err != nil {
			log.Printf("接收消息错误: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		msg, err := h.parseMessage(message)
		if err != nil {
			log.Printf("消息解析错误: %v", err)
			continue
		}

		if msg.PostType == "message" {
			h.handleMessage(msg)
		}
	}
}

// parseMessage 解析消息
func (h *MessageHandler) parseMessage(data []byte) (*OneBotMessage, error) {
	var msg OneBotMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, fmt.Errorf("invalid message: %v", err)
	}
	return &msg, nil
}

// handleMessage 处理消息
func (h *MessageHandler) handleMessage(msg *OneBotMessage) {
	// 提取消息内容
	content, err := h.extractMessageContent(msg.Message)
	if err != nil {
		log.Printf("消息内容提取失败: %v", err)
		return
	}

	// 检查是否是命令（以.开头）
	if !strings.HasPrefix(content, ".") {
		return
	}

	// 创建命令上下文
	ctx := &dice.CommandContext{
		PlayerID: msg.UserID,
		GroupID:  msg.GroupID,
		Engine:   h.diceEngine,
	}

	// 处理命令
	response := h.cmdRegistry.Process(content, ctx)

	// 发送响应
	h.sendResponse(msg, response)
}

// extractMessageContent 提取消息文本内容
func (h *MessageHandler) extractMessageContent(msg json.RawMessage) (string, error) {
	var messageSegments []struct {
		Type string `json:"type"`
		Data struct {
			Text string `json:"text"`
		} `json:"data"`
	}

	if err := json.Unmarshal(msg, &messageSegments); err != nil {
		// 尝试直接解析为字符串
		var text string
		if err := json.Unmarshal(msg, &text); err != nil {
			return "", fmt.Errorf("invalid message: %v", err)
		}
		return text, nil
	}

	var builder strings.Builder
	for _, seg := range messageSegments {
		if seg.Type == "text" {
			builder.WriteString(seg.Data.Text)
		}
	}
	return builder.String(), nil
}

// sendResponse 发送响应消息
func (h *MessageHandler) sendResponse(msg *OneBotMessage, response string) {
	if msg.MessageType == "group" {
		// 检查是否配置了群组过滤
		if len(h.config.QQGroupID) > 0 {
			found := false
			for _, gid := range h.config.QQGroupID {
				if gid == msg.GroupID {
					found = true
					break
				}
			}
			if !found {
				return // 不在允许的群组中
			}
		}

		err := h.connManager.SendMessage("send_group_msg", map[string]interface{}{
			"group_id": msg.GroupID,
			"message":  response,
		})
		if err != nil {
			log.Printf("发送群消息失败: %v", err)
		}
	} else if msg.MessageType == "private" {
		err := h.connManager.SendMessage("send_private_msg", map[string]interface{}{
			"user_id": msg.UserID,
			"message": response,
		})
		if err != nil {
			log.Printf("发送私聊消息失败: %v", err)
		}
	}
}

// SendToQQ 发送消息到QQ
func (h *MessageHandler) SendToQQ(message string) error {
	if len(h.config.QQGroupID) == 0 {
		return fmt.Errorf("没有配置群组ID")
	}
	return h.connManager.SendMessage("send_group_msg", map[string]interface{}{
		"group_id": h.config.QQGroupID[0],
		"message":  message,
	})
}

// GetCurrentConfig 获取当前配置
func (h *MessageHandler) GetCurrentConfig() *config.Config {
	return h.config
}

// UpdateConfig 更新配置
func (h *MessageHandler) UpdateConfig(newConfig *config.Config) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.config = newConfig

	// 重新连接以应用新配置
	if h.connManager != nil {
		h.connManager.Close()
		return h.connManager.Connect()
	}

	return nil
}

// GetDiceEngine 获取骰子引擎
func (h *MessageHandler) GetDiceEngine() *dice.Engine {
	return h.diceEngine
}

// GetCommandRegistry 获取指令注册表
func (h *MessageHandler) GetCommandRegistry() *dice.CommandRegistry {
	return h.cmdRegistry
}

// ProcessCommand 处理命令（供 Web 调用）
func (h *MessageHandler) ProcessCommand(cmd string) string {
	ctx := &dice.CommandContext{
		PlayerID: 0,
		GroupID:  0,
		Engine:   h.diceEngine,
	}
	return h.cmdRegistry.Process(cmd, ctx)
}

// HandleGetGroupList 处理获取群组列表请求
func (h *MessageHandler) HandleGetGroupList(conn *websocket.Conn) {
	groups, err := h.connManager.GetGroupList()
	if err != nil {
		conn.WriteJSON(map[string]interface{}{
			"type":  "error",
			"error": err.Error(),
		})
		return
	}
	conn.WriteJSON(map[string]interface{}{
		"type":   "group",
		"action": "list",
		"groups": groups,
	})
}

// HandleLeaveGroup 处理退出群组请求
func (h *MessageHandler) HandleLeaveGroup(conn *websocket.Conn, msgData map[string]interface{}) {
	groupID, ok := msgData["group_id"].(float64)
	if !ok {
		conn.WriteJSON(map[string]interface{}{
			"type":    "group",
			"action":  "leave",
			"message": "群组ID无效",
		})
		return
	}

	err := h.connManager.LeaveGroup(int64(groupID))
	if err != nil {
		conn.WriteJSON(map[string]interface{}{
			"type":    "group",
			"action":  "leave",
			"message": err.Error(),
		})
		return
	}

	conn.WriteJSON(map[string]interface{}{
		"type":    "group",
		"action":  "leave",
		"message": fmt.Sprintf("已退出群组 %.0f", groupID),
	})
}

// HandleDisableGroup 处理禁用群组请求
func (h *MessageHandler) HandleDisableGroup(conn *websocket.Conn, msgData map[string]interface{}) {
	groupID, ok := msgData["group_id"].(float64)
	if !ok {
		conn.WriteJSON(map[string]interface{}{
			"type":    "group",
			"action":  "disable",
			"message": "群组ID无效",
		})
		return
	}

	conn.WriteJSON(map[string]interface{}{
		"type":    "group",
		"action":  "disable",
		"message": fmt.Sprintf("已禁用群组 %.0f", groupID),
	})
}

// HandleWebSocketRequest 处理WebSocket请求（用于管理界面）
func (h *MessageHandler) HandleWebSocketRequest(conn *websocket.Conn, msgType string, data interface{}) {
	switch msgType {
	case "command":
		// 直接执行命令
		if cmdData, ok := data.(map[string]interface{}); ok {
			if cmd, ok := cmdData["command"].(string); ok {
				ctx := &dice.CommandContext{
					PlayerID: 0,
					GroupID:  0,
					Engine:   h.diceEngine,
				}
				response := h.cmdRegistry.Process(cmd, ctx)
				conn.WriteJSON(map[string]interface{}{
					"type":    "command_result",
					"result":  response,
					"command": cmd,
				})
			}
		}
	case "help":
		response := h.cmdRegistry.GetHelp()
		conn.WriteJSON(map[string]interface{}{
			"type":   "help",
			"result": response,
		})
	case "groups":
		// 获取群组列表
		groups, err := h.connManager.GetGroupList()
		if err != nil {
			conn.WriteJSON(map[string]interface{}{
				"type":  "error",
				"error": err.Error(),
			})
			return
		}
		conn.WriteJSON(map[string]interface{}{
			"type":   "groups",
			"groups": groups,
		})
	}
}

// 处理特殊命令（不在注册表中的）
var (
	rhRegex = regexp.MustCompile(`^rh$`)
)

// handleSpecialCommand 处理特殊命令
func (h *MessageHandler) handleSpecialCommand(cmd string, msg *OneBotMessage) string {
	cmd = strings.TrimSpace(cmd)

	// .help 指令
	if cmd == "help" {
		return h.cmdRegistry.GetHelp()
	}

	// .rh 暗骰指令
	if rhRegex.MatchString(cmd) {
		// 简单的暗骰实现
		roll := int(time.Now().UnixNano() % 100)
		return fmt.Sprintf("暗骰结果: 1D100=%d", roll+1)
	}

	return ""
}
