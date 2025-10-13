// 设置管理模块
const SettingsManager = {
    // 全局状态
    state: {
        ws: null,
        isConnected: false,
        currentSettings: {}
    },

    // 标签页切换
    switchTab(tabId) {
        // 隐藏所有标签内容
        document.querySelectorAll('.admin-settings').forEach(tab => {
            tab.classList.remove('active');
        });
        
        // 移除所有标签的激活状态
        document.querySelectorAll('.tab').forEach(tab => {
            tab.classList.remove('active');
        });
        
        // 显示选中的标签内容
        document.getElementById(tabId).classList.add('active');
        
        // 激活选中的标签
        event.target.classList.add('active');
    },

    // 连接方式改变时的处理
    onConnectionModeChange() {
        const mode = document.getElementById('connectionMode').value;
        
        // 隐藏所有连接设置
        document.getElementById('websocketSettings').style.display = 'none';
        document.getElementById('httpSettings').style.display = 'none';
        document.getElementById('reverseWsSettings').style.display = 'none';
        
        // 显示选中的连接设置
        if (mode === 'websocket') {
            document.getElementById('websocketSettings').style.display = 'block';
        } else if (mode === 'http') {
            document.getElementById('httpSettings').style.display = 'block';
        } else if (mode === 'reverse_ws') {
            document.getElementById('reverseWsSettings').style.display = 'block';
        }
    },

    // 连接函数
    connect() {
        const httpPort = document.getElementById('httpPort').value || '8088';
        const connectionMode = document.getElementById('connectionMode').value;
        
        let connectionData = {
            httpPort: parseInt(httpPort),
            connectionMode: connectionMode
        };
        
        // 根据连接模式添加相应的配置
        if (connectionMode === 'websocket') {
            const wsUrl = document.getElementById('wsUrl').value;
            if (!wsUrl) {
                Utils.addMessage('system', '请输入WebSocket URL');
                return;
            }
            connectionData.qqWSURL = wsUrl;
        } else if (connectionMode === 'http') {
            const httpUrl = document.getElementById('httpUrl').value;
            const httpToken = document.getElementById('httpToken').value;
            if (!httpUrl) {
                Utils.addMessage('system', '请输入HTTP API URL');
                return;
            }
            connectionData.qqHTTPURL = httpUrl;
            if (httpToken) {
                connectionData.qqAccessToken = httpToken;
            }
        } else if (connectionMode === 'reverse_ws') {
            const reverseWsPort = document.getElementById('reverseWsPort').value;
            if (!reverseWsPort) {
                Utils.addMessage('system', '请输入反向WebSocket端口');
                return;
            }
            connectionData.qqReverseWS = reverseWsPort;
        }
        
        // 保存设置到本地存储
        this.saveConnectionSettings(connectionData);
        
        // 发送连接请求
        this.sendConnectionRequest(connectionData);
    },

    // 保存连接设置到本地存储
    saveConnectionSettings(settings) {
        localStorage.setItem('connectionSettings', JSON.stringify(settings));
        this.state.currentSettings = settings;
    },

    // 发送连接请求到服务器
    sendConnectionRequest(connectionData) {
        const connectBtn = document.getElementById('connectBtn');
        const originalText = Utils.showLoading(connectBtn);

        fetch('/api/settings', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify(connectionData)
        })
        .then(response => response.json())
        .then(data => {
            if (data.message && data.message.includes('成功')) {
                Utils.addMessage('system', '连接设置已保存');
                this.updateConnectionStatus(true);
            } else if (data.error) {
                Utils.addMessage('system', '连接设置保存失败: ' + data.error);
            } else {
                Utils.addMessage('system', '连接设置保存成功');
                this.updateConnectionStatus(true);
            }
        })
        .catch(error => {
            Utils.addMessage('system', '连接失败: ' + error.message);
        })
        .finally(() => {
            Utils.restoreButton(connectBtn, originalText);
        });
    },

    // 断开连接
    disconnect() {
        if (this.state.ws) {
            this.state.ws.close();
            this.state.ws = null;
        }
        this.updateConnectionStatus(false);
        Utils.addMessage('system', '已断开连接');
    },

    // 更新连接状态
    updateConnectionStatus(connected) {
        this.state.isConnected = connected;
        const statusElement = document.getElementById('connectionStatus');
        const connectBtn = document.getElementById('connectBtn');
        const disconnectBtn = document.getElementById('disconnectBtn');
        
        if (connected) {
            statusElement.className = 'status status-connected';
            statusElement.innerHTML = '<i class="fas fa-check-circle"></i> 已连接';
            connectBtn.style.display = 'none';
            disconnectBtn.style.display = 'inline-block';
        } else {
            statusElement.className = 'status status-disconnected';
            statusElement.innerHTML = '<i class="fas fa-exclamation-circle"></i> 未连接';
            connectBtn.style.display = 'inline-block';
            disconnectBtn.style.display = 'none';
        }
    },

    // 保存管理员设置
    saveAdminSettings() {
        const adminQQ = document.getElementById('adminQQ').value;
        if (!adminQQ) {
            Utils.addMessage('system', '请输入管理员QQ号');
            return;
        }
        
        // 保存到本地存储
        localStorage.setItem('adminQQ', adminQQ);
        Utils.addMessage('system', '管理员设置已保存');
    },

    // 刷新群组列表
    refreshGroups() {
        const refreshBtn = document.querySelector('#group-management button');
        const originalText = Utils.showLoading(refreshBtn);

        // 这里应该从服务器获取群组列表
        // 暂时使用模拟数据
        setTimeout(() => {
            this.renderGroupList(this.getMockGroups());
            Utils.restoreButton(refreshBtn, originalText);
            Utils.addMessage('system', '群组列表已刷新');
        }, 1000);
    },

    // 获取模拟群组数据
    getMockGroups() {
        return [
            { id: 123456789, name: '测试群组1', enabled: true },
            { id: 987654321, name: '测试群组2', enabled: false },
            { id: 555555555, name: '测试群组3', enabled: true }
        ];
    },

    // 渲染群组列表
    renderGroupList(groups) {
        const groupList = document.getElementById('groupList');
        groupList.innerHTML = '';

        groups.forEach(group => {
            const groupItem = document.createElement('div');
            groupItem.className = 'group-item';
            groupItem.innerHTML = `
                <div class="group-info">
                    <strong>${group.name}</strong><br>
                    <small>群号: ${group.id}</small>
                </div>
                <div class="group-actions">
                    <button class="${group.enabled ? 'btn-warning' : 'btn-success'} btn-sm" 
                            onclick="SettingsManager.toggleGroup(${group.id})">
                        <i class="fas fa-power-off"></i> ${group.enabled ? '禁用' : '启用'}
                    </button>
                    <button class="btn-danger btn-sm" onclick="SettingsManager.removeGroup(${group.id})">
                        <i class="fas fa-trash"></i> 删除
                    </button>
                </div>
            `;
            groupList.appendChild(groupItem);
        });
    },

    // 切换群组状态
    toggleGroup(groupId) {
        Utils.addMessage('system', `切换群组 ${groupId} 状态`);
        // 这里应该发送请求到服务器更新群组状态
        this.refreshGroups(); // 刷新列表以显示更新后的状态
    },

    // 删除群组
    removeGroup(groupId) {
        if (confirm(`确定要删除群组 ${groupId} 吗？`)) {
            Utils.addMessage('system', `已删除群组 ${groupId}`);
            // 这里应该发送请求到服务器删除群组
            this.refreshGroups(); // 刷新列表以显示更新后的状态
        }
    },

    // 加载保存的设置
    loadSavedSettings() {
        // 加载连接设置
        const savedSettings = localStorage.getItem('connectionSettings');
        if (savedSettings) {
            const settings = JSON.parse(savedSettings);
            document.getElementById('httpPort').value = settings.httpPort || '';
            document.getElementById('connectionMode').value = settings.connectionMode || 'websocket';
            
            if (settings.qqWSURL) document.getElementById('wsUrl').value = settings.qqWSURL;
            if (settings.qqHTTPURL) document.getElementById('httpUrl').value = settings.qqHTTPURL;
            if (settings.qqAccessToken) document.getElementById('httpToken').value = settings.qqAccessToken;
            if (settings.qqReverseWS) document.getElementById('reverseWsPort').value = settings.qqReverseWS;
            
            // 触发连接方式改变以显示正确的设置
            this.onConnectionModeChange();
        }
        
        // 加载管理员设置
        const adminQQ = localStorage.getItem('adminQQ');
        if (adminQQ) {
            document.getElementById('adminQQ').value = adminQQ;
        }
    },

    // 初始化设置管理器
    init() {
        // 设置发送目标切换监听器
        document.getElementById('sendToQQ').addEventListener('change', function() {
            const label = document.getElementById('targetLabel');
            label.textContent = this.checked ? '发送到QQ' : '本地处理';
        });

        // 加载保存的设置
        this.loadSavedSettings();

        Utils.addMessage('system', '设置管理器已初始化');
    }
};

// 全局导出
window.SettingsManager = SettingsManager;
