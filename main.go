package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"island/parser"
	"log"
	"math/rand"
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/caarlos0/env/v6"
	"github.com/gorilla/websocket"
)

type Config struct {
	HTTPPort  string  `env:"HTTP_PORT" envDefault:"8088"`
	QQWSURL   string  `env:"QQ_WS_URL" envDefault:"ws://127.0.0.1:3009"`
	QQGroupID []int64 `env:"QQ_GROUP_ID" envSeparator:","` // 修改为切片支持多个群组
}

type ConnectionManager struct {
	conn     *websocket.Conn
	url      string
	mu       sync.RWMutex
	retries  int
	maxRetry int
	quit     chan struct{}
}

type OneBotMessage struct {
	PostType    string          `json:"post_type"`
	MessageType string          `json:"message_type"`
	Message     json.RawMessage `json:"message"` // 修改为RawMessage以便更灵活处理
	UserID      int64           `json:"user_id"`
	GroupID     int64           `json:"group_id"`
	RawMessage  string          `json:"raw_message"`
	SelfID      int64           `json:"self_id"`
}

type ResponseMessage struct {
	Action string      `json:"action"`
	Params interface{} `json:"params"`
}

type CommandRequest struct {
	Command string `json:"command"`
}

type CommandResponse struct {
	Response string `json:"response"`
}

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	webClients    = make(map[*websocket.Conn]struct{})
	webMutex      sync.RWMutex
	cocAttributes = [...]string{"STR", "CON", "SIZ", "DEX", "APP", "INT", "POW", "EDU", "LUK"}
	qqManager     *ConnectionManager
	appConfig     *Config

	rollRegex       = regexp.MustCompile(`^r\s*((\d+)#)?\s*((?:\d*d?\d+[\+\-\*]\d+)+|(?:\d*d\d+(?:[\+\-\*]\d+)*)+|(?:\d+[\+\-\*]\d+)+)$`)
	scRegex         = regexp.MustCompile(`sc\s+(\d+)/(\d+)`)
	raRegex         = regexp.MustCompile(`^ra\s+(\d+)$`)
	rcRegex         = regexp.MustCompile(`^rc\s+(\d+)$`)
	rbRegex         = regexp.MustCompile(`^rb\s+(\d+)$`)
	rhRegex         = regexp.MustCompile(`^rh$`)
	reasonRollRegex = regexp.MustCompile(`^r(d?)\s*(.*)$`)
	setDiceRegex    = regexp.MustCompile(`^set(\d+)$`)
	enRegex         = regexp.MustCompile(`^en\s+(\d+)$`)
	tiRegex         = regexp.MustCompile(`^ti$`)
	liRegex         = regexp.MustCompile(`^li$`)
	stRegex         = regexp.MustCompile(`^st\s+([^\d]+)\s+(\d+)(?:\s+([^\d]+)\s+(\d+))?(?:\s+([^\d]+)\s+(\d+))?(?:\s+([^\d]+)\s+(\d+))?$`)
	coc7Regex       = regexp.MustCompile(`^coc7$`)
	dndStatRegex    = regexp.MustCompile(`^dnd\s+(\w+)$`)
	dndInitRegex    = regexp.MustCompile(`^init\s+(\d+)$`)
	dndSaveRegex    = regexp.MustCompile(`^save\s+(\w+)$`)
	dndCheckRegex   = regexp.MustCompile(`^check\s+(\w+)$`)
	dndAttackRegex  = regexp.MustCompile(`^attack\s+(\d+)$`)
	dndDamageRegex  = regexp.MustCompile(`^damage\s+(.+)$`)
	dndAdvRegex     = regexp.MustCompile(`^adv\s+(\w+)$`)
	dndDisRegex     = regexp.MustCompile(`^dis\s+(\w+)$`)
	dndHPRegex      = regexp.MustCompile(`^hp\s+(\d+)/(\d+)$`)
	dndSpellRegex   = regexp.MustCompile(`^spell\s+(\d+)$`)
	dndInitiativeRegex = regexp.MustCompile(`^initiative$`)
	dndConditionRegex = regexp.MustCompile(`^condition\s+(.+)$`)

	defaultDiceSides = 100
	diceMutex        sync.RWMutex

	ErrMaxRetries  = errors.New("maximum retry attempts reached")
	ErrInvalidMsg  = errors.New("invalid message format")
	ErrConnClosed  = errors.New("connection closed")
	ErrConfigLoad  = errors.New("configuration load failed")
	ErrInvalidDice = errors.New("invalid dice sides")
)

func NewConnectionManager(url string, maxRetry int) *ConnectionManager {
	return &ConnectionManager{
		url:      url,
		maxRetry: maxRetry,
		quit:     make(chan struct{}),
	}
}

func (cm *ConnectionManager) Connect() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.conn != nil {
		return nil
	}

	var err error
	for i := 0; i < cm.maxRetry; i++ {
		cm.conn, _, err = websocket.DefaultDialer.Dial(cm.url, nil)
		if err == nil {
			cm.retries = 0
			return nil
		}

		select {
		case <-cm.quit:
			return ErrConnClosed
		default:
			waitTime := time.Duration(i+1) * time.Second
			log.Printf("连接失败 (尝试 %d/%d), %v秒后重试...", i+1, cm.maxRetry, waitTime)
			time.Sleep(waitTime)
		}
	}
	return fmt.Errorf("%w: %v", ErrMaxRetries, err)
}

func (cm *ConnectionManager) Get() (*websocket.Conn, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if cm.conn == nil {
		return nil, ErrConnClosed
	}
	return cm.conn, nil
}

func (cm *ConnectionManager) Close() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	close(cm.quit)
	if cm.conn != nil {
		cm.conn.Close()
		cm.conn = nil
	}
}

func loadConfig() (*Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConfigLoad, err)
	}

	if len(cfg.QQGroupID) == 0 {
		log.Println("未配置QQ_GROUP_ID，将处理所有群组消息")
	}
	return &cfg, nil
}

func main() {
	rand.Seed(time.Now().UnixNano())

	var err error
	appConfig, err = loadConfig()
	if err != nil {
		log.Printf("配置加载错误: %v", err)
		log.Println("使用默认配置继续运行...")
		appConfig = &Config{
			HTTPPort:  "8088",
			QQWSURL:   "ws://127.0.0.1:3009",
			QQGroupID: []int64{},
		}
	}

	qqManager = NewConnectionManager(appConfig.QQWSURL, 5)
	defer qqManager.Close()

	if err := qqManager.Connect(); err != nil {
		log.Printf("初始化QQ连接失败: %v", err)
		log.Println("将在消息处理时尝试重新连接...")
	}

	go startHTTPServer()

	messageLoop()
}

func messageLoop() {
	for {
		conn, err := qqManager.Get()
		if err != nil {
			if err := qqManager.Connect(); err != nil {
				log.Printf("QQ连接不可用: %v", err)
				time.Sleep(5 * time.Second)
				continue
			}
			conn, _ = qqManager.Get()
		}

		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("读取消息错误: %v", err)
			qqManager.mu.Lock()
			qqManager.conn = nil
			qqManager.mu.Unlock()
			time.Sleep(1 * time.Second)
			continue
		}

		msg, err := parseIncomingMessage(message)
		if err != nil {
			log.Printf("消息解析错误: %v", err)
			continue
		}

		if msg.PostType == "message" {
			handleMessage(conn, msg)
		}
	}
}

func parseIncomingMessage(data []byte) (*OneBotMessage, error) {
	var msg OneBotMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidMsg, err)
	}
	return &msg, nil
}

func startHTTPServer() {
	http.HandleFunc("/", serveStatic)
	http.HandleFunc("/ws", handleWebSocket)
	http.HandleFunc("/command", handleCommand)
	log.Printf("Web服务器已启动 :%s", appConfig.HTTPPort)
	if err := http.ListenAndServe(":"+appConfig.HTTPPort, nil); err != nil {
		log.Fatalf("HTTP服务器错误: %v", err)
	}
}

func serveStatic(w http.ResponseWriter, r *http.Request) {
	path := filepath.Clean(r.URL.Path)
	if path == "/" || path == "/index.html" {
		http.ServeFile(w, r, "web/UI.html")
		return
	}

	if strings.HasPrefix(path, "/") {
		path = filepath.Join("web", path)
	}
	http.ServeFile(w, r, path)
}

func handleCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response := processCommand(req.Command)
	json.NewEncoder(w).Encode(CommandResponse{Response: response})
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket升级失败: %v", err)
		return
	}

	webMutex.Lock()
	webClients[conn] = struct{}{}
	webMutex.Unlock()

	go func() {
		defer func() {
			webMutex.Lock()
			delete(webClients, conn)
			webMutex.Unlock()
			conn.Close()
		}()

		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				if !websocket.IsCloseError(err, websocket.CloseNormalClosure) {
					log.Printf("WebSocket读取错误: %v", err)
				}
				break
			}

			// 处理从Web界面收到的消息
			response := processCommand(string(message))
			if err := conn.WriteMessage(websocket.TextMessage, []byte(response)); err != nil {
				log.Printf("WebSocket写入错误: %v", err)
				break
			}

			// 同时发送到QQ
			if err := sendToQQ(response); err != nil {
				log.Printf("发送到QQ失败: %v", err)
			}
		}
	}()
}

func handleMessage(conn *websocket.Conn, msg *OneBotMessage) {
	messageStr, err := extractMessageContent(msg.Message)
	if err != nil {
		log.Printf("消息内容提取失败: %v", err)
		return
	}

	if !strings.HasPrefix(messageStr, ".") {
		return
	}

	response := processCommand(messageStr)
	if err := sendResponse(conn, msg, response); err != nil {
		log.Printf("发送响应失败: %v", err)
		return
	}

	broadcastToWeb(formatWebMessage(msg, response))
}

func extractMessageContent(msg json.RawMessage) (string, error) {
	var messageSegments []struct {
		Type string `json:"type"`
		Data struct {
			Text string `json:"text"`
		} `json:"data"`
	}

	if err := json.Unmarshal(msg, &messageSegments); err != nil {
		// 尝试解析为纯字符串
		var text string
		if err := json.Unmarshal(msg, &text); err != nil {
			return "", fmt.Errorf("%w: %v", ErrInvalidMsg, err)
		}
		return text, nil
	}

	var builder strings.Builder
	for _, seg := range messageSegments {
		if seg.Type == "text" {
			builder.WriteString(seg.Data.Text)
		}
	}
	return builder.String(), nil
}

func formatWebMessage(msg *OneBotMessage, response string) string {
	if msg.MessageType == "group" {
		return fmt.Sprintf("[群消息] %s", response)
	}
	return fmt.Sprintf("[私聊] %s", response)
}

func processCommand(cmd string) string {
	cmd = strings.TrimPrefix(cmd, ".")
	switch {
	case rollRegex.MatchString(cmd):
		return processRoll(cmd)
	case scRegex.MatchString(cmd):
		return processSanCheck(cmd)
	case cmd == "coc7":
		return processCoC7()
	case raRegex.MatchString(cmd):
		return processRACheck(cmd)
	case rcRegex.MatchString(cmd):
		return processRCCheck(cmd)
	case rbRegex.MatchString(cmd):
		return processRBCheck(cmd)
	case rhRegex.MatchString(cmd):
		return "rh" // Special case handled in handleMessage
	case reasonRollRegex.MatchString(cmd):
		return processReasonRoll(cmd)
	case setDiceRegex.MatchString(cmd):
		return processSetDice(cmd)
	case enRegex.MatchString(cmd):
		return processEnCheck(cmd)
	case tiRegex.MatchString(cmd):
		return processTICheck()
	case liRegex.MatchString(cmd):
		return processLICheck()
	case stRegex.MatchString(cmd):
		return processStCheck(cmd)
	case dndStatRegex.MatchString(cmd):
		return processDNDStat(cmd)
	case dndInitRegex.MatchString(cmd):
		return processDNDInit(cmd)
	case dndSaveRegex.MatchString(cmd):
		return processDNDSave(cmd)
	case dndCheckRegex.MatchString(cmd):
		return processDNDCheck(cmd)
	case dndAttackRegex.MatchString(cmd):
		return processDNDAttack(cmd)
	case dndDamageRegex.MatchString(cmd):
		return processDNDDamage(cmd)
	case dndAdvRegex.MatchString(cmd):
		return processDNDAdvantage(cmd)
	case dndDisRegex.MatchString(cmd):
		return processDNDDisadvantage(cmd)
	case dndHPRegex.MatchString(cmd):
		return processDNDHP(cmd)
	case dndSpellRegex.MatchString(cmd):
		return processDNDSpell(cmd)
	case dndInitiativeRegex.MatchString(cmd):
		return processDNDInitiative()
	case dndConditionRegex.MatchString(cmd):
		return processDNDCondition(cmd)
	case cmd == "help":
		return "TRPG指令帮助:\n" +
			"=== COC指令 ===\n" +
			".r[骰子指令] 掷骰\n" +
			".rh 暗骰(仅群聊)\n" +
			".coc7 生成调查员(7版规则)\n" +
			".sc [成功损失]/[失败损失] 理智检定\n" +
			".ra [技能值] COCTRPG检定\n" +
			".rc [技能值] COC7th核心规则检定\n" +
			".rb [技能值] 奖励骰检定\n" +
			".en [技能值] 技能成长检定\n" +
			".ti 临时疯狂症状\n" +
			".li 总结性疯狂症状\n" +
			".st [技能名] [数值] 记录技能属性(可多个)\n" +
			".r[理由] 带理由的投掷\n" +
			".set[数字] 设置默认骰子面数(如.set6)\n\n" +
			"=== DND指令 ===\n" +
			".dnd [属性] 属性检定(如.dnd str)\n" +
			".init [敏捷调整值] 先攻检定\n" +
			".save [属性] 豁免检定(如.save con)\n" +
			".check [技能] 技能检定(如.check stealth)\n" +
			".attack [攻击加值] 攻击检定\n" +
			".damage [伤害表达式] 伤害骰(如.damage 2d6+3)\n" +
			".adv [属性] 优势检定\n" +
			".dis [属性] 劣势检定\n" +
			".hp [当前值]/[最大值] 记录生命值\n" +
			".spell [法术等级] 法术攻击检定\n" +
			".initiative 查看先攻列表\n" +
			".condition [状态] 记录状态效果\n" +
			"可用属性: str, dex, con, int, wis, cha"
	default:
		return "未知指令，请输入.help查看帮助"
	}
}

func processRoll(cmd string) string {
	matches := rollRegex.FindStringSubmatch(cmd)
	if len(matches) < 4 {
		return "无效的骰子指令格式，正确格式：.r [骰子表达式]"
	}

	rounds := 1
	if matches[2] != "" {
		var err error
		rounds, err = strconv.Atoi(matches[2])
		if err != nil || rounds < 1 || rounds > 10 {
			return "轮次必须为1-10的整数"
		}
	}
	expression := strings.TrimSpace(matches[3])

	var results []string
	for i := 0; i < rounds; i++ {
		result, err := evaluateRollExpression(expression)
		if err != nil {
			return err.Error()
		}
		results = append(results, result)
	}
	return fmt.Sprintf("骰子结果: %s", strings.Join(results, "\n"))
}

func evaluateRollExpression(expression string) (string, error) {
	if res := ParseAndEvaluate(expression)["过程"].(string); res != "" {
		return res, nil
	} else {
		return "", fmt.Errorf("骰子表达式解析失败")
	}
}

func ParseAndEvaluate(input string) map[string]interface{} {
	// 设置词法分析器
	l := parser.NewLexer(input)

	// 创建一个包装器，确保它实现 parser.yyLexer 接口
	lexerWrapper := parser.NewLexerWrapper(l)

	// 使用包装函数
	result := parser.Parse(lexerWrapper)

	// 检查解析状态，0 表示成功
	if result == 0 {
		// 解析成功，获取计算结果
		return parser.GetResult()
	}

	// 解析失败
	return nil
}

func processSetDice(cmd string) string {
	matches := setDiceRegex.FindStringSubmatch(cmd)
	if len(matches) < 2 {
		return "无效的设置指令格式，正确格式：.set[数字]，例如.set6"
	}

	sides, err := strconv.Atoi(matches[1])
	if err != nil || sides <= 0 {
		return "骰子面数必须是正整数"
	}

	diceMutex.Lock()
	defaultDiceSides = sides
	diceMutex.Unlock()

	return fmt.Sprintf("已设置默认骰子面数为D%d", sides)
}

func processReasonRoll(cmd string) string {
	matches := reasonRollRegex.FindStringSubmatch(cmd)
	if len(matches) < 3 {
		return "无效的投掷指令格式"
	}

	reason := strings.TrimSpace(matches[2])
	if reason == "" {
		return "请输入投掷理由，例如：.r 测试"
	}

	diceMutex.RLock()
	sides := defaultDiceSides
	diceMutex.RUnlock()

	roll := rand.Intn(sides) + 1
	return fmt.Sprintf("因为 %s 1D%d=%d", reason, sides, roll)
}

func processRACheck(cmd string) string {
	matches := raRegex.FindStringSubmatch(cmd)
	if len(matches) < 2 {
		return "无效的ra指令格式，正确格式：.ra 技能值"
	}

	skillValue, err := strconv.Atoi(matches[1])
	if err != nil || skillValue < 1 || skillValue > 100 {
		return "技能值必须为1-100的整数"
	}

	roll := rand.Intn(100) + 1
	result := fmt.Sprintf("检定ra %d → %d", skillValue, roll)

	if roll <= skillValue {
		if roll <= 5 {
			result += " 大成功！"
		} else {
			result += " 成功"
		}
	} else {
		if roll >= 96 {
			result += " 大失败！"
		} else {
			result += " 失败"
		}
	}
	return result
}

func processRBCheck(cmd string) string {
	matches := rbRegex.FindStringSubmatch(cmd)
	if len(matches) < 2 {
		return "无效的rb指令格式，正确格式：.rb 技能值"
	}

	skillValue, err := strconv.Atoi(matches[1])
	if err != nil || skillValue < 1 || skillValue > 100 {
		return "技能值必须为1-100的整数"
	}

	bonusDice := rand.Intn(10) * 10
	roll := rand.Intn(100) + 1
	effectiveRoll := roll
	if bonusDice < roll {
		effectiveRoll = bonusDice
	}

	result := fmt.Sprintf("奖励骰检定rb %d → 原始值:%d 奖励骰:%d 最终值:%d", skillValue, roll, bonusDice, effectiveRoll)

	if effectiveRoll <= skillValue {
		if effectiveRoll <= 5 {
			result += " 大成功！"
		} else {
			result += " 成功"
		}
	} else {
		if effectiveRoll >= 96 {
			result += " 大失败！"
		} else {
			result += " 失败"
		}
	}
	return result
}

func processRCCheck(cmd string) string {
	matches := rcRegex.FindStringSubmatch(cmd)
	if len(matches) < 2 {
		return "无效的rc指令格式，正确格式：.rc 技能值"
	}

	skillValue, err := strconv.Atoi(matches[1])
	if err != nil || skillValue < 1 || skillValue > 100 {
		return "技能值必须为1-100的整数"
	}

	roll := rand.Intn(100) + 1
	result := fmt.Sprintf("检定rc %d → %d", skillValue, roll)

	success := roll <= skillValue
	var critical bool
	if skillValue <= 50 {
		critical = roll <= 5
	} else {
		critical = roll <= skillValue/5
	}
	critical = critical && roll <= 95
	extreme := success && (roll <= skillValue/5)
	hard := success && (roll <= skillValue/2)
	fumble := roll == 100 || (roll >= 96 && !success)

	switch {
	case success && critical:
		result += " 大成功！"
	case success && extreme:
		result += " 极难成功"
	case success && hard:
		result += " 困难成功"
	case success:
		result += " 成功"
	case fumble:
		result += " 大失败！"
	default:
		result += " 失败"
	}
	return result
}

func processEnCheck(cmd string) string {
	matches := enRegex.FindStringSubmatch(cmd)
	if len(matches) < 2 {
		return "无效的en指令格式，正确格式：.en 技能值"
	}

	skillValue, err := strconv.Atoi(matches[1])
	if err != nil || skillValue < 1 || skillValue > 100 {
		return "技能值必须为1-100的整数"
	}

	roll := rand.Intn(100) + 1
	if roll > skillValue {
		increase := rand.Intn(10) + 1
		newSkill := skillValue + increase
		if newSkill > 100 {
			newSkill = 100
		}
		return fmt.Sprintf("技能成长检定en %d → %d/%d 成功！技能提升%d点，新值:%d", skillValue, roll, skillValue, increase, newSkill)
	}
	return fmt.Sprintf("技能成长检定en %d → %d/%d 失败", skillValue, roll, skillValue)
}

func processTICheck() string {
	symptoms := []string{
		"1. 失忆: 调查员会发现自己只记得最后身处的安全地点，却没有任何来到这里的记忆。",
		"2. 假性残疾: 调查员陷入了心理性的失明，失聪以及躯体缺失感中。",
		"3. 暴力倾向: 调查员陷入了六亲不认的暴力行为中。",
		"4. 偏执: 调查员陷入了严重的偏执妄想之中。",
		"5. 人际依赖: 调查员因为一些原因而将他人误认为了他重要的人。",
		"6. 昏厥: 调查员当场昏倒。",
		"7. 逃避行为: 调查员会用任何的手段试图逃离现在所处的位置。",
		"8. 竭嘶底里: 调查员表现出大笑，哭泣，嘶吼，害怕等的极端情绪表现。",
		"9. 恐惧: 调查员投一个D100或者由守秘人选择，来从恐惧症状表中选择一个恐惧源。",
		"10. 狂躁: 调查员投一个D100或者由守秘人选择，来从狂躁症状表中选择一个狂躁的表现。",
	}
	roll := rand.Intn(10)
	return "临时疯狂症状:\n" + symptoms[roll]
}

func processLICheck() string {
	symptoms := []string{
		"1. 失忆: 回过神来，调查员们发现自己身处一个陌生的地方，并忘记了自己是谁。",
		"2. 被窃: 调查员在1D10小时后恢复清醒，发觉自己被盗，身体毫发无损。",
		"3. 遍体鳞伤: 调查员在1D10小时后恢复清醒，发现自己身上满是拳痕和瘀伤。",
		"4. 暴力倾向: 调查员陷入强烈的暴力与破坏欲之中。",
		"5. 极端信念: 调查员在1D10小时后恢复清醒，遵循着某个启示。",
		"6. 重要之人: 调查员在1D10小时后恢复清醒，决定与某人或某物建立深厚的联系。",
		"7. 被收容: 调查员在1D10小时后恢复清醒，发现自己在监狱或精神病院。",
		"8. 逃避行为: 调查员恢复清醒时发现自己在陌生的地方，可能无意识旅行了很远。",
		"9. 恐惧: 投一个D100或者由守秘人选择，来从恐惧症状表中选择一个恐惧源。",
		"10. 狂躁: 投一个D100或者由守秘人选择，来从狂躁症状表中选择一个狂躁的表现。",
	}
	roll := rand.Intn(10)
	return "总结性疯狂症状:\n" + symptoms[roll]
}

func processStCheck(cmd string) string {
	matches := stRegex.FindStringSubmatch(cmd)
	if len(matches) < 3 {
		return "无效的st指令格式，正确格式：.st [技能名] [数值] (可多个)"
	}

	var result strings.Builder
	result.WriteString("记录属性:\n")

	for i := 1; i < len(matches); i += 2 {
		if i+1 >= len(matches) {
			break
		}

		skillName := strings.TrimSpace(matches[i])
		skillValue := strings.TrimSpace(matches[i+1])

		if skillName == "" || skillValue == "" {
			continue
		}

		value, err := strconv.Atoi(skillValue)
		if err != nil {
			result.WriteString(fmt.Sprintf("无效数值: %s = %s\n", skillName, skillValue))
			continue
		}

		result.WriteString(fmt.Sprintf("%s = %d\n", skillName, value))
	}

	return result.String()
}

func processCoC7() string {
	var attributes []string
	for _, attr := range cocAttributes {
		var value int
		switch attr {
		case "STR", "CON":
			value = rand.Intn(6)*5 + 30
		case "SIZ":
			value = rand.Intn(6) + rand.Intn(6) + 6
		case "DEX":
			value = rand.Intn(6)*5 + 30
		case "APP":
			value = rand.Intn(6)*5 + 30
		case "INT":
			value = rand.Intn(6)*5 + 50
		case "POW":
			value = rand.Intn(6)*5 + 30
		case "EDU":
			value = rand.Intn(6)*5 + 30
		case "LUK":
			value = rand.Intn(6)*5 + 30
		}
		attributes = append(attributes, fmt.Sprintf("%s: %d", attr, value))
	}
	return "调查员属性(7版规则):\n" + strings.Join(attributes, "\n")
}

func processSanCheck(cmd string) string {
	matches := scRegex.FindStringSubmatch(cmd)
	if len(matches) < 3 {
		return "理智检定格式错误，正确格式：.sc 成功损失/失败损失"
	}
	successLoss, _ := strconv.Atoi(matches[1])
	failLoss, _ := strconv.Atoi(matches[2])

	roll := rand.Intn(100) + 1
	result := fmt.Sprintf("理智检定: %d/%d 掷骰: %d", successLoss, failLoss, roll)
	return result
}

// DND相关指令处理函数
func processDNDStat(cmd string) string {
	matches := dndStatRegex.FindStringSubmatch(cmd)
	if len(matches) < 2 {
		return "无效的dnd指令格式，正确格式：.dnd [属性]"
	}

	attr := strings.ToLower(matches[1])
	validAttrs := map[string]bool{
		"str": true, "dex": true, "con": true,
		"int": true, "wis": true, "cha": true,
	}

	if !validAttrs[attr] {
		return "无效的属性，可用属性: str, dex, con, int, wis, cha"
	}

	modifier := rand.Intn(20) + 1
	return fmt.Sprintf("DND %s属性检定: d20=%d", strings.ToUpper(attr), modifier)
}

func processDNDInit(cmd string) string {
	matches := dndInitRegex.FindStringSubmatch(cmd)
	if len(matches) < 2 {
		return "无效的init指令格式，正确格式：.init [敏捷调整值]"
	}

	dexMod, err := strconv.Atoi(matches[1])
	if err != nil {
		return "敏捷调整值必须是整数"
	}

	roll := rand.Intn(20) + 1
	total := roll + dexMod
	
	// 生成一个简单的角色名（在实际使用中，用户应该提供角色名）
	characterName := fmt.Sprintf("角色%d", len(initiativeList)+1)
	
	initiativeMutex.Lock()
	initiativeList[characterName] = total
	initiativeMutex.Unlock()
	
	return fmt.Sprintf("先攻检定: d20(%d) + 敏捷调整(%d) = %d\n已添加 %s 到先攻列表", roll, dexMod, total, characterName)
}

func processDNDSave(cmd string) string {
	matches := dndSaveRegex.FindStringSubmatch(cmd)
	if len(matches) < 2 {
		return "无效的save指令格式，正确格式：.save [属性]"
	}

	attr := strings.ToLower(matches[1])
	validAttrs := map[string]bool{
		"str": true, "dex": true, "con": true,
		"int": true, "wis": true, "cha": true,
	}

	if !validAttrs[attr] {
		return "无效的属性，可用属性: str, dex, con, int, wis, cha"
	}

	roll := rand.Intn(20) + 1
	return fmt.Sprintf("DND %s豁免检定: d20=%d", strings.ToUpper(attr), roll)
}

func processDNDCheck(cmd string) string {
	matches := dndCheckRegex.FindStringSubmatch(cmd)
	if len(matches) < 2 {
		return "无效的check指令格式，正确格式：.check [技能]"
	}

	skill := strings.ToLower(matches[1])
	validSkills := map[string]bool{
		"acrobatics": true, "animal": true, "arcana": true,
		"athletics": true, "deception": true, "history": true,
		"insight": true, "intimidation": true, "investigation": true,
		"medicine": true, "nature": true, "perception": true,
		"performance": true, "persuasion": true, "religion": true,
		"sleight": true, "stealth": true, "survival": true,
	}

	if !validSkills[skill] {
		return "无效的技能，可用技能: acrobatics, animal, arcana, athletics, deception, history, insight, intimidation, investigation, medicine, nature, perception, performance, persuasion, religion, sleight, stealth, survival"
	}

	roll := rand.Intn(20) + 1
	return fmt.Sprintf("DND %s技能检定: d20=%d", strings.Title(skill), roll)
}

// DND攻击检定
func processDNDAttack(cmd string) string {
	matches := dndAttackRegex.FindStringSubmatch(cmd)
	if len(matches) < 2 {
		return "无效的attack指令格式，正确格式：.attack [攻击加值]"
	}

	attackBonus, err := strconv.Atoi(matches[1])
	if err != nil {
		return "攻击加值必须是整数"
	}

	roll := rand.Intn(20) + 1
	total := roll + attackBonus
	
	var result string
	if roll == 20 {
		result = " 暴击！"
	} else if roll == 1 {
		result = " 失手！"
	} else {
		result = ""
	}
	
	return fmt.Sprintf("攻击检定: d20(%d) + 攻击加值(%d) = %d%s", roll, attackBonus, total, result)
}

// DND伤害骰
func processDNDDamage(cmd string) string {
	matches := dndDamageRegex.FindStringSubmatch(cmd)
	if len(matches) < 2 {
		return "无效的damage指令格式，正确格式：.damage [伤害表达式]，例如.damage 2d6+3"
	}

	damageExpr := strings.TrimSpace(matches[1])
	result, err := evaluateRollExpression(damageExpr)
	if err != nil {
		return fmt.Sprintf("伤害骰解析失败: %v", err)
	}

	return fmt.Sprintf("伤害骰: %s", result)
}

// DND优势检定
func processDNDAdvantage(cmd string) string {
	matches := dndAdvRegex.FindStringSubmatch(cmd)
	if len(matches) < 2 {
		return "无效的adv指令格式，正确格式：.adv [属性]"
	}

	attr := strings.ToLower(matches[1])
	validAttrs := map[string]bool{
		"str": true, "dex": true, "con": true,
		"int": true, "wis": true, "cha": true,
	}

	if !validAttrs[attr] {
		return "无效的属性，可用属性: str, dex, con, int, wis, cha"
	}

	roll1 := rand.Intn(20) + 1
	roll2 := rand.Intn(20) + 1
	finalRoll := roll1
	if roll2 > roll1 {
		finalRoll = roll2
	}

	return fmt.Sprintf("DND %s优势检定: d20=%d (投掷: %d, %d)", strings.ToUpper(attr), finalRoll, roll1, roll2)
}

// DND劣势检定
func processDNDDisadvantage(cmd string) string {
	matches := dndDisRegex.FindStringSubmatch(cmd)
	if len(matches) < 2 {
		return "无效的dis指令格式，正确格式：.dis [属性]"
	}

	attr := strings.ToLower(matches[1])
	validAttrs := map[string]bool{
		"str": true, "dex": true, "con": true,
		"int": true, "wis": true, "cha": true,
	}

	if !validAttrs[attr] {
		return "无效的属性，可用属性: str, dex, con, int, wis, cha"
	}

	roll1 := rand.Intn(20) + 1
	roll2 := rand.Intn(20) + 1
	finalRoll := roll1
	if roll2 < roll1 {
		finalRoll = roll2
	}

	return fmt.Sprintf("DND %s劣势检定: d20=%d (投掷: %d, %d)", strings.ToUpper(attr), finalRoll, roll1, roll2)
}

// DND生命值管理
func processDNDHP(cmd string) string {
	matches := dndHPRegex.FindStringSubmatch(cmd)
	if len(matches) < 3 {
		return "无效的hp指令格式，正确格式：.hp [当前值]/[最大值]"
	}

	current, err1 := strconv.Atoi(matches[1])
	max, err2 := strconv.Atoi(matches[2])
	if err1 != nil || err2 != nil {
		return "生命值必须是整数"
	}

	if current < 0 {
		current = 0
	}
	if max < 1 {
		return "最大生命值必须大于0"
	}
	if current > max {
		current = max
	}

	percentage := (current * 100) / max
	var status string
	if current == 0 {
		status = " (濒死)"
	} else if percentage <= 25 {
		status = " (重伤)"
	} else if percentage <= 50 {
		status = " (轻伤)"
	}

	return fmt.Sprintf("生命值: %d/%d (%d%%)%s", current, max, percentage, status)
}

// DND法术攻击检定
func processDNDSpell(cmd string) string {
	matches := dndSpellRegex.FindStringSubmatch(cmd)
	if len(matches) < 2 {
		return "无效的spell指令格式，正确格式：.spell [法术等级]"
	}

	spellLevel, err := strconv.Atoi(matches[1])
	if err != nil || spellLevel < 0 || spellLevel > 9 {
		return "法术等级必须是0-9的整数"
	}

	roll := rand.Intn(20) + 1
	spellAttackBonus := 5 + spellLevel // 假设基础攻击加值为5
	total := roll + spellAttackBonus

	var result string
	if roll == 20 {
		result = " 暴击！"
	} else if roll == 1 {
		result = " 失手！"
	} else {
		result = ""
	}

	return fmt.Sprintf("法术攻击检定(等级%d): d20(%d) + 法术攻击加值(%d) = %d%s", spellLevel, roll, spellAttackBonus, total, result)
}

// DND先攻列表管理
var initiativeList = make(map[string]int)
var initiativeMutex sync.RWMutex

func processDNDInitiative() string {
	initiativeMutex.RLock()
	defer initiativeMutex.RUnlock()

	if len(initiativeList) == 0 {
		return "先攻列表为空，使用.init [敏捷调整值] 添加角色到先攻列表"
	}

	var entries []struct {
		name string
		init int
	}
	for name, init := range initiativeList {
		entries = append(entries, struct {
			name string
			init int
		}{name, init})
	}

	// 按先攻值排序
	for i := 0; i < len(entries)-1; i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[j].init > entries[i].init {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}

	var result strings.Builder
	result.WriteString("先攻列表:\n")
	for i, entry := range entries {
		result.WriteString(fmt.Sprintf("%d. %s: %d\n", i+1, entry.name, entry.init))
	}

	return result.String()
}

// DND状态效果跟踪
var conditionList = make(map[string][]string)
var conditionMutex sync.RWMutex

func processDNDCondition(cmd string) string {
	matches := dndConditionRegex.FindStringSubmatch(cmd)
	if len(matches) < 2 {
		return "无效的condition指令格式，正确格式：.condition [状态]，例如.condition poisoned"
	}

	condition := strings.TrimSpace(matches[1])
	
	// 模拟添加状态到当前角色
	conditionMutex.Lock()
	conditionList["current"] = append(conditionList["current"], condition)
	conditionMutex.Unlock()

	conditionMutex.RLock()
	currentConditions := conditionList["current"]
	conditionMutex.RUnlock()

	return fmt.Sprintf("已添加状态: %s\n当前状态: %s", condition, strings.Join(currentConditions, ", "))
}

func broadcastToWeb(message string) {
	webMutex.RLock()
	defer webMutex.RUnlock()

	for client := range webClients {
		client := client
		go func() {
			if err := client.WriteMessage(websocket.TextMessage, []byte(message)); err != nil {
				log.Printf("广播消息失败: %v", err)
				webMutex.Lock()
				delete(webClients, client)
				client.Close()
				webMutex.Unlock()
			}
		}()
	}
}

func sendToQQ(message string) error {
	conn, err := qqManager.Get()
	if err != nil {
		if err := qqManager.Connect(); err != nil {
			return fmt.Errorf("连接失败: %w", err)
		}
		conn, _ = qqManager.Get()
	}

	resp := ResponseMessage{
		Action: "send_group_msg",
		Params: map[string]interface{}{
			"group_id": appConfig.QQGroupID,
			"message":  message,
		},
	}

	if err := conn.WriteJSON(resp); err != nil {
		qqManager.mu.Lock()
		qqManager.conn = nil
		qqManager.mu.Unlock()
		return fmt.Errorf("写入失败: %w", err)
	}
	return nil
}

func sendResponse(conn *websocket.Conn, msg *OneBotMessage, response string) error {
	// 处理暗骰指令
	if strings.TrimPrefix(msg.RawMessage, ".") == "rh" && msg.MessageType == "group" {
		groupMsg := ResponseMessage{
			Action: "send_group_msg",
			Params: map[string]interface{}{
				"group_id": msg.GroupID,
				"message":  "事情似乎发生了什么变化",
			},
		}
		if err := conn.WriteJSON(groupMsg); err != nil {
			return fmt.Errorf("发送群消息失败: %w", err)
		}

		roll := rand.Intn(100) + 1
		privateMsg := ResponseMessage{
			Action: "send_private_msg",
			Params: map[string]interface{}{
				"user_id": msg.UserID,
				"message": fmt.Sprintf("在群 %d 中的暗骰结果是1D100=%d", msg.GroupID, roll),
			},
		}
		return conn.WriteJSON(privateMsg)
	}

	resp := ResponseMessage{
		Action: "send_msg",
		Params: map[string]interface{}{
			"message_type": msg.MessageType,
			"user_id":      msg.UserID,
			"group_id":     msg.GroupID,
			"message":      response,
		},
	}

	return conn.WriteJSON(resp)
}
