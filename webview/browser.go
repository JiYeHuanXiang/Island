package webview

import (
	"island/config"
	"log"
	"os/exec"
	"runtime"
	"time"
)

// OpenBrowser 使用系统默认浏览器打开Web UI（回退方案）
func OpenBrowser(appConfig *config.Config) {
	// 等待Web服务器完全启动
	time.Sleep(3 * time.Second)

	url := "http://localhost:" + appConfig.HTTPPort
	log.Printf("正在打开系统默认浏览器: %s", url)

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	default:
		log.Printf("不支持的操作系统: %s", runtime.GOOS)
		return
	}

	if err := cmd.Start(); err != nil {
		log.Printf("打开系统浏览器失败: %v", err)
	} else {
		log.Println("系统浏览器已成功打开")
	}
}

// OpenBrowserAsync 异步打开浏览器
func OpenBrowserAsync(appConfig *config.Config) {
	// 优先尝试使用嵌入式浏览器
	log.Println("尝试启动嵌入式浏览器...")
	go func() {
		// 等待Web服务器启动
		time.Sleep(3 * time.Second)
		
		// 尝试使用嵌入式浏览器
		OpenEmbeddedBrowser(appConfig)
	}()
}
