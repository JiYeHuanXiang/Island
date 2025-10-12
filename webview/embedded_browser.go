package webview

import (
	"island/config"
	"log"
	"runtime"

	"github.com/jchv/go-webview2"
)

// OpenEmbeddedBrowser 使用嵌入式WebView打开Web UI
func OpenEmbeddedBrowser(appConfig *config.Config) {
	// 确保在主线程中运行（WebView要求）
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// 创建WebView窗口
	w := webview2.NewWithOptions(webview2.WebViewOptions{
		WindowOptions: webview2.WindowOptions{
			Title:  "Island - QQ机器人管理界面",
			Width:  1200,
			Height: 800,
		},
		Debug: true,
	})

	if w == nil {
		log.Printf("创建嵌入式浏览器失败")
		log.Println("将回退到系统默认浏览器...")
		OpenBrowser(appConfig)
		return
	}

	log.Printf("正在启动嵌入式浏览器: http://localhost:%s", appConfig.HTTPPort)
	
	// 导航到本地Web服务器
	w.Navigate("http://localhost:" + appConfig.HTTPPort)
	
	// 运行WebView消息循环
	w.Run()
}

// OpenEmbeddedBrowserAsync 异步打开嵌入式浏览器
func OpenEmbeddedBrowserAsync(appConfig *config.Config) {
	go func() {
		OpenEmbeddedBrowser(appConfig)
	}()
}
