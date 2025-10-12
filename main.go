package main

import (
	"island/config"
	"island/connection"
	"island/handlers"
	"island/web"
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
			HTTPPort:      "8088",
			QQWSURL:       "",
			QQHTTPURL:     "",
			QQReverseWS:   "",
			ConnectionMode: "websocket",
			QQGroupID:     []int64{},
		}
	}

	// 初始化连接管理器
	connManager := connection.NewConnectionManager(appConfig, 3) // 最大重试次数为3
	defer connManager.Close()

	// 根据配置的连接模式初始化连接
	if err := connManager.Connect(); err != nil {
		log.Printf("初始化连接失败: %v", err)
		log.Println("将在消息处理时尝试重新连接...")
	}

	// 初始化消息处理器
	msgHandler := handlers.NewMessageHandler(connManager, appConfig)

	// 启动Web服务器
	go web.StartHTTPServer(appConfig, msgHandler)

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

	// 保持主程序运行
	select {}
}
