package main

import (
	"island/config"
	"island/connection"
	"island/handlers"
	"island/web"
	"island/webview"
	"log"
	"math/rand"
	"time"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	// 加载配置
	appConfig, err := config.LoadConfig()
	if err != nil {
		log.Printf("配置加载错误: %v", err)
		log.Println("使用默认配置继续运行...")
		appConfig = &config.Config{
			HTTPPort:       "8088",
			QQWSURL:        "",
			QQHTTPURL:      "",
			QQReverseWS:    "",
			ConnectionMode: "websocket",
			QQGroupID:      []int64{},
		}
	}

	// 初始化连接管理器（但不立即连接）
	connManager := connection.NewConnectionManager(appConfig, 3) // 最大重试次数为3
	defer connManager.Close()

	// 初始化消息处理器
	msgHandler := handlers.NewMessageHandler(connManager, appConfig)

	// 启动Web服务器
	go web.StartHTTPServer(appConfig, msgHandler)

	// 等待Web服务器启动
	time.Sleep(2 * time.Second)

	// 启动嵌入式GUI作为默认界面
	log.Println("正在启动嵌入式GUI...")
	go webview.OpenEmbeddedBrowserAsync(appConfig)

	// 如果配置完整，尝试连接
	if err := appConfig.Validate(); err == nil {
		log.Println("检测到完整配置，尝试连接...")
		if err := connManager.Connect(); err != nil {
			log.Printf("初始化连接失败: %v", err)
			log.Println("将在消息处理时尝试重新连接...")
		} else {
			// 根据连接模式启动相应的消息循环
			switch appConfig.ConnectionMode {
			case "websocket":
				msgHandler.StartWebSocketMessageLoop()
			case "reverse_websocket":
				// 反向WebSocket由客户端连接，不需要主动循环
				log.Println("反向WebSocket模式已启用，等待客户端连接...")
			case "http":
				log.Println("HTTP API模式已启用，等待HTTP请求...")
			default:
				log.Printf("未知连接模式: %s，使用WebSocket模式", appConfig.ConnectionMode)
				msgHandler.StartWebSocketMessageLoop()
			}
		}
	} else {
		log.Printf("配置不完整，等待用户通过Web界面配置: %v", err)
		log.Println("请通过Web界面配置连接设置后，程序将自动连接")
	}

	// 保持主程序运行
	select {}
}
