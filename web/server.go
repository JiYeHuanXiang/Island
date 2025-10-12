package web

import (
	"encoding/json"
	"island/config"
	"island/handlers"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
)

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	webClients = make(map[*websocket.Conn]struct{})
	webMutex   sync.RWMutex
	msgHandler *handlers.MessageHandler
)

type CommandRequest struct {
	Command string `json:"command"`
}

type CommandResponse struct {
	Response string `json:"response"`
}

func StartHTTPServer(appConfig *config.Config, handler *handlers.MessageHandler) {
	msgHandler = handler
	http.HandleFunc("/", serveStatic)
	http.HandleFunc("/ws", handleWebSocket)
	http.HandleFunc("/command", handleCommand)
	http.HandleFunc("/api/settings", handleSettings)
	log.Printf("Web服务器已启动 :%s", appConfig.HTTPPort)
	if err := http.ListenAndServe(":"+appConfig.HTTPPort, nil); err != nil {
		log.Fatalf("HTTP服务器错误: %v", err)
	}
}

func serveStatic(w http.ResponseWriter, r *http.Request) {
	path := filepath.Clean(r.URL.Path)
	if path == "/" || path == "/index.html" {
		http.ServeFile(w, r, "web/UI.html")
		return
	}

	if strings.HasPrefix(path, "/") {
		path = filepath.Join("web", path)
	}
	http.ServeFile(w, r, path)
}

func handleCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if msgHandler == nil {
		json.NewEncoder(w).Encode(CommandResponse{Response: "消息处理器未初始化"})
		return
	}

	response := msgHandler.ProcessCommand(req.Command)
	json.NewEncoder(w).Encode(CommandResponse{Response: response})
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket升级失败: %v", err)
		return
	}

	webMutex.Lock()
	webClients[conn] = struct{}{}
	webMutex.Unlock()

	go func() {
		defer func() {
			webMutex.Lock()
			delete(webClients, conn)
			webMutex.Unlock()
			conn.Close()
		}()

		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				if !websocket.IsCloseError(err, websocket.CloseNormalClosure) {
					log.Printf("WebSocket读取错误: %v", err)
				}
				break
			}

			// 解析消息类型
			var msgData map[string]interface{}
			if err := json.Unmarshal(message, &msgData); err != nil {
				// 如果不是JSON格式，当作普通命令处理
				response := msgHandler.ProcessCommand(string(message))
				if err := conn.WriteMessage(websocket.TextMessage, []byte(response)); err != nil {
					log.Printf("WebSocket写入错误: %v", err)
					break
				}

				// 同时发送到QQ
				if err := msgHandler.SendToQQ(response); err != nil {
					log.Printf("发送到QQ失败: %v", err)
				}
				continue
			}

			// 根据消息类型路由到不同的处理函数
			if msgType, ok := msgData["type"].(string); ok {
				switch msgType {
				case "group":
					handleGroupMessage(conn, msgData)
				case "admin":
					handleAdminMessage(conn, msgData)
				default:
					// 未知类型，当作普通命令处理
					if command, ok := msgData["command"].(string); ok {
						response := msgHandler.ProcessCommand(command)
						if err := conn.WriteMessage(websocket.TextMessage, []byte(response)); err != nil {
							log.Printf("WebSocket写入错误: %v", err)
							break
						}
					}
				}
			} else {
				// 没有type字段，当作普通命令处理
				if command, ok := msgData["command"].(string); ok {
					response := msgHandler.ProcessCommand(command)
					if err := conn.WriteMessage(websocket.TextMessage, []byte(response)); err != nil {
						log.Printf("WebSocket写入错误: %v", err)
						break
					}
				}
			}
		}
	}()
}

// 处理群组管理消息
func handleGroupMessage(conn *websocket.Conn, msgData map[string]interface{}) {
	action, ok := msgData["action"].(string)
	if !ok {
		log.Printf("群组消息缺少action字段")
		return
	}

	if msgHandler == nil {
		log.Printf("消息处理器未初始化")
		return
	}

	switch action {
	case "list":
		msgHandler.HandleGetGroupList(conn)
	case "leave":
		msgHandler.HandleLeaveGroup(conn, msgData)
	case "disable":
		msgHandler.HandleDisableGroup(conn, msgData)
	default:
		log.Printf("未知的群组操作: %s", action)
	}
}

// 处理管理员消息
func handleAdminMessage(conn *websocket.Conn, msgData map[string]interface{}) {
	action, ok := msgData["action"].(string)
	if !ok {
		log.Printf("管理员消息缺少action字段")
		return
	}

	switch action {
	case "update":
		handleUpdateAdmin(conn, msgData)
	default:
		log.Printf("未知的管理员操作: %s", action)
	}
}

// 更新管理员信息
func handleUpdateAdmin(conn *websocket.Conn, msgData map[string]interface{}) {
	qq, ok := msgData["qq"].(string)
	if !ok {
		log.Printf("管理员更新缺少QQ号")
		return
	}

	log.Printf("管理员QQ已更新: %s", qq)
	// 这里可以保存管理员QQ到配置文件或数据库
}

// 处理设置同步
func handleSettings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case "GET":
		// 获取当前配置
		if msgHandler == nil {
			http.Error(w, `{"error": "消息处理器未初始化"}`, http.StatusInternalServerError)
			return
		}

		// 从消息处理器获取当前配置
		currentConfig := msgHandler.GetCurrentConfig()
		if currentConfig == nil {
			http.Error(w, `{"error": "无法获取当前配置"}`, http.StatusInternalServerError)
			return
		}

		if err := json.NewEncoder(w).Encode(currentConfig); err != nil {
			log.Printf("序列化配置失败: %v", err)
			http.Error(w, `{"error": "序列化配置失败"}`, http.StatusInternalServerError)
		}

	case "POST":
		// 更新配置
		if msgHandler == nil {
			http.Error(w, `{"error": "消息处理器未初始化"}`, http.StatusInternalServerError)
			return
		}

		var newConfig config.Config
		if err := json.NewDecoder(r.Body).Decode(&newConfig); err != nil {
			http.Error(w, `{"error": "解析配置失败"}`, http.StatusBadRequest)
			return
		}

		// 验证配置
		if err := newConfig.Validate(); err != nil {
			http.Error(w, `{"error": "配置验证失败: `+err.Error()+`"}`, http.StatusBadRequest)
			return
		}

		// 更新配置
		if err := msgHandler.UpdateConfig(&newConfig); err != nil {
			http.Error(w, `{"error": "更新配置失败: `+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}

		// 广播配置更新到所有WebSocket客户端
		BroadcastToWeb(`{"type": "config_updated", "message": "配置已更新"}`)

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "配置更新成功"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// 广播消息到所有Web客户端
func BroadcastToWeb(message string) {
	webMutex.RLock()
	defer webMutex.RUnlock()

	for client := range webClients {
		client := client
		go func() {
			if err := client.WriteMessage(websocket.TextMessage, []byte(message)); err != nil {
				log.Printf("广播消息失败: %v", err)
				webMutex.Lock()
				delete(webClients, client)
				client.Close()
				webMutex.Unlock()
			}
		}()
	}
}
