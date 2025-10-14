package web

import (
	"encoding/json"
	"fmt"
	"island/config"
	"island/handlers"
	"log"
	"net/http"
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

	// 自定义处理函数，处理根路径并提供index.html
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// 如果是根路径，提供index.html
		if r.URL.Path == "/" || r.URL.Path == "/index.html" {
			http.ServeFile(w, r, "web/index.html")
			return
		}

		// 如果是静态资源路径，提供对应的文件
		if r.URL.Path == "/css/styles.css" {
			http.ServeFile(w, r, "web/css/styles.css")
			return
		}

		if r.URL.Path == "/js/utils.js" {
			http.ServeFile(w, r, "web/js/utils.js")
			return
		}

		if r.URL.Path == "/js/settings.js" {
			http.ServeFile(w, r, "web/js/settings.js")
			return
		}

		if r.URL.Path == "/js/app.js" {
			http.ServeFile(w, r, "web/js/app.js")
			return
		}

		// 其他路径返回404
		http.NotFound(w, r)
	})

	http.HandleFunc("/ws", handleWebSocket)
	http.HandleFunc("/command", handleCommand)
	http.HandleFunc("/api/settings", handleSettings)
	log.Printf("Web服务器已启动 :%s", appConfig.HTTPPort)
	if err := http.ListenAndServe(":"+appConfig.HTTPPort, nil); err != nil {
		log.Fatalf("HTTP服务器错误: %v", err)
	}
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
				command := string(message)
				log.Printf("收到WebSocket命令: %s", command)
				response := msgHandler.ProcessCommand(command)

				// 发送响应回WebSocket客户端
				if err := conn.WriteMessage(websocket.TextMessage, []byte(response)); err != nil {
					log.Printf("WebSocket写入错误: %v", err)
					break
				}

				// 如果配置了QQ连接，也发送到QQ
				currentConfig := msgHandler.GetCurrentConfig()
				if currentConfig != nil && len(currentConfig.QQGroupID) > 0 {
					if err := msgHandler.SendToQQ(response); err != nil {
						log.Printf("发送到QQ失败: %v", err)
					}
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

		// 使用map来解析前端发送的数据（字段名可能大小写不一致）
		var settingsData map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&settingsData); err != nil {
			http.Error(w, `{"error": "解析配置失败"}`, http.StatusBadRequest)
			return
		}

		// 转换为Config结构体
		newConfig := config.Config{}

		// 处理HTTP端口
		if httpPort, ok := settingsData["httpPort"].(float64); ok {
			newConfig.HTTPPort = fmt.Sprintf("%.0f", httpPort)
		} else if httpPort, ok := settingsData["HTTPPort"].(string); ok {
			newConfig.HTTPPort = httpPort
		}

		// 处理连接模式
		if connectionMode, ok := settingsData["connectionMode"].(string); ok {
			newConfig.ConnectionMode = connectionMode
		} else if connectionMode, ok := settingsData["ConnectionMode"].(string); ok {
			newConfig.ConnectionMode = connectionMode
		}

		// 处理WebSocket URL
		if qqWSURL, ok := settingsData["qqWSURL"].(string); ok {
			newConfig.QQWSURL = qqWSURL
		} else if qqWSURL, ok := settingsData["QQWSURL"].(string); ok {
			newConfig.QQWSURL = qqWSURL
		}

		// 处理HTTP URL
		if qqHTTPURL, ok := settingsData["qqHTTPURL"].(string); ok {
			newConfig.QQHTTPURL = qqHTTPURL
		} else if qqHTTPURL, ok := settingsData["QQHTTPURL"].(string); ok {
			newConfig.QQHTTPURL = qqHTTPURL
		}

		// 处理反向WebSocket端口
		if qqReverseWS, ok := settingsData["qqReverseWS"].(string); ok {
			newConfig.QQReverseWS = qqReverseWS
		} else if qqReverseWS, ok := settingsData["QQReverseWS"].(string); ok {
			newConfig.QQReverseWS = qqReverseWS
		}

		// 处理访问令牌
		if qqAccessToken, ok := settingsData["qqAccessToken"].(string); ok {
			newConfig.QQAccessToken = qqAccessToken
		} else if qqAccessToken, ok := settingsData["QQAccessToken"].(string); ok {
			newConfig.QQAccessToken = qqAccessToken
		}

		// 验证配置（使用宽松验证）
		if err := newConfig.ValidateForSave(); err != nil {
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
