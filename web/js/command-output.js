// 指令输出设置管理模块
const CommandOutputManager = {
    // 默认设置
    defaultSettings: {
        rollCommand: "掷骰结果: {dice} = {total} ({values})",
        cocCommand: "COC角色生成: 力量{str} 体质{con} 体型{siz} 敏捷{dex} 外貌{app} 智力{int} 意志{pow} 教育{edu} 幸运{luck}",
        dndCommand: "DND角色生成: 力量{str} 敏捷{dex} 体质{con} 智力{int} 感知{wis} 魅力{cha}",
        helpCommand: "可用命令: .r [骰子表达式] - 掷骰子, .coc - COC角色生成, .dnd - DND角色生成, .help - 显示帮助",
        unknownCommand: "未知命令: {command}"
    },

    // 当前设置
    currentSettings: {},

    // 加载指令输出设置
    async loadCommandOutputSettings() {
        try {
            const response = await fetch('/api/command-output-settings');
            if (response.ok) {
                const settings = await response.json();
                this.currentSettings = settings;
                this.renderSettings(settings);
                Utils.addMessage('system', '指令输出设置已加载');
            } else {
                throw new Error('获取设置失败');
            }
        } catch (error) {
            Utils.addMessage('system', '加载指令输出设置失败: ' + error.message);
            // 使用默认设置
            this.currentSettings = { ...this.defaultSettings };
            this.renderSettings(this.defaultSettings);
        }
    },

    // 渲染设置到界面
    renderSettings(settings) {
        document.getElementById('rollCommand').value = settings.rollCommand || this.defaultSettings.rollCommand;
        document.getElementById('cocCommand').value = settings.cocCommand || this.defaultSettings.cocCommand;
        document.getElementById('dndCommand').value = settings.dndCommand || this.defaultSettings.dndCommand;
        document.getElementById('helpCommand').value = settings.helpCommand || this.defaultSettings.helpCommand;
        document.getElementById('unknownCommand').value = settings.unknownCommand || this.defaultSettings.unknownCommand;
    },

    // 保存指令输出设置
    async saveCommandOutputSettings() {
        const settings = {
            rollCommand: document.getElementById('rollCommand').value.trim(),
            cocCommand: document.getElementById('cocCommand').value.trim(),
            dndCommand: document.getElementById('dndCommand').value.trim(),
            helpCommand: document.getElementById('helpCommand').value.trim(),
            unknownCommand: document.getElementById('unknownCommand').value.trim()
        };

        // 验证设置
        if (!this.validateSettings(settings)) {
            return;
        }

        const saveBtn = document.querySelector('#command-output-settings button[onclick="saveCommandOutputSettings()"]');
        const originalText = Utils.showLoading(saveBtn);

        try {
            const response = await fetch('/api/command-output-settings', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(settings)
            });

            const data = await response.json();

            if (response.ok) {
                this.currentSettings = settings;
                Utils.addMessage('system', '指令输出设置已保存');
            } else {
                throw new Error(data.error || '保存失败');
            }
        } catch (error) {
            Utils.addMessage('system', '保存指令输出设置失败: ' + error.message);
        } finally {
            Utils.restoreButton(saveBtn, originalText);
        }
    },

    // 验证设置
    validateSettings(settings) {
        const requiredFields = ['rollCommand', 'cocCommand', 'dndCommand', 'helpCommand', 'unknownCommand'];
        
        for (const field of requiredFields) {
            if (!settings[field]) {
                Utils.addMessage('system', `${this.getFieldDisplayName(field)}不能为空`);
                return false;
            }
        }

        // 验证必要的变量是否存在
        if (!settings.rollCommand.includes('{dice}') || !settings.rollCommand.includes('{total}')) {
            Utils.addMessage('system', '掷骰命令输出格式必须包含 {dice} 和 {total} 变量');
            return false;
        }

        if (!settings.unknownCommand.includes('{command}')) {
            Utils.addMessage('system', '未知命令输出格式必须包含 {command} 变量');
            return false;
        }

        return true;
    },

    // 获取字段显示名称
    getFieldDisplayName(field) {
        const names = {
            rollCommand: '掷骰命令输出格式',
            cocCommand: 'COC角色生成输出格式',
            dndCommand: 'DND角色生成输出格式',
            helpCommand: '帮助命令输出格式',
            unknownCommand: '未知命令输出格式'
        };
        return names[field] || field;
    },

    // 恢复默认设置
    resetCommandOutputSettings() {
        if (confirm('确定要恢复默认设置吗？这将覆盖当前的设置。')) {
            this.renderSettings(this.defaultSettings);
            Utils.addMessage('system', '已恢复默认设置');
        }
    },

    // 格式化命令输出
    formatCommandOutput(commandType, variables = {}) {
        const template = this.currentSettings[commandType] || this.defaultSettings[commandType];
        
        if (!template) {
            return `未知命令类型: ${commandType}`;
        }

        // 替换变量
        let output = template;
        for (const [key, value] of Object.entries(variables)) {
            output = output.replace(new RegExp(`\\{${key}\\}`, 'g'), value);
        }

        return output;
    },

    // 获取当前设置
    getCurrentSettings() {
        return { ...this.currentSettings };
    },

    // 初始化指令输出管理器
    async init() {
        await this.loadCommandOutputSettings();
        Utils.addMessage('system', '指令输出管理器已初始化');
    }
};

// 全局导出
window.CommandOutputManager = CommandOutputManager;
window.saveCommandOutputSettings = () => CommandOutputManager.saveCommandOutputSettings();
window.resetCommandOutputSettings = () => CommandOutputManager.resetCommandOutputSettings();
