package webview

import (
	"island/config"
	"log"
	"runtime"
	"time"

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
		Debug: false, // 生产环境关闭调试
	})

	if w == nil {
		log.Printf("创建嵌入式浏览器失败")
		log.Println("将回退到系统默认浏览器...")
		OpenBrowser(appConfig)
		return
	}

	// 注意：当前webview2库版本可能不支持SetOnClose方法
	// 窗口关闭时程序会继续在后台运行

	log.Printf("正在启动嵌入式GUI: http://localhost:%s", appConfig.HTTPPort)

	// 等待Web服务器完全启动
	time.Sleep(1 * time.Second)

	// 导航到本地Web服务器
	url := "http://localhost:" + appConfig.HTTPPort
	w.Navigate(url)

	log.Printf("嵌入式GUI已启动，如需使用浏览器访问: %s", url)

	// 运行WebView消息循环
	w.Run()
}

// OpenEmbeddedBrowserAsync 异步打开嵌入式浏览器
func OpenEmbeddedBrowserAsync(appConfig *config.Config) {
	go func() {
		OpenEmbeddedBrowser(appConfig)
	}()
}
