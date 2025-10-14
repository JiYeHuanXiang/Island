// 主应用程序模块
const App = {
    // 应用程序状态
    state: {
        isConnected: false,
        currentCommand: '',
        messageHistory: []
    },

    // 发送命令
    sendCommand() {
        const commandInput = document.getElementById('commandInput');
        const command = commandInput.value.trim();
        
        if (!command) {
            return;
        }
        
        Utils.addMessage('command', command);
        commandInput.value = '';
        
        // 如果是本地处理，直接显示结果
        if (!document.getElementById('sendToQQ').checked) {
            // 模拟骰子结果
            setTimeout(() => {
                const result = Utils.simulateDiceRoll(command);
                Utils.addMessage('result', result);
            }, 500);
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
        } else {
            Utils.addMessage('system', '未连接到服务器，命令将在本地处理');
            setTimeout(() => {
                const result = Utils.simulateDiceRoll(command);
                Utils.addMessage('result', result);
            }, 500);
        }
    },

    // 处理回车键
    handleKeyPress(event) {
        if (event.key === 'Enter') {
            this.sendCommand();
        }
    },

    // 清除输出日志
    clearOutput() {
        const output = document.getElementById('output');
        output.innerHTML = '';
        Utils.addMessage('system', '输出日志已清除');
    },

    // 导出日志
    exportLogs() {
        const output = document.getElementById('output');
        const logs = output.innerText;
        
        if (!logs.trim()) {
            Utils.addMessage('system', '没有日志可导出');
            return;
        }

        const blob = new Blob([logs], { type: 'text/plain' });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `island-dice-logs-${new Date().toISOString().slice(0, 10)}.txt`;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        URL.revokeObjectURL(url);
        
        Utils.addMessage('system', '日志已导出');
    },

    // 切换主题
    toggleTheme() {
        const currentTheme = document.documentElement.getAttribute('data-theme');
        const newTheme = currentTheme === 'dark' ? 'light' : 'dark';
        
        document.documentElement.setAttribute('data-theme', newTheme);
        localStorage.setItem('theme', newTheme);
        
        Utils.addMessage('system', `已切换到${newTheme === 'dark' ? '深色' : '浅色'}主题`);
    },

    // 检查更新
    checkForUpdates() {
        Utils.addMessage('system', '正在检查更新...');
        
        // 模拟检查更新
        setTimeout(() => {
            Utils.addMessage('system', '当前已是最新版本');
        }, 1500);
    },

    // 显示系统信息
    showSystemInfo() {
        const info = `
系统信息:
- 用户代理: ${navigator.userAgent}
- 语言: ${navigator.language}
- 平台: ${navigator.platform}
- 在线状态: ${navigator.onLine ? '在线' : '离线'}
- 屏幕分辨率: ${screen.width}x${screen.height}
- 颜色深度: ${screen.colorDepth}位
- 时区: ${Intl.DateTimeFormat().resolvedOptions().timeZone}
        `.trim();
        
        Utils.addMessage('system', info);
    },

    // 初始化键盘快捷键
    initKeyboardShortcuts() {
        document.addEventListener('keydown', (event) => {
            // Ctrl + L: 清除日志
            if (event.ctrlKey && event.key === 'l') {
                event.preventDefault();
                this.clearOutput();
            }
            
            // Ctrl + E: 导出日志
            if (event.ctrlKey && event.key === 'e') {
                event.preventDefault();
                this.exportLogs();
            }
            
            // Ctrl + T: 切换主题
            if (event.ctrlKey && event.key === 't') {
                event.preventDefault();
                this.toggleTheme();
            }
            
            // Ctrl + U: 检查更新
            if (event.ctrlKey && event.key === 'u') {
                event.preventDefault();
                this.checkForUpdates();
            }
            
            // Ctrl + I: 显示系统信息
            if (event.ctrlKey && event.key === 'i') {
                event.preventDefault();
                this.showSystemInfo();
            }
        });
    },

    // 初始化事件监听器
    initEventListeners() {
        // 命令输入框回车键监听
        const commandInput = document.getElementById('commandInput');
        commandInput.addEventListener('keypress', (event) => {
            this.handleKeyPress(event);
        });

        // 网络状态监听
        window.addEventListener('online', () => {
            Utils.addMessage('system', '网络连接已恢复');
        });

        window.addEventListener('offline', () => {
            Utils.addMessage('system', '网络连接已断开');
        });

        // 页面可见性监听
        document.addEventListener('visibilitychange', () => {
            if (!document.hidden) {
                // 页面变为可见时刷新连接状态
                if (SettingsManager.state.isConnected) {
                    Utils.addMessage('system', '页面已恢复显示');
                }
            }
        });
    },

    // 初始化应用程序
    init() {
        // 初始化设置管理器
        SettingsManager.init();
        
        // 初始化事件监听器
        this.initEventListeners();
        
        // 初始化键盘快捷键
        this.initKeyboardShortcuts();
        
        // 加载保存的主题
        const savedTheme = localStorage.getItem('theme') || 'light';
        document.documentElement.setAttribute('data-theme', savedTheme);
        
        Utils.addMessage('system', '应用程序已初始化完成');
        Utils.addMessage('system', '系统已就绪，请配置连接设置后开始使用');
        
        // 显示键盘快捷键帮助
        setTimeout(() => {
            Utils.addMessage('system', '提示: 使用 Ctrl+L 清除日志, Ctrl+E 导出日志, Ctrl+T 切换主题');
        }, 2000);
    }
};

// 全局函数（为了向后兼容）
function switchTab(tabId) {
    SettingsManager.switchTab(tabId);
}

function onConnectionModeChange() {
    SettingsManager.onConnectionModeChange();
}

function connect() {
    SettingsManager.connect();
}

function disconnect() {
    SettingsManager.disconnect();
}

function saveAdminSettings() {
    SettingsManager.saveAdminSettings();
}

function refreshGroups() {
    SettingsManager.refreshGroups();
}

function toggleGroup(groupId) {
    SettingsManager.toggleGroup(groupId);
}

function removeGroup(groupId) {
    SettingsManager.removeGroup(groupId);
}

function sendCommand() {
    App.sendCommand();
}

function handleKeyPress(event) {
    App.handleKeyPress(event);
}

// 页面加载时初始化
window.addEventListener('DOMContentLoaded', () => {
    App.init();
});

// 全局导出
window.App = App;
