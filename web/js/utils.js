// 工具函数模块
const Utils = {
    // 添加消息到输出区域
    addMessage(type, content) {
        const output = document.getElementById('output');
        const timestamp = new Date().toLocaleTimeString();
        
        const messageDiv = document.createElement('div');
        messageDiv.className = `message ${type}`;
        
        let icon = 'fa-terminal';
        let headerText = '命令';
        
        if (type === 'result') {
            icon = 'fa-dice';
            headerText = '结果';
        } else if (type === 'system') {
            icon = 'fa-info-circle';
            headerText = '系统';
        }
        
        messageDiv.innerHTML = `
            <div class="message-header">
                <span class="timestamp">${timestamp}</span>
                <i class="fas ${icon} message-icon"></i>
                <span>${headerText}</span>
            </div>
            <div class="message-content">${content}</div>
        `;
        
        output.appendChild(messageDiv);
        output.scrollTop = output.scrollHeight;
    },

    // 模拟骰子投掷
    simulateDiceRoll(command) {
        if (command.startsWith('.r ')) {
            const diceExpr = command.substring(3);
            return `掷骰结果: ${diceExpr} = 15 (6 + 5 + 4)`;
        } else if (command === '.coc') {
            return 'COC角色生成: 力量65 体质70 体型55 敏捷60 外貌75 智力80 意志65 教育70 幸运50';
        } else if (command === '.dnd') {
            return 'DND角色生成: 力量14 敏捷16 体质12 智力18 感知15 魅力13';
        } else if (command === '.help') {
            return '可用命令: .r [骰子表达式] - 掷骰子, .coc - COC角色生成, .dnd - DND角色生成, .help - 显示帮助';
        } else {
            return `未知命令: ${command}`;
        }
    },

    // 处理回车键
    handleKeyPress(event) {
        if (event.key === 'Enter') {
            sendCommand();
        }
    },

    // 显示加载状态
    showLoading(element) {
        const originalText = element.textContent;
        element.innerHTML = '<i class="fas fa-spinner fa-spin"></i> 加载中...';
        element.disabled = true;
        return originalText;
    },

    // 恢复按钮状态
    restoreButton(element, originalText) {
        element.innerHTML = originalText;
        element.disabled = false;
    },

    // 验证数字输入
    validateNumber(input, min = 1, max = 65535) {
        const value = parseInt(input);
        if (isNaN(value) || value < min || value > max) {
            return false;
        }
        return true;
    },

    // 验证URL格式
    validateURL(url) {
        try {
            new URL(url);
            return true;
        } catch {
            return false;
        }
    },

    // 格式化时间戳
    formatTimestamp(date = new Date()) {
        return date.toLocaleTimeString('zh-CN', {
            hour12: false,
            hour: '2-digit',
            minute: '2-digit',
            second: '2-digit'
        });
    },

    // 防抖函数
    debounce(func, wait) {
        let timeout;
        return function executedFunction(...args) {
            const later = () => {
                clearTimeout(timeout);
                func(...args);
            };
            clearTimeout(timeout);
            timeout = setTimeout(later, wait);
        };
    },

    // 节流函数
    throttle(func, limit) {
        let inThrottle;
        return function(...args) {
            if (!inThrottle) {
                func.apply(this, args);
                inThrottle = true;
                setTimeout(() => inThrottle = false, limit);
            }
        };
    },

    // 深拷贝对象
    deepClone(obj) {
        if (obj === null || typeof obj !== 'object') return obj;
        if (obj instanceof Date) return new Date(obj);
        if (obj instanceof Array) return obj.map(item => this.deepClone(item));
        if (obj instanceof Object) {
            const clonedObj = {};
            for (const key in obj) {
                if (obj.hasOwnProperty(key)) {
                    clonedObj[key] = this.deepClone(obj[key]);
                }
            }
            return clonedObj;
        }
    },

    // 生成随机ID
    generateId(length = 8) {
        const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
        let result = '';
        for (let i = 0; i < length; i++) {
            result += chars.charAt(Math.floor(Math.random() * chars.length));
        }
        return result;
    },

    // 检查网络连接状态
    checkNetworkStatus() {
        return navigator.onLine;
    },

    // 复制文本到剪贴板
    copyToClipboard(text) {
        return new Promise((resolve, reject) => {
            if (navigator.clipboard && window.isSecureContext) {
                navigator.clipboard.writeText(text).then(resolve).catch(reject);
            } else {
                const textArea = document.createElement('textarea');
                textArea.value = text;
                textArea.style.position = 'fixed';
                textArea.style.opacity = '0';
                document.body.appendChild(textArea);
                textArea.select();
                try {
                    document.execCommand('copy');
                    resolve();
                } catch (err) {
                    reject(err);
                }
                document.body.removeChild(textArea);
            }
        });
    }
};

// 全局导出
window.Utils = Utils;
