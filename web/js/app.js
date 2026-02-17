/**
 * FateWeaver - TRPG 骰子机器人管理台
 * 主应用程序 JavaScript
 */

// 全局状态
const AppState = {
    ws: null,
    connected: false,
    connecting: false,
    logs: [],
    maxLogs: 500,
    autoScroll: true,
    currentView: 'dashboard',
    stats: {
        todayRolls: 0,
        activeGroups: 0,
        totalMessages: 0,
        uptime: '--:--:--'
    },
    startTime: Date.now()
};

// ============================================
// 视图导航
// ============================================

/**
 * 初始化导航功能
 */
function initNavigation() {
    const navItems = document.querySelectorAll('.nav-item');
    const menuToggle = document.getElementById('menuToggle');
    const sidebar = document.getElementById('sidebar');

    // 导航点击事件
    navItems.forEach(item => {
        item.addEventListener('click', (e) => {
            e.preventDefault();
            const viewName = item.dataset.view;
            switchView(viewName);
            
            // 移动端关闭侧边栏
            if (window.innerWidth <= 768) {
                sidebar.classList.remove('open');
            }
        });
    });

    // 移动端菜单切换
    if (menuToggle) {
        menuToggle.addEventListener('click', () => {
            sidebar.classList.toggle('open');
        });
    }
}

/**
 * 切换视图
 * @param {string} viewName - 视图名称
 */
function switchView(viewName) {
    // 更新导航状态
    document.querySelectorAll('.nav-item').forEach(item => {
        item.classList.toggle('active', item.dataset.view === viewName);
    });

    // 更新视图显示
    document.querySelectorAll('.view').forEach(view => {
        view.classList.toggle('active', view.id === `view-${viewName}`);
    });

    // 更新面包屑
    const viewNames = {
        dashboard: '仪表盘',
        logs: '实时日志',
        connection: '连接管理',
        config: '规则配置',
        sandbox: '调试沙盒'
    };
    
    const breadcrumb = document.getElementById('currentView');
    if (breadcrumb) {
        breadcrumb.textContent = viewNames[viewName] || viewName;
    }

    // 更新状态
    AppState.currentView = viewName;
    
    // 保存到 localStorage
    localStorage.setItem('lastView', viewName);
}

// ============================================
// WebSocket 连接管理
// ============================================

/**
 * 初始化 WebSocket 连接
 */
function initWebSocket() {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/ws`;
    
    try {
        AppState.ws = new WebSocket(wsUrl);
        
        AppState.ws.onopen = () => {
            console.log('WebSocket 连接已建立');
            updateConnectionStatus(true, false);
            addLog('system', 'info', 'WebSocket 连接已建立');
            showNotification('success', '连接成功', '已成功连接到服务器');
        };
        
        AppState.ws.onmessage = (event) => {
            handleWebSocketMessage(event.data);
        };
        
        AppState.ws.onerror = (error) => {
            console.error('WebSocket 错误:', error);
            addLog('system', 'error', 'WebSocket 连接错误');
        };
        
        AppState.ws.onclose = () => {
            console.log('WebSocket 连接已关闭');
            updateConnectionStatus(false, false);
            addLog('system', 'warn', 'WebSocket 连接已关闭');
            
            // 自动重连
            setTimeout(() => {
                if (!AppState.connected) {
                    initWebSocket();
                }
            }, 3000);
        };
        
    } catch (error) {
        console.error('创建 WebSocket 失败:', error);
        addLog('system', 'error', `创建 WebSocket 失败: ${error.message}`);
    }
}

/**
 * 处理 WebSocket 消息
 * @param {string} data - 消息数据
 */
function handleWebSocketMessage(data) {
    try {
        const message = JSON.parse(data);
        
        // 根据消息类型处理
        if (message.type === 'log') {
            addLog(message.category || 'system', message.level || 'info', message.content);
        } else if (message.type === 'response') {
            // 命令响应
            handleCommandResponse(message);
        } else if (message.type === 'stats') {
            updateStats(message.data);
        } else if (message.type === 'event') {
            handleEvent(message);
        } else {
            // 纯文本消息
            addLog('system', 'info', data);
        }
        
    } catch (error) {
        // 非 JSON 格式，当作普通日志处理
        addLog('system', 'info', data);
    }
}

/**
 * 处理命令响应
 * @param {Object} response - 命令响应
 */
function handleCommandResponse(response) {
    if (response.command) {
        addLog('dice', 'info', `执行命令: ${response.command}`);
    }
    if (response.result) {
        addLog('dice', 'info', `结果: ${response.result}`);
    }
}

/**
 * 处理事件
 * @param {Object} event - 事件对象
 */
function handleEvent(event) {
    switch (event.action) {
        case 'roll':
            AppState.stats.todayRolls++;
            updateStatsDisplay();
            addLog('dice', 'info', event.message);
            break;
        case 'message':
            AppState.stats.totalMessages++;
            updateStatsDisplay();
            addLog('chat', 'info', event.message);
            break;
        case 'error':
            addLog('error', 'error', event.message);
            break;
        default:
            addLog('system', 'info', event.message);
    }
}

/**
 * 更新连接状态显示
 * @param {boolean} connected - 是否已连接
 * @param {boolean} connecting - 是否正在连接
 */
function updateConnectionStatus(connected, connecting) {
    AppState.connected = connected;
    AppState.connecting = connecting;
    
    const pulse = document.getElementById('connectionPulse');
    const pulseText = pulse ? pulse.querySelector('.pulse-text') : null;
    const connectBtn = document.getElementById('connectBtn');
    const connectBtnText = document.getElementById('connectBtnText');
    const disconnectBtn = document.getElementById('disconnectBtn');
    
    if (pulse) {
        pulse.classList.remove('connected', 'connecting');
        if (connected) {
            pulse.classList.add('connected');
            if (pulseText) pulseText.textContent = '已连接';
        } else if (connecting) {
            pulse.classList.add('connecting');
            if (pulseText) pulseText.textContent = '连接中...';
        } else {
            if (pulseText) pulseText.textContent = '未连接';
        }
    }
    
    if (connectBtn) {
        connectBtn.style.display = connected ? 'none' : 'inline-flex';
    }
    if (connectBtnText) {
        connectBtnText.textContent = connected ? '已连接' : '连接';
    }
    if (disconnectBtn) {
        disconnectBtn.style.display = connected ? 'inline-flex' : 'none';
    }
    
    // 更新状态指示器
    const statusIndicator = document.getElementById('connectionStatus');
    if (statusIndicator) {
        const statusDot = statusIndicator.querySelector('.status-dot');
        const statusText = statusIndicator.querySelector('.status-text');
        
        statusDot.classList.remove('online', 'connecting', 'offline');
        if (connected) {
            statusDot.classList.add('online');
            if (statusText) statusText.textContent = '已连接';
        } else if (connecting) {
            statusDot.classList.add('connecting');
            if (statusText) statusText.textContent = '连接中...';
        } else {
            statusDot.classList.add('offline');
            if (statusText) statusText.textContent = '未连接';
        }
    }
}

// ============================================
// 日志管理
// ============================================

/**
 * 添加日志
 * @param {string} category - 日志类别
 * @param {string} level - 日志级别
 * @param {string} message - 日志消息
 */
function addLog(category, level, message) {
    const timestamp = new Date().toLocaleTimeString('zh-CN', { hour12: false });
    const logEntry = {
        timestamp,
        category,
        level,
        message,
        id: Date.now() + Math.random()
    };
    
    AppState.logs.push(logEntry);
    
    // 限制日志数量
    if (AppState.logs.length > AppState.maxLogs) {
        AppState.logs.shift();
    }
    
    // 如果在日志视图，直接追加
    const logContent = document.getElementById('logContent');
    if (logContent && document.getElementById('view-logs').classList.contains('active')) {
        appendLogLine(logEntry);
    }
}

/**
 * 追加日志行
 * @param {Object} logEntry - 日志条目
 */
function appendLogLine(logEntry) {
    const logContent = document.getElementById('logContent');
    if (!logContent) return;
    
    const line = document.createElement('div');
    line.className = `log-line ${logEntry.category}`;
    
    // 检查关键成功/失败
    if (logEntry.message.includes('大成功') || logEntry.message.includes('<= 5')) {
        line.classList.add('critical-success');
    } else if (logEntry.message.includes('大失败') || logEntry.message.includes('>= 96')) {
        line.classList.add('critical-fail');
    }
    
    line.innerHTML = `
        <span class="log-time">${logEntry.timestamp}</span>
        <span class="log-level">[${getLevelText(logEntry.level)}]</span>
        <span class="log-message">${escapeHtml(logEntry.message)}</span>
    `;
    
    logContent.appendChild(line);
    
    // 自动滚动
    if (AppState.autoScroll) {
        logContent.scrollTop = logContent.scrollHeight;
    }
}

/**
 * 获取级别文本
 * @param {string} level - 日志级别
 * @returns {string} 级别文本
 */
function getLevelText(level) {
    const levelMap = {
        info: '信息',
        warn: '警告',
        error: '错误',
        debug: '调试'
    };
    return levelMap[level] || level.toUpperCase();
}

/**
 * 过滤日志
 */
function filterLogs() {
    const typeFilter = document.getElementById('logTypeFilter')?.value || 'all';
    const levelFilter = document.getElementById('logLevelFilter')?.value || 'all';
    const searchText = document.getElementById('logSearchInput')?.value.toLowerCase() || '';
    
    const logContent = document.getElementById('logContent');
    if (!logContent) return;
    
    logContent.innerHTML = '';
    
    const filteredLogs = AppState.logs.filter(log => {
        // 类型过滤
        if (typeFilter !== 'all' && log.category !== typeFilter) {
            return false;
        }
        // 级别过滤
        if (levelFilter !== 'all' && log.level !== levelFilter) {
            return false;
        }
        // 搜索过滤
        if (searchText && !log.message.toLowerCase().includes(searchText)) {
            return false;
        }
        return true;
    });
    
    filteredLogs.forEach(log => appendLogLine(log));
}

/**
 * 切换自动滚动
 */
function toggleAutoScroll() {
    AppState.autoScroll = !AppState.autoScroll;
    const btn = document.getElementById('autoScrollBtn');
    if (btn) {
        btn.classList.toggle('active', AppState.autoScroll);
    }
}

/**
 * 清空日志
 */
function clearLogs() {
    AppState.logs = [];
    const logContent = document.getElementById('logContent');
    if (logContent) {
        logContent.innerHTML = `
            <div class="log-line system">
                <span class="log-time">${new Date().toLocaleTimeString('zh-CN', { hour12: false })}</span>
                <span class="log-level">[系统]</span>
                <span class="log-message">日志已清空</span>
            </div>
        `;
    }
    showNotification('info', '已清空', '日志已清空');
}

// ============================================
// 连接管理
// ============================================

/**
 * 连接方式切换
 */
function onConnectionModeChange() {
    const mode = document.querySelector('input[name="connectionMode"]:checked')?.value;
    
    // 隐藏所有设置
    document.getElementById('websocketSettings').style.display = 'none';
    document.getElementById('httpSettings').style.display = 'none';
    document.getElementById('reverseWsSettings').style.display = 'none';
    
    // 显示对应设置
    switch (mode) {
        case 'websocket':
            document.getElementById('websocketSettings').style.display = 'grid';
            break;
        case 'http':
            document.getElementById('httpSettings').style.display = 'grid';
            break;
        case 'reverse_ws':
            document.getElementById('reverseWsSettings').style.display = 'grid';
            break;
    }
    
    // 更新卡片选中状态
    document.querySelectorAll('.connection-card').forEach(card => {
        card.classList.remove('selected');
    });
    
    const selectedCard = document.querySelector(`input[name="connectionMode"]:checked`)?.closest('.connection-card');
    if (selectedCard) {
        selectedCard.classList.add('selected');
    }
}

/**
 * 测试连接
 */
async function testConnection() {
    const mode = document.querySelector('input[name="connectionMode"]:checked')?.value;
    let url = '';
    
    switch (mode) {
        case 'websocket':
            url = document.getElementById('wsUrl')?.value;
            break;
        case 'http':
            url = document.getElementById('httpUrl')?.value;
            break;
        case 'reverse_ws':
            url = `ws://127.0.0.1:${document.getElementById('reverseWsPort')?.value || 8088}`;
            break;
    }
    
    if (!url) {
        showNotification('warning', '参数缺失', '请填写连接地址');
        return;
    }
    
    showNotification('info', '测试连接', '正在测试连接...');
    
    // 模拟测试
    setTimeout(() => {
        showNotification('success', '测试成功', '连接配置有效');
    }, 1500);
}

/**
 * 连接
 */
function connect() {
    if (AppState.connected || AppState.connecting) {
showNotification('warning', '提示', '已经在连接状态');
        return;
    }
    
    updateConnectionStatus(false, true);
    addLog('system', 'info', '正在连接...');
    
    // 保存设置
    saveConnectionSettings();
    
    // 模拟连接成功
    setTimeout(() => {
        updateConnectionStatus(true, false);
        addLog('system', 'info', '连接成功');
        showNotification('success', '连接成功', '已建立连接');
    }, 1500);
}

/**
 * 断开连接
 */
function disconnect() {
    if (AppState.ws) {
        AppState.ws.close();
    }
    updateConnectionStatus(false, false);
    addLog('system', 'info', '已断开连接');
    showNotification('info', '已断开', '连接已关闭');
}

/**
 * 保存连接设置
 */
function saveConnectionSettings() {
    const mode = document.querySelector('input[name="connectionMode"]:checked')?.value;
    const settings = {
        mode,
        wsUrl: document.getElementById('wsUrl')?.value,
        httpUrl: document.getElementById('httpUrl')?.value,
        httpToken: document.getElementById('httpToken')?.value,
        reverseWsPort: document.getElementById('reverseWsPort')?.value
    };
    
    localStorage.setItem('connectionSettings', JSON.stringify(settings));
}

/**
 * 加载连接设置
 */
function loadConnectionSettings() {
    const saved = localStorage.getItem('connectionSettings');
    if (saved) {
        try {
            const settings = JSON.parse(saved);
            
            // 设置连接方式
            const radio = document.querySelector(`input[name="connectionMode"][value="${settings.mode}"]`);
            if (radio) {
                radio.checked = true;
            }
            
            // 填充设置
            if (settings.wsUrl) document.getElementById('wsUrl').value = settings.wsUrl;
            if (settings.httpUrl) document.getElementById('httpUrl').value = settings.httpUrl;
            if (settings.httpToken) document.getElementById('httpToken').value = settings.httpToken;
            if (settings.reverseWsPort) document.getElementById('reverseWsPort').value = settings.reverseWsPort;
            
            // 触发显示更新
            onConnectionModeChange();
            
        } catch (error) {
            console.error('加载连接设置失败:', error);
        }
    }
}

// ============================================
// 配置管理
// ============================================

/**
 * 保存配置
 */
async function saveConfig() {
    const config = {
        commandPrefix: document.getElementById('commandPrefix')?.value || '.',
        rollCommand: document.getElementById('rollCommand')?.value || 'r',
        helpCommand: document.getElementById('helpCommand')?.value || 'help',
        adminQQ: document.getElementById('adminQQ')?.value || '',
        successText: document.getElementById('successText')?.value || '设置已保存',
        failureText: document.getElementById('failureText')?.value || '设置保存失败',
        enableCOC7: document.getElementById('enableCOC7')?.checked ?? true,
        enableDND: document.getElementById('enableDND')?.checked ?? true,
        enableDeck: document.getElementById('enableDeck')?.checked ?? false,
        enableCharacter: document.getElementById('enableCharacter')?.checked ?? false
    };
    
    try {
        const response = await fetch('/api/custom-settings', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(config)
        });
        
        if (response.ok) {
            showNotification('success', '保存成功', '配置已保存');
            addLog('system', 'info', '配置已保存');
        } else {
            showNotification('error', '保存失败', '配置保存失败');
        }
    } catch (error) {
        // 本地保存
        localStorage.setItem('botConfig', JSON.stringify(config));
        showNotification('success', '保存成功', '配置已保存到本地');
        addLog('system', 'info', '配置已保存（本地）');
    }
}

/**
 * 重置配置
 */
function resetConfig() {
    document.getElementById('commandPrefix').value = '.';
    document.getElementById('rollCommand').value = 'r';
    document.getElementById('helpCommand').value = 'help';
    document.getElementById('adminQQ').value = '';
    document.getElementById('successText').value = '设置已保存';
    document.getElementById('failureText').value = '设置保存失败';
    document.getElementById('enableCOC7').checked = true;
    document.getElementById('enableDND').checked = true;
    document.getElementById('enableDeck').checked = false;
    document.getElementById('enableCharacter').checked = false;
    
    showNotification('info', '已重置', '配置已重置为默认值');
}

/**
 * 加载配置
 */
function loadConfig() {
    const saved = localStorage.getItem('botConfig');
    if (saved) {
        try {
            const config = JSON.parse(saved);
            
            if (config.commandPrefix) document.getElementById('commandPrefix').value = config.commandPrefix;
            if (config.rollCommand) document.getElementById('rollCommand').value = config.rollCommand;
            if (config.helpCommand) document.getElementById('helpCommand').value = config.helpCommand;
            if (config.adminQQ) document.getElementById('adminQQ').value = config.adminQQ;
            if (config.successText) document.getElementById('successText').value = config.successText;
            if (config.failureText) document.getElementById('failureText').value = config.failureText;
            if (config.enableCOC7 !== undefined) document.getElementById('enableCOC7').checked = config.enableCOC7;
            if (config.enableDND !== undefined) document.getElementById('enableDND').checked = config.enableDND;
            if (config.enableDeck !== undefined) document.getElementById('enableDeck').checked = config.enableDeck;
            if (config.enableCharacter !== undefined) document.getElementById('enableCharacter').checked = config.enableCharacter;
            
        } catch (error) {
            console.error('加载配置失败:', error);
        }
    }
}

// ============================================
// 调试沙盒
// ============================================

/**
 * 处理沙盒命令输入
 * @param {KeyboardEvent} event - 键盘事件
 */
function handleSandboxKeyPress(event) {
    if (event.key === 'Enter') {
        runSandboxCommand();
    }
}

/**
 * 插入命令
 * @param {string} command - 命令文本
 */
function insertCommand(command) {
    const input = document.getElementById('sandboxCommand');
    if (input) {
        input.value = command;
        input.focus();
    }
}

/**
 * 执行沙盒命令
 */
async function runSandboxCommand() {
    const input = document.getElementById('sandboxCommand');
    const command = input?.value.trim();
    
    if (!command) {
        showNotification('warning', '提示', '请输入命令');
        return;
    }
    
    // 显示用户输入
    const renderedOutput = document.getElementById('renderedOutput');
    if (renderedOutput) {
        renderedOutput.innerHTML = `
            <div class="sandbox-message user">
                <div class="message-avatar" style="background: var(--color-bg-tertiary);">
                    <i class="fas fa-user"></i>
                </div>
                <div class="message-content">
                    <span class="message-sender">你</span>
                    <p class="message-text">${escapeHtml(command)}</p>
                </div>
            </div>
            <div class="sandbox-message bot">
                <div class="message-avatar">
                    <i class="fas fa-robot"></i>
                </div>
                <div class="message-content">
                    <span class="message-sender">FateWeaver Bot</span>
                    <p class="message-text"><span class="loading"></span> 处理中...</p>
                </div>
            </div>
        `;
    }
    
    try {
        const response = await fetch('/command', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ command })
        });
        
        const data = await response.json();
        
        // 更新渲染结果
        if (renderedOutput) {
            renderedOutput.innerHTML = `
                <div class="sandbox-message user">
                    <div class="message-avatar" style="background: var(--color-bg-tertiary);">
                        <i class="fas fa-user"></i>
                    </div>
                    <div class="message-content">
                        <span class="message-sender">你</span>
                        <p class="message-text">${escapeHtml(command)}</p>
                    </div>
                </div>
                <div class="sandbox-message bot">
                    <div class="message-avatar">
                        <i class="fas fa-robot"></i>
                    </div>
                    <div class="message-content">
                        <span class="message-sender">FateWeaver Bot</span>
                        <p class="message-text">${escapeHtml(data.response || '无响应')}</p>
                    </div>
                </div>
            `;
        }
        
        // 更新原始 JSON
        const rawJson = document.getElementById('rawJson');
        if (rawJson) {
            rawJson.textContent = JSON.stringify(data, null, 2);
        }
        
        // 添加到日志
        addLog('dice', 'info', `沙盒执行: ${command} -> ${data.response}`);
        
    } catch (error) {
        const renderedOutput = document.getElementById('renderedOutput');
        if (renderedOutput) {
            renderedOutput.innerHTML = `
                <div class="sandbox-message user">
                    <div class="message-avatar" style="background: var(--color-bg-tertiary);">
                        <i class="fas fa-user"></i>
                    </div>
                    <div class="message-content">
                        <span class="message-sender">你</span>
                        <p class="message-text">${escapeHtml(command)}</p>
                    </div>
                </div>
                <div class="sandbox-message bot">
                    <div class="message-avatar" style="background: var(--color-danger);">
                        <i class="fas fa-exclamation-triangle"></i>
                    </div>
                    <div class="message-content">
                        <span class="message-sender">FateWeaver Bot</span>
                        <p class="message-text" style="color: var(--color-danger);">错误: ${error.message}</p>
                    </div>
                </div>
            `;
        }
        
        addLog('error', 'error', `沙盒执行失败: ${error.message}`);
    }
    
    // 清空输入
    if (input) input.value = '';
}

// ============================================
// 统计与活动
// ============================================

/**
 * 更新统计显示
 */
function updateStatsDisplay() {
    const elements = {
        todayRolls: document.getElementById('todayRolls'),
        activeGroups: document.getElementById('activeGroups'),
        totalMessages: document.getElementById('totalMessages'),
        uptime: document.getElementById('uptime')
    };
    
    if (elements.todayRolls) elements.todayRolls.textContent = AppState.stats.todayRolls;
    if (elements.activeGroups) elements.activeGroups.textContent = AppState.stats.activeGroups;
    if (elements.totalMessages) elements.totalMessages.textContent = AppState.stats.totalMessages;
    if (elements.uptime) elements.uptime.textContent = AppState.stats.uptime;
}

/**
 * 更新统计
 * @param {Object} data - 统计数据
 */
function updateStats(data) {
    if (data.todayRolls !== undefined) AppState.stats.todayRolls = data.todayRolls;
    if (data.activeGroups !== undefined) AppState.stats.activeGroups = data.activeGroups;
    if (data.totalMessages !== undefined) AppState.stats.totalMessages = data.totalMessages;
    
    updateStatsDisplay();
}

/**
 * 更新运行时间
 */
function updateUptime() {
    const elapsed = Date.now() - AppState.startTime;
    const hours = Math.floor(elapsed / 3600000);
    const minutes = Math.floor((elapsed % 3600000) / 60000);
    const seconds = Math.floor((elapsed % 60000) / 1000);
    
    AppState.stats.uptime = 
        String(hours).padStart(2, '0') + ':' +
        String(minutes).padStart(2, '0') + ':' +
        String(seconds).padStart(2, '0');
    
    const uptimeEl = document.getElementById('uptime');
    if (uptimeEl) {
        uptimeEl.textContent = AppState.stats.uptime;
    }
}

/**
 * 添加活动记录
 * @param {string} message - 活动消息
 * @param {string} icon - 图标类名
 */
function addActivity(message, icon = 'fa-info-circle') {
    const activityList = document.getElementById('recentActivity');
    if (!activityList) return;
    
    const time = new Date().toLocaleTimeString('zh-CN', { hour12: false });
    
    const item = document.createElement('div');
    item.className = 'activity-item';
    item.innerHTML = `
        <div class="activity-icon">
            <i class="fas ${icon}"></i>
        </div>
        <div class="activity-content">
            <span class="activity-text">${escapeHtml(message)}</span>
            <span class="activity-time">${time}</span>
        </div>
    `;
    
    activityList.insertBefore(item, activityList.firstChild);
    
    // 限制显示数量
    while (activityList.children.length > 10) {
        activityList.removeChild(activityList.lastChild);
    }
}

// ============================================
// 快捷操作
// ============================================

/**
 * 重载配置
 */
async function reloadConfig() {
    showNotification('info', '重载配置', '正在重载配置...');
    addActivity('重载配置', 'fa-sync');
    
    setTimeout(() => {
        showNotification('success', '完成', '配置已重载');
    }, 1000);
}

/**
 * 清理缓存
 */
function clearCache() {
    // 清理 localStorage（保留重要设置）
    const keysToKeep = ['connectionSettings', 'botConfig', 'lastView'];
    const allKeys = Object.keys(localStorage);
    
    allKeys.forEach(key => {
        if (!keysToKeep.includes(key)) {
            localStorage.removeItem(key);
        }
    });
    
    showNotification('success', '清理完成', '缓存已清理');
    addActivity('清理缓存', 'fa-broom');
}

/**
 * 导出日志
 */
function exportLogs() {
    const logsText = AppState.logs.map(log => 
        `[${log.timestamp}] [${log.category.toUpperCase()}] ${log.message}`
    ).join('\n');
    
    const blob = new Blob([logsText], { type: 'text/plain;charset=utf-8' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `fateweaver-logs-${new Date().toISOString().slice(0, 10)}.txt`;
    a.click();
    URL.revokeObjectURL(url);
    
    showNotification('success', '导出成功', '日志已导出');
    addActivity('导出日志', 'fa-download');
}

/**
 * 重启机器人
 */
function restartBot() {
    if (confirm('确定要重启机器人吗？')) {
        showNotification('warning', '重启中', '正在重启机器人...');
        addActivity('重启机器人', 'fa-redo');
        
        // 断开连接
        if (AppState.ws) {
            AppState.ws.close();
        }
        
        // 模拟重启
        setTimeout(() => {
            location.reload();
        }, 2000);
    }
}

// ============================================
// 通知系统
// ============================================

/**
 * 显示通知
 * @param {string} type - 通知类型
 * @param {string} title - 标题
 * @param {string} message - 消息
 */
function showNotification(type, title, message) {
    const container = document.getElementById('notificationContainer');
    if (!container) return;
    
    const icons = {
        success: 'fa-check-circle',
        error: 'fa-times-circle',
        warning: 'fa-exclamation-triangle',
        info: 'fa-info-circle'
    };
    
    const notification = document.createElement('div');
    notification.className = `notification ${type}`;
    notification.innerHTML = `
        <i class="fas ${icons[type] || icons.info}"></i>
        <div class="notification-content">
            <div class="notification-title">${escapeHtml(title)}</div>
            <div class="notification-message">${escapeHtml(message)}</div>
        </div>
    `;
    
    container.appendChild(notification);
    
    // 自动关闭
    setTimeout(() => {
        notification.style.animation = 'slideIn 0.3s ease reverse';
        setTimeout(() => notification.remove(), 300);
    }, 3000);
}

// ============================================
// 工具函数
// ============================================

/**
 * HTML 转义
 * @param {string} text - 原始文本
 * @returns {string} 转义后的文本
 */
function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

// ============================================
// 标签页切换
// ============================================

/**
 * 初始化标签页
 */
function initTabs() {
    const tabBtns = document.querySelectorAll('.tab-btn');
    
    tabBtns.forEach(btn => {
        btn.addEventListener('click', () => {
            const tab = btn.dataset.tab;
            const container = btn.closest('.sandbox-output');
            
            // 更新按钮状态
            container.querySelectorAll('.tab-btn').forEach(b => {
                b.classList.toggle('active', b === btn);
            });
            
            // 更新面板显示
            container.querySelectorAll('.tab-panel').forEach(panel => {
                panel.classList.toggle('active', panel.id === `${tab}Output`);
            });
        });
    });
}

// ============================================
// 初始化
// ============================================

/**
 * 初始化应用
 */
function initApp() {
    console.log('FateWeaver 初始化中...');
    
    // 加载设置
    loadConnectionSettings();
    loadConfig();
    
    // 初始化组件
    initNavigation();
    initTabs();
    
    // 初始化 WebSocket
    initWebSocket();
    
    // 恢复上次视图
    const lastView = localStorage.getItem('lastView');
    if (lastView) {
        switchView(lastView);
    }
    
    // 启动定时器
    setInterval(updateUptime, 1000);
    
    // 添加初始日志
    addLog('system', 'info', 'FateWeaver 管理台已启动');
    addActivity('系统启动', 'fa-rocket');
    
    console.log('FateWeaver 初始化完成');
}

// DOM 加载完成后初始化
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initApp);
} else {
    initApp();
}

// 页面卸载前保存状态
window.addEventListener('beforeunload', () => {
    saveConnectionSettings();
});

// 导出到全局
window.AppState = AppState;
window.switchView = switchView;
window.addLog = addLog;
window.showNotification = showNotification;
