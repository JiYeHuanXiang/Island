# 指令处理问题修复报告

## 问题分析

经过检查发现，指令显示"未知指令"的问题主要有以下几个原因：

### 1. WebSocket连接问题
- **问题**: 前端没有建立WebSocket连接
- **原因**: `settings.js`中的连接逻辑只保存配置，没有实际建立WebSocket连接
- **影响**: 指令无法通过WebSocket发送到服务器

### 2. 命令发送逻辑缺失
- **问题**: 前端命令发送逻辑不完整
- **原因**: `app.js`中的命令发送只是模拟，没有实际发送到服务器
- **影响**: 用户输入的命令无法到达后端处理

### 3. 连接状态管理不完整
- **问题**: 连接状态管理混乱
- **原因**: 前端没有正确管理WebSocket连接状态
- **影响**: 用户无法知道是否已连接到服务器

## 修复方案

### 1. 修复WebSocket连接逻辑
在 `settings.js` 中添加了 `establishWebSocketConnection()` 函数：
```javascript
// 建立WebSocket连接
establishWebSocketConnection() {
    if (this.state.ws) {
        this.state.ws.close();
    }

    const wsUrl = `ws://localhost:${document.getElementById('httpPort').value || '8088'}/ws`;
    Utils.addMessage('system', '正在建立WebSocket连接...');

    try {
        this.state.ws = new WebSocket(wsUrl);
        
        this.state.ws.onopen = () => {
            Utils.addMessage('system', 'WebSocket连接已建立');
            this.updateConnectionStatus(true);
        };

        this.state.ws.onmessage = (event) => {
            try {
                const data = JSON.parse(event.data);
                if (data.response) {
                    Utils.addMessage('result', data.response);
                }
            } catch (e) {
                // 如果不是JSON，直接显示文本
                Utils.addMessage('result', event.data);
            }
        };

        this.state.ws.onerror = (error) => {
            Utils.addMessage('system', 'WebSocket连接错误: ' + error);
            this.updateConnectionStatus(false);
        };

        this.state.ws.onclose = () => {
            Utils.addMessage('system', 'WebSocket连接已关闭');
            this.updateConnectionStatus(false);
        };

    } catch (error) {
        Utils.addMessage('system', 'WebSocket连接失败: ' + error.message);
        this.updateConnectionStatus(false);
    }
}
```

### 2. 修复命令发送逻辑
在 `app.js` 中修复了命令发送逻辑：
```javascript
} else if (SettingsManager.state.isConnected) {
    // 发送到WebSocket
    if (SettingsManager.sendCommandToWebSocket(command)) {
        Utils.addMessage('system', '命令已发送到服务器');
    } else {
        Utils.addMessage('system', '命令发送失败，将在本地处理');
        setTimeout(() => {
            const result = Utils.simulateDiceRoll(command);
            Utils.addMessage('result', result);
        }, 500);
    }
}
```

### 3. 添加WebSocket命令发送函数
在 `settings.js` 中添加了 `sendCommandToWebSocket()` 函数：
```javascript
// 发送命令到WebSocket
sendCommandToWebSocket(command) {
    if (!this.state.ws || this.state.ws.readyState !== WebSocket.OPEN) {
        Utils.addMessage('system', 'WebSocket未连接，无法发送命令');
        return false;
    }

    try {
        this.state.ws.send(command);
        return true;
    } catch (error) {
        Utils.addMessage('system', '发送命令失败: ' + error.message);
        return false;
    }
}
```

### 4. 修复后端WebSocket处理
在 `server.go` 中改进了WebSocket消息处理：
```go
// 如果不是JSON格式，当作普通命令处理
command := string(message)
log.Printf("收到WebSocket命令: %s", command)
response := msgHandler.ProcessCommand(command)

// 发送响应回WebSocket客户端
if err := conn.WriteMessage(websocket.TextMessage, []byte(response)); err != nil {
    log.Printf("WebSocket写入错误: %v", err)
    break
}
```

### 5. 添加自动连接功能
在 `settings.js` 中添加了自动连接功能：
```javascript
// 自动连接
autoConnect() {
    // 检查是否有保存的连接设置
    const savedSettings = localStorage.getItem('connectionSettings');
    if (savedSettings) {
        try {
            const settings = JSON.parse(savedSettings);
            // 如果之前有连接，尝试重新连接
            if (settings.httpPort) {
                Utils.addMessage('system', '尝试自动连接...');
                this.establishWebSocketConnection();
            }
        } catch (e) {
            Utils.addMessage('system', '自动连接失败: 配置解析错误');
        }
    }
}
```

## 测试验证

### 1. 创建了测试页面
- `test_commands.html` - 完整的命令测试页面
- `test_ws.html` - 简单的WebSocket连接测试

### 2. 验证结果
- ✅ HTTP服务器正常运行 (端口8088)
- ✅ API端点正常响应
- ✅ 命令处理功能正常
- ✅ WebSocket连接可以建立
- ✅ 命令可以通过WebSocket发送和接收

### 3. 测试命令
测试了以下命令，都能正常处理：
- `.help` - 显示帮助信息
- `.r 3d6` - 掷骰子
- `.coc7` - COC角色生成
- `.ra 50` - 技能检定
- `.sc 50/30` - 理智检定
- `.dnd str` - DND属性生成

## 修复总结

通过以上修复，解决了指令处理的问题：

1. **WebSocket连接**: 前端现在可以正确建立WebSocket连接
2. **命令传递**: 命令可以通过WebSocket正确发送到服务器
3. **响应处理**: 服务器响应可以正确显示在前端
4. **状态管理**: 连接状态得到正确管理
5. **用户体验**: 用户可以看到命令发送状态和结果

现在指令不再显示"未知指令"，而是能够正确处理并返回相应的骰子结果。
