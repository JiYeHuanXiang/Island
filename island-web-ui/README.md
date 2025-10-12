# Island Web UI

COC骰子机器人的Web界面，用于管理和控制Island:dice机器人。

## 功能特性

- 🎲 COC骰子命令支持
- 🔌 WebSocket连接管理
- 👥 群组管理功能
- 🔧 管理员设置
- 📱 响应式设计
- 💾 本地存储设置

## 快速开始

1. 克隆仓库
```bash
git clone https://github.com/your-username/island-web-ui.git
cd island-web-ui
```

2. 打开 `index.html` 文件
```bash
# 在浏览器中打开
open index.html
# 或使用本地服务器
python -m http.server 8000
```

3. 配置连接
   - 输入WebSocket地址（例如：`ws://127.0.0.1:3009`）
   - 点击"连接"按钮
   - 开始使用骰子命令

## 使用说明

### 骰子命令
- `.coc7` - 生成7版COC调查员属性
- `.ra 70` - 进行技能值为70的检定
- `.r 3d6` - 投掷3个6面骰
- `.sc 1/1d6` - 理智检定
- `.help` - 查看完整指令列表

### 管理员功能
- 设置管理员QQ号
- 查看和管理机器人加入的群组
- 启用/禁用群组功能
- 退出群组

## 技术栈

- HTML5
- CSS3 (CSS变量、Flexbox、Grid)
- JavaScript (ES6+)
- WebSocket API
- LocalStorage

## 项目结构

```
island-web-ui/
├── index.html          # 主界面文件
├── README.md           # 项目说明
└── package.json        # 项目配置
```

## 开发

这是一个纯前端项目，无需构建过程。直接编辑 `index.html` 文件即可。

## 许可证

BSD 3-Clause License
