package web

import (
	"encoding/json"
	"fmt"
	"island/config"
	"island/handlers"
	"log"
	"net/http"
	"sync"
	"os"
	"path/filepath"
	"strings"
	"time"

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

// 获取可执行文件所在目录
func getExecutableDir() string {
	exe, err := os.Executable()
	if err != nil {
		log.Printf("获取可执行文件路径失败: %v", err)
		return "."
	}
	return filepath.Dir(exe)
}

// 获取web目录路径
func getWebDir() string {
	execDir := getExecutableDir()
	webDir := filepath.Join(execDir, "web")
	
	// 检查web目录是否存在
	if _, err := os.Stat(webDir); !os.IsNotExist(err) {
		return webDir
	}
	
	// 如果web目录在可执行文件目录中不存在，则尝试使用当前工作目录
	cwd, err := os.Getwd()
	if err == nil {
		webDirCwd := filepath.Join(cwd, "web")
		if _, err := os.Stat(webDirCwd); !os.IsNotExist(err) {
			return webDirCwd
		}
	}
	
	// 最后回退到相对路径
	log.Printf("警告: 未找到绝对web目录，使用相对路径")
	return "web"
}

type CommandRequest struct {
	Command string `json:"command"`
}

type CommandResponse struct {
	Response string `json:"response"`
}

// CustomSettings 自定义设置结构体
type CustomSettings struct {
	CommandPrefix string `json:"commandPrefix"`
	RollCommand   string `json:"rollCommand"`
	HelpCommand   string `json:"helpCommand"`
	SuccessText   string `json:"successText"`
	FailureText   string `json:"failureText"`
}

func StartHTTPServer(appConfig *config.Config, handler *handlers.MessageHandler) {
	msgHandler = handler
	
	// 获取web目录路径
	webDir := getWebDir()
	log.Printf("Web目录路径: %s", webDir)
	
	// 提前检查web目录是否存在
	if _, err := os.Stat(webDir); os.IsNotExist(err) {
		log.Printf("警告: Web目录不存在: %s", webDir)
	}
	
	// 使用http.FileServer提供静态文件服务，提高性能
	fs := http.FileServer(http.Dir(webDir))
	
	// 自定义处理函数，处理根路径并提供index.html
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// 如果是根路径，提供index.html
		if r.URL.Path == "/" || r.URL.Path == "/index.html" {
			// 设置缓存头以提高性能
			w.Header().Set("Cache-Control", "public, max-age=3600")
			http.ServeFile(w, r, filepath.Join(webDir, "index.html"))
			return
		}
		
		// 处理静态资源请求
		if strings.HasPrefix(r.URL.Path, "/css/") || 
		   strings.HasPrefix(r.URL.Path, "/js/") ||
		   strings.HasPrefix(r.URL.Path, "/images/") ||
		   strings.HasPrefix(r.URL.Path, "/fonts/") {
			// 设置缓存头以提高性能
			w.Header().Set("Cache-Control", "public, max-age=86400") // 静态资源缓存24小时
			// 使用http.StripPrefix移除URL前缀，然后使用FileServer处理
			http.StripPrefix("/", fs).ServeHTTP(w, r)
			return
		}
		
		// 其他路径返回404
		http.NotFound(w, r)
	})
	
	http.HandleFunc("/ws", handleWebSocket)
	http.HandleFunc("/command", handleCommand)
	http.HandleFunc("/api/settings", handleSettings)
	http.HandleFunc("/api/custom-settings", handleCustomSettings)
	
	// 绑定到127.0.0.1而不是所有接口，提高安全性和性能
	addr := "127.0.0.1:" + appConfig.HTTPPort
	log.Printf("Web服务器已启动 %s", addr)
	
	server := &http.Server{
		Addr: addr,
		// 设置读写超时以防止连接挂起
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		// 添加IdleTimeout以更好地管理连接
		IdleTimeout: 60 * time.Second,
	}
	
	if err := server.ListenAndServe(); err != nil {
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

		// 创建响应结构体，确保字段名与前端一致
		response := map[string]interface{}{
			"httpPort":        currentConfig.HTTPPort,
			"connectionMode":  currentConfig.ConnectionMode,
			"qqWSURL":         currentConfig.QQWSURL,
			"qqHTTPURL":       currentConfig.QQHTTPURL,
			"qqReverseWS":     currentConfig.QQReverseWS,
			"qqAccessToken":   currentConfig.QQAccessToken,
			"qqGroupID":       currentConfig.QQGroupID,
		}

		if err := json.NewEncoder(w).Encode(response); err != nil {
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

		// 从消息处理器获取当前配置
		currentConfig := msgHandler.GetCurrentConfig()
		if currentConfig == nil {
			http.Error(w, `{"error": "无法获取当前配置"}`, http.StatusInternalServerError)
			return
		}

		// 创建新配置（基于当前配置）
		newConfig := *currentConfig
		
		// 处理HTTP端口
		if httpPort, ok := settingsData["httpPort"].(string); ok && httpPort != "" {
			newConfig.HTTPPort = httpPort
		} else if httpPort, ok := settingsData["httpPort"].(float64); ok {
			newConfig.HTTPPort = fmt.Sprintf("%.0f", httpPort)
		}
		
		// 处理连接模式
		if connectionMode, ok := settingsData["connectionMode"].(string); ok && connectionMode != "" {
			newConfig.ConnectionMode = connectionMode
		}
		
		// 处理WebSocket URL
		if qqWSURL, ok := settingsData["qqWSURL"].(string); ok {
			newConfig.QQWSURL = qqWSURL
		}
		
		// 处理HTTP URL
		if qqHTTPURL, ok := settingsData["qqHTTPURL"].(string); ok {
			newConfig.QQHTTPURL = qqHTTPURL
		}
		
		// 处理反向WebSocket端口
		if qqReverseWS, ok := settingsData["qqReverseWS"].(string); ok {
			newConfig.QQReverseWS = qqReverseWS
		}
		
		// 处理访问令牌
		if qqAccessToken, ok := settingsData["qqAccessToken"].(string); ok {
			newConfig.QQAccessToken = qqAccessToken
		}

		// 处理群组ID
		if qqGroupID, ok := settingsData["qqGroupID"].([]interface{}); ok {
			var groupIDs []int64
			for _, id := range qqGroupID {
				if groupID, ok := id.(float64); ok {
					groupIDs = append(groupIDs, int64(groupID))
				}
			}
			newConfig.QQGroupID = groupIDs
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

// 处理自定义设置
func handleCustomSettings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case "GET":
		// 返回当前自定义设置
		settings := CustomSettings{
			CommandPrefix: ".",
			RollCommand:   "r",
			HelpCommand:   "help",
			SuccessText:   "设置已保存",
			FailureText:   "设置保存失败",
		}
		
		// 这里可以从配置文件或数据库中加载实际设置
		// 暂时返回默认值
		
		if err := json.NewEncoder(w).Encode(settings); err != nil {
			log.Printf("序列化自定义设置失败: %v", err)
			http.Error(w, `{"error": "序列化自定义设置失败"}`, http.StatusInternalServerError)
		}

	case "POST":
		// 保存自定义设置
		var customSettings CustomSettings
		if err := json.NewDecoder(r.Body).Decode(&customSettings); err != nil {
			http.Error(w, `{"error": "解析自定义设置失败"}`, http.StatusBadRequest)
			return
		}

		// 验证设置
		if customSettings.CommandPrefix == "" {
			customSettings.CommandPrefix = "." // 默认值
		}
		
		if customSettings.RollCommand == "" {
			customSettings.RollCommand = "r" // 默认值
		}
		
		if customSettings.HelpCommand == "" {
			customSettings.HelpCommand = "help" // 默认值
		}

		if customSettings.SuccessText == "" {
			customSettings.SuccessText = "设置已保存" // 默认值
		}

		if customSettings.FailureText == "" {
			customSettings.FailureText = "设置保存失败" // 默认值
		}

		// TODO: 实际保存设置到配置文件或数据库
		// 这里暂时只是接收并确认设置已被接收
		log.Printf("收到自定义设置: %+v", customSettings)

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "自定义设置已保存"})

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
