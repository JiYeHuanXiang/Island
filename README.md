# Island:dice - COC TRPG骰子机器人

**本项目所有代码均由人工智能生成**  
`开源协议：BSD 3-Clause License`

---

## 📜 项目说明

这是一个完整的COC(克苏鲁的呼唤)TRPG骰子机器人，运行在go-cqhttp（OneBot格式）平台上。项目支持COC TRPG游戏中的各种投掷指令，并扩展了DND 5e系统。

### 主要特性

- ✅ **完整的COC TRPG指令系统**
- ✅ **扩展的DND 5e指令系统** 
- ✅ **WebSocket连接管理**
- ✅ **Web界面支持**
- ✅ **环境变量配置**
- ✅ **多群组支持**
- ✅ **暗骰功能**
- ✅ **骰子表达式解析器**

---

## 🎲 支持的指令

### COC TRPG指令

| 指令 | 说明 | 示例 |
|------|------|------|
| `.r [骰子表达式]` | 基本掷骰 | `.r 3d6`, `.r 2d10+5` |
| `.rh` | 暗骰（仅群聊） | `.rh` |
| `.coc7` | 生成7版COC调查员属性 | `.coc7` |
| `.sc [成功损失]/[失败损失]` | 理智检定 | `.sc 1/1d6` |
| `.ra [技能值]` | COC检定 | `.ra 70` |
| `.rc [技能值]` | COC7th核心规则检定 | `.rc 65` |
| `.rb [技能值]` | 奖励骰检定 | `.rb 75` |
| `.en [技能值]` | 技能成长检定 | `.en 50` |
| `.ti` | 临时疯狂症状 | `.ti` |
| `.li` | 总结性疯狂症状 | `.li` |
| `.st [技能名] [数值]` | 记录技能属性 | `.st 力量 70 敏捷 65` |
| `.r[理由]` | 带理由的投掷 | `.r 测试投掷` |
| `.set[数字]` | 设置默认骰子面数 | `.set6` |

### DND 5e指令

| 指令 | 说明 | 示例 |
|------|------|------|
| `.dnd [属性]` | 属性检定 | `.dnd str`, `.dnd dex` |
| `.init [敏捷调整值]` | 先攻检定 | `.init 3` |
| `.save [属性]` | 豁免检定 | `.save con`, `.save wis` |
| `.check [技能]` | 技能检定 | `.check stealth`, `.check perception` |
| `.attack [攻击加值]` | 攻击检定 | `.attack 5` |
| `.damage [伤害表达式]` | 伤害骰 | `.damage 2d6+3` |
| `.adv [属性]` | 优势检定 | `.adv str` |
| `.dis [属性]` | 劣势检定 | `.dis dex` |
| `.hp [当前值]/[最大值]` | 生命值管理 | `.hp 25/45` |
| `.spell [法术等级]` | 法术攻击检定 | `.spell 3` |
| `.initiative` | 查看先攻列表 | `.initiative` |
| `.condition [状态]` | 记录状态效果 | `.condition poisoned` |

### 帮助指令
- `.help` - 查看完整的指令帮助

---

## 🚀 快速开始

### 环境要求
- Go 1.23.4 或更高版本
- go-cqhttp (OneBot协议)

### 安装步骤

1. **克隆项目**
   ```bash
   git clone https://github.com/JiYeHuanXiang/island
   cd island
   ```

2. **安装依赖**
   ```bash
   go mod download
   ```

3. **配置环境变量**
   ```bash
   # 可选环境变量
   export HTTP_PORT=8088                    # Web服务器端口
   export QQ_WS_URL=ws://127.0.0.1:3009     # go-cqhttp WebSocket地址
   export QQ_GROUP_ID=123456,789012         # 允许的群组ID（逗号分隔）
   ```

4. **编译运行**
   ```bash
   go build
   ./island
   ```

### 配置go-cqhttp

确保go-cqhttp配置文件中启用了WebSocket正向服务器：

```yaml
# go-cqhttp config.yml
servers:
  - ws:
      host: 127.0.0.1
      port: 3009
```

---

## 🌐 Web界面使用

项目包含一个现代化的模块化Web界面，可以通过浏览器访问：

1. 启动程序后，打开浏览器访问 `http://localhost:8088`
2. 在Web界面中：
   - **系统设置**：统一的设置界面，包含连接设置、管理员设置、群组管理
   - **连接管理**：支持WebSocket、HTTP API、反向WebSocket三种连接模式
   - **骰子指令**：发送COC TRPG和DND 5e骰子指令
   - **实时通信**：通过WebSocket与后端实时交互
   - **主题切换**：支持深色/浅色主题切换
   - **配置同步**：自动保存和加载界面设置

### 前端架构
- **模块化设计**：HTML、CSS、JavaScript分离为独立文件
- **响应式布局**：适配不同屏幕尺寸
- **现代技术**：使用CSS变量、ES6模块、localStorage等现代Web技术
- **代码组织**：清晰的代码结构，便于维护和扩展

---

## ⚙️ 配置说明

### 环境变量

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `HTTP_PORT` | `8088` | Web服务器端口 |
| `QQ_WS_URL` | `ws://127.0.0.1:3009` | go-cqhttp WebSocket地址 |
| `QQ_GROUP_ID` | 空 | 允许的群组ID，多个用逗号分隔 |

### 配置文件

项目使用环境变量进行配置，无需额外的配置文件。

---

## 🎯 使用示例

### COC TRPG示例
```
.ra 75        # 技能值为75的检定
.coc7         # 生成调查员属性
.sc 1/1d6     # 理智检定
.r 3d6+2      # 投掷3个6面骰加2
.rh           # 暗骰（结果私聊发送）
```

### DND 5e示例
```
.dnd str      # 力量属性检定
.init 3       # 先攻检定（敏捷调整值+3）
.attack 5     # 攻击检定（攻击加值+5）
.damage 2d6+3 # 伤害骰
.adv dex      # 敏捷优势检定
```

---

## 🔧 技术架构

### 核心组件

- **main.go**: 主程序入口，包含所有指令处理逻辑
- **parser/**: 骰子表达式解析器
  - `lexer.go`: 词法分析器
  - `parser.go`: 语法分析器
  - `interface.go`: 解析器接口
- **web/**: 模块化Web界面
  - `index.html`: 主HTML文件
  - `server.go`: Web服务器
  - `css/styles.css`: 样式文件
  - `js/app.js`: 主应用逻辑
  - `js/settings.js`: 设置管理
  - `js/utils.js`: 工具函数
- **config/**: 配置管理
  - `config.go`: 配置结构定义
  - `storage.go`: 配置存储
- **connection/**: 连接管理
  - `http.go`: HTTP连接处理
  - `manager.go`: 连接管理器
  - `reverse_ws.go`: 反向WebSocket
  - `types.go`: 连接类型定义
- **handlers/**: 消息处理器
  - `message.go`: 消息处理逻辑

### 依赖库

- `github.com/gorilla/websocket`: WebSocket连接管理
- `github.com/caarlos0/env/v6`: 环境变量解析

---

## 🛠️ 开发说明

### 项目结构
```
island/
├── main.go              # 主程序
├── go.mod              # Go模块文件
├── parser/             # 骰子解析器
│   ├── lexer.go
│   ├── parser.go
│   ├── interface.go
│   └── ...
├── web/                # Web界面模块
│   ├── index.html      # 主HTML文件
│   ├── server.go       # Web服务器
│   ├── css/
│   │   └── styles.css  # 样式文件
│   └── js/             # JavaScript模块
│       ├── app.js      # 主应用逻辑
│       ├── settings.js # 设置管理
│       └── utils.js    # 工具函数
├── config/             # 配置管理
│   ├── config.go
│   └── storage.go
├── connection/         # 连接管理
│   ├── http.go
│   ├── manager.go
│   ├── reverse_ws.go
│   └── types.go
├── handlers/           # 消息处理器
│   └── message.go
```

### 添加新指令

1. 在 `main.go` 中添加正则表达式模式
2. 在 `processCommand` 函数中添加处理逻辑
3. 在帮助信息中更新指令说明

---

## 📝 许可证

本项目采用 BSD 3-Clause License 开源协议。

---

## 🤝 贡献

欢迎提交Issue和Pull Request来改进项目！

### 开发团队
| 角色 | 贡献者 |
|------|--------|
| **主要开发者** | deepseekV3.2，minimax(agent) |
| **辅助开发者** | BegoniaHe, dreamtel(jiyehuanxiang) |
| **校验** | copilot（GPT-4o）, claude 3.7 thinking |

---

## 📞 支持

如有问题，请通过以下方式联系：
- GitHub Issues: [项目Issues页面](https://github.com/JiYeHuanXiang/island/issues)
- QQgroup：684311770

---

**享受你的TRPG游戏！🎲**
