// 自定义设置模块
const CustomSettings = {
    // 默认设置
    defaultSettings: {
        commandPrefix: ".",
        rollCommand: "r",
        helpCommand: "help",
        successText: "设置已保存",
        failureText: "设置保存失败"
    },

    // 当前设置
    currentSettings: {},

    // 初始化自定义设置
    init() {
        this.loadSettings();
        Utils.addMessage('system', '自定义设置模块已初始化');
    },

    // 加载设置
    loadSettings() {
        const savedSettings = localStorage.getItem('customSettings');
        if (savedSettings) {
            this.currentSettings = JSON.parse(savedSettings);
        } else {
            this.currentSettings = {...this.defaultSettings};
        }

        // 更新UI
        if (document.getElementById('commandPrefix')) {
            document.getElementById('commandPrefix').value = this.currentSettings.commandPrefix || this.defaultSettings.commandPrefix;
        }
        
        if (document.getElementById('rollCommand')) {
            document.getElementById('rollCommand').value = this.currentSettings.rollCommand || this.defaultSettings.rollCommand;
        }
        
        if (document.getElementById('helpCommand')) {
            document.getElementById('helpCommand').value = this.currentSettings.helpCommand || this.defaultSettings.helpCommand;
        }
        
        if (document.getElementById('successText')) {
            document.getElementById('successText').value = this.currentSettings.successText || this.defaultSettings.successText;
        }
        
        if (document.getElementById('failureText')) {
            document.getElementById('failureText').value = this.currentSettings.failureText || this.defaultSettings.failureText;
        }
    },

    // 保存设置
    saveSettings() {
        const commandPrefix = document.getElementById('commandPrefix').value || this.defaultSettings.commandPrefix;
        const rollCommand = document.getElementById('rollCommand').value || this.defaultSettings.rollCommand;
        const helpCommand = document.getElementById('helpCommand').value || this.defaultSettings.helpCommand;
        const successText = document.getElementById('successText').value || this.defaultSettings.successText;
        const failureText = document.getElementById('failureText').value || this.defaultSettings.failureText;

        this.currentSettings = {
            commandPrefix: commandPrefix,
            rollCommand: rollCommand,
            helpCommand: helpCommand,
            successText: successText,
            failureText: failureText
        };

        localStorage.setItem('customSettings', JSON.stringify(this.currentSettings));
        Utils.addMessage('system', '自定义设置已保存到本地');

        // 发送到服务器保存
        this.saveSettingsToServer();
    },

    // 保存设置到服务器
    saveSettingsToServer() {
        const settingsData = {
            commandPrefix: this.currentSettings.commandPrefix,
            rollCommand: this.currentSettings.rollCommand,
            helpCommand: this.currentSettings.helpCommand,
            successText: this.currentSettings.successText,
            failureText: this.currentSettings.failureText
        };

        fetch('/api/custom-settings', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify(settingsData)
        })
        .then(response => response.json())
        .then(data => {
            if (data.message) {
                Utils.addMessage('system', data.message);
            } else if (data.error) {
                Utils.addMessage('system', '保存失败: ' + data.error);
            }
        })
        .catch(error => {
            console.error('保存自定义设置失败:', error);
            Utils.addMessage('system', '保存自定义设置失败: ' + error.message);
        });
    },

    // 获取当前指令前缀
    getCommandPrefix() {
        return this.currentSettings.commandPrefix || this.defaultSettings.commandPrefix;
    },
    
    // 获取投掷指令
    getRollCommand() {
        return this.currentSettings.rollCommand || this.defaultSettings.rollCommand;
    },
    
    // 获取帮助指令
    getHelpCommand() {
        return this.currentSettings.helpCommand || this.defaultSettings.helpCommand;
    },

    // 获取成功提示文本
    getSuccessText() {
        return this.currentSettings.successText || this.defaultSettings.successText;
    },

    // 获取失败提示文本
    getFailureText() {
        return this.currentSettings.failureText || this.defaultSettings.failureText;
    }
};

// 全局函数
function saveCustomSettings() {
    CustomSettings.saveSettings();
}

// 初始化模块
document.addEventListener('DOMContentLoaded', function() {
    CustomSettings.init();
});

// 全局导出
window.CustomSettings = CustomSettings;