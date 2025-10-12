package handlers

import (
	"encoding/json"
	"fmt"
	"island/config"
	"island/connection"
	"island/parser"
	"log"
	"math"
	"math/rand"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// MessageHandler 处理OneBot V11协议消息
type MessageHandler struct {
	connManager *connection.ConnectionManager
	config      *config.Config
}

// OneBotMessage OneBot V11协议消息结构
type OneBotMessage struct {
	PostType    string          `json:"post_type"`
	MessageType string          `json:"message_type"`
	Message     json.RawMessage `json:"message"`
	UserID      int64           `json:"user_id"`
	GroupID     int64           `json:"group_id"`
	RawMessage  string          `json:"raw_message"`
	SelfID      int64           `json:"self_id"`
}

// ResponseMessage 响应消息结构
type ResponseMessage struct {
	Action string      `json:"action"`
	Params interface{} `json:"params"`
}


var (
	cocAttributes = [...]string{"STR", "CON", "SIZ", "DEX", "APP", "INT", "POW", "EDU", "LUK"}
	defaultDiceSides = 100
	diceMutex        sync.RWMutex

	// 正则表达式
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

	// DND相关变量
	initiativeList = make(map[string]int)
	initiativeMutex sync.RWMutex
	conditionList = make(map[string][]string)
	conditionMutex sync.RWMutex
)

// NewMessageHandler 创建新的消息处理器
func NewMessageHandler(connManager *connection.ConnectionManager, config *config.Config) *MessageHandler {
	return &MessageHandler{
		connManager: connManager,
		config:      config,
	}
}

// StartWebSocketMessageLoop 启动WebSocket消息循环
func (h *MessageHandler) StartWebSocketMessageLoop() {
	for {
		message, err := h.connManager.ReceiveMessage()
		if err != nil {
			log.Printf("接收消息错误: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		msg, err := parseIncomingMessage(message)
		if err != nil {
			log.Printf("消息解析错误: %v", err)
			continue
		}

		if msg.PostType == "message" {
			h.HandleOneBotMessage(msg)
		}
	}
}

// HandleOneBotMessage 处理OneBot消息
func (h *MessageHandler) HandleOneBotMessage(msg *OneBotMessage) {
	messageStr, err := extractMessageContent(msg.Message)
	if err != nil {
		log.Printf("消息内容提取失败: %v", err)
		return
	}

	if !strings.HasPrefix(messageStr, ".") {
		return
	}

	response := h.ProcessCommand(messageStr)
	if err := h.sendResponse(msg, response); err != nil {
		log.Printf("发送响应失败: %v", err)
		return
	}
}

// ProcessCommand 处理命令
func (h *MessageHandler) ProcessCommand(cmd string) string {
	cmd = strings.TrimPrefix(cmd, ".")
	switch {
	case rollRegex.MatchString(cmd):
		return h.processRoll(cmd)
	case scRegex.MatchString(cmd):
		return h.processSanCheck(cmd)
	case cmd == "coc7":
		return h.processCoC7()
	case raRegex.MatchString(cmd):
		return h.processRACheck(cmd)
	case rcRegex.MatchString(cmd):
		return h.processRCCheck(cmd)
	case rbRegex.MatchString(cmd):
		return h.processRBCheck(cmd)
	case rhRegex.MatchString(cmd):
		return "rh" // Special case handled in handleMessage
	case reasonRollRegex.MatchString(cmd):
		return h.processReasonRoll(cmd)
	case setDiceRegex.MatchString(cmd):
		return h.processSetDice(cmd)
	case enRegex.MatchString(cmd):
		return h.processEnCheck(cmd)
	case tiRegex.MatchString(cmd):
		return h.processTICheck()
	case liRegex.MatchString(cmd):
		return h.processLICheck()
	case stRegex.MatchString(cmd):
		return h.processStCheck(cmd)
	case dndStatRegex.MatchString(cmd):
		return h.processDNDStat(cmd)
	case dndInitRegex.MatchString(cmd):
		return h.processDNDInit(cmd)
	case dndSaveRegex.MatchString(cmd):
		return h.processDNDSave(cmd)
	case dndCheckRegex.MatchString(cmd):
		return h.processDNDCheck(cmd)
	case dndAttackRegex.MatchString(cmd):
		return h.processDNDAttack(cmd)
	case dndDamageRegex.MatchString(cmd):
		return h.processDNDDamage(cmd)
	case dndAdvRegex.MatchString(cmd):
		return h.processDNDAdvantage(cmd)
	case dndDisRegex.MatchString(cmd):
		return h.processDNDDisadvantage(cmd)
	case dndHPRegex.MatchString(cmd):
		return h.processDNDHP(cmd)
	case dndSpellRegex.MatchString(cmd):
		return h.processDNDSpell(cmd)
	case dndInitiativeRegex.MatchString(cmd):
		return h.processDNDInitiative()
	case dndConditionRegex.MatchString(cmd):
		return h.processDNDCondition(cmd)
	case cmd == "help":
		return h.getHelp()
	default:
		return "未知指令，请输入.help查看帮助"
	}
}

// SendToQQ 发送消息到QQ
func (h *MessageHandler) SendToQQ(message string) error {
	if len(h.config.QQGroupID) == 0 {
		return fmt.Errorf("没有配置群组ID")
	}
	return h.connManager.SendMessage("send_group_msg", map[string]interface{}{
		"group_id": h.config.QQGroupID[0],
		"message":  message,
	})
}

// HandleGetGroupList 处理获取群组列表
func (h *MessageHandler) HandleGetGroupList(conn *websocket.Conn) {
	groups, err := h.connManager.GetGroupList()
	if err != nil {
		h.sendGroupListResponse(conn, nil, "获取群组列表失败: "+err.Error())
		return
	}

	h.sendGroupListResponse(conn, groups, "")
}

// HandleLeaveGroup 处理退出群组
func (h *MessageHandler) HandleLeaveGroup(conn *websocket.Conn, msgData map[string]interface{}) {
	groupID, ok := msgData["group_id"].(string)
	if !ok {
		h.sendGroupOperationResponse(conn, "leave", "缺少群组ID")
		return
	}

	gid, err := strconv.ParseInt(groupID, 10, 64)
	if err != nil {
		h.sendGroupOperationResponse(conn, "leave", "无效的群组ID格式")
		return
	}

	if err := h.connManager.LeaveGroup(gid); err != nil {
		h.sendGroupOperationResponse(conn, "leave", "退出群组失败: "+err.Error())
		return
	}

	h.sendGroupOperationResponse(conn, "leave", fmt.Sprintf("已发送退出群组 %d 的请求", gid))
}

// HandleDisableGroup 处理禁用群组
func (h *MessageHandler) HandleDisableGroup(conn *websocket.Conn, msgData map[string]interface{}) {
	groupID, ok := msgData["group_id"].(string)
	if !ok {
		h.sendGroupOperationResponse(conn, "disable", "缺少群组ID")
		return
	}

	h.sendGroupOperationResponse(conn, "disable", fmt.Sprintf("已切换群组 %s 的骰子功能状态", groupID))
}

// 辅助方法
func parseIncomingMessage(data []byte) (*OneBotMessage, error) {
	var msg OneBotMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, fmt.Errorf("invalid message: %v", err)
	}
	return &msg, nil
}

func extractMessageContent(msg json.RawMessage) (string, error) {
	var messageSegments []struct {
		Type string `json:"type"`
		Data struct {
			Text string `json:"text"`
		} `json:"data"`
	}

	if err := json.Unmarshal(msg, &messageSegments); err != nil {
		var text string
		if err := json.Unmarshal(msg, &text); err != nil {
			return "", fmt.Errorf("invalid message: %v", err)
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

func (h *MessageHandler) formatWebMessage(msg *OneBotMessage, response string) string {
	if msg.MessageType == "group" {
		return fmt.Sprintf("[群消息] %s", response)
	}
	return fmt.Sprintf("[私聊] %s", response)
}

func (h *MessageHandler) sendResponse(msg *OneBotMessage, response string) error {
	if strings.TrimPrefix(msg.RawMessage, ".") == "rh" && msg.MessageType == "group" {
		// 处理暗骰
		if err := h.connManager.SendMessage("send_group_msg", map[string]interface{}{
			"group_id": msg.GroupID,
			"message":  "事情似乎发生了什么变化",
		}); err != nil {
			return err
		}

		roll := rand.Intn(100) + 1
		privateMsg := fmt.Sprintf("在群 %d 中的暗骰结果是1D100=%d", msg.GroupID, roll)
		return h.connManager.SendMessage("send_private_msg", map[string]interface{}{
			"user_id": msg.UserID,
			"message": privateMsg,
		})
	}

	// 根据消息类型发送响应
	if msg.MessageType == "group" {
		return h.connManager.SendMessage("send_group_msg", map[string]interface{}{
			"group_id": msg.GroupID,
			"message":  response,
		})
	} else {
		return h.connManager.SendMessage("send_private_msg", map[string]interface{}{
			"user_id": msg.UserID,
			"message": response,
		})
	}
}

func (h *MessageHandler) sendGroupListResponse(conn *websocket.Conn, groups []connection.GroupInfo, errorMsg string) {
	response := map[string]interface{}{
		"type":   "group",
		"action": "list",
	}

	if errorMsg != "" {
		response["error"] = errorMsg
		response["groups"] = []connection.GroupInfo{}
	} else {
		response["groups"] = groups
	}

	if err := conn.WriteJSON(response); err != nil {
		log.Printf("发送群组列表响应失败: %v", err)
	}
}

func (h *MessageHandler) sendGroupOperationResponse(conn *websocket.Conn, action string, message string) {
	response := map[string]interface{}{
		"type":    "group",
		"action":  action,
		"message": message,
	}

	if err := conn.WriteJSON(response); err != nil {
		log.Printf("发送群组操作响应失败: %v", err)
	}
}

// 命令处理方法（从原main.go移植）
func (h *MessageHandler) processRoll(cmd string) string {
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
	l := parser.NewLexer(input)
	lexerWrapper := parser.NewLexerWrapper(l)
	result := parser.Parse(lexerWrapper)

	if result == 0 {
		return parser.GetResult()
	}
	return nil
}

func (h *MessageHandler) processSetDice(cmd string) string {
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

func (h *MessageHandler) processReasonRoll(cmd string) string {
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

func (h *MessageHandler) processRACheck(cmd string) string {
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

func (h *MessageHandler) processRBCheck(cmd string) string {
	matches := rbRegex.FindStringSubmatch(cmd)
	if len(matches) < 2 {
		return "无效的rb指令格式，正确格式：.rb 技能值"
	}

	skillValue, err := strconv.Atoi(matches[1])
	if err != nil || skillValue < 1 || skillValue > 100 {
		return "技能值必须为1-100的整数"
	}

	roll := rand.Intn(100) + 1
	result := fmt.Sprintf("检定rb %d → %d", skillValue, roll)

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

func (h *MessageHandler) processRCCheck(cmd string) string {
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

func (h *MessageHandler) processSanCheck(cmd string) string {
	matches := scRegex.FindStringSubmatch(cmd)
	if len(matches) < 3 {
		return "无效的sc指令格式，正确格式：.sc 成功值/失败值"
	}

	successValue, err := strconv.Atoi(matches[1])
	if err != nil || successValue < 1 || successValue > 100 {
		return "成功值必须为1-100的整数"
	}

	failValue, err := strconv.Atoi(matches[2])
	if err != nil || failValue < 1 || failValue > 100 {
		return "失败值必须为1-100的整数"
	}

	roll := rand.Intn(100) + 1
	result := fmt.Sprintf("理智检定sc %d/%d → %d", successValue, failValue, roll)

	if roll <= successValue {
		result += " 成功"
	} else if roll >= failValue {
		result += " 失败"
	} else {
		result += " 普通"
	}
	return result
}

func (h *MessageHandler) processCoC7() string {
	var attributes []string
	for i := 0; i < 9; i++ {
		roll := rand.Intn(6) + rand.Intn(6) + rand.Intn(6) + 3
		attributes = append(attributes, fmt.Sprintf("%s: %d", cocAttributes[i], roll*5))
	}
	return fmt.Sprintf("COC7版角色属性：\n%s", strings.Join(attributes, "\n"))
}

func (h *MessageHandler) processEnCheck(cmd string) string {
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
		return fmt.Sprintf("成长检定en %d → %d (失败)，技能提升到 %d", skillValue, roll, newSkill)
	}
	return fmt.Sprintf("成长检定en %d → %d (成功)，技能未提升", skillValue, roll)
}

func (h *MessageHandler) processTICheck() string {
	roll := rand.Intn(10) + 1
	effects := []string{
		"1. 失忆",
		"2. 被收容",
		"3. 偏执",
		"4. 狂躁",
		"5. 恐惧",
		"6. 幻觉",
		"7. 失语",
		"8. 失明",
		"9. 失聪",
		"10. 疯狂",
	}
	return fmt.Sprintf("临时疯狂ti → %d\n%s", roll, effects[roll-1])
}

func (h *MessageHandler) processLICheck() string {
	roll := rand.Intn(10) + 1
	effects := []string{
		"1. 失忆",
		"2. 被收容",
		"3. 偏执",
		"4. 狂躁",
		"5. 恐惧",
		"6. 幻觉",
		"7. 失语",
		"8. 失明",
		"9. 失聪",
		"10. 疯狂",
	}
	return fmt.Sprintf("长期疯狂li → %d\n%s", roll, effects[roll-1])
}

func (h *MessageHandler) processStCheck(cmd string) string {
	matches := stRegex.FindStringSubmatch(cmd)
	if len(matches) < 3 {
		return "无效的st指令格式，正确格式：.st 属性名 属性值 [属性名 属性值]..."
	}

	var attributes []string
	for i := 1; i < len(matches); i += 2 {
		if matches[i] != "" && matches[i+1] != "" {
			value, err := strconv.Atoi(matches[i+1])
			if err != nil {
				return fmt.Sprintf("无效的属性值: %s", matches[i+1])
			}
			attributes = append(attributes, fmt.Sprintf("%s: %d", matches[i], value))
		}
	}

	return fmt.Sprintf("角色属性设置成功：\n%s", strings.Join(attributes, "\n"))
}

// DND相关命令处理方法
func (h *MessageHandler) processDNDStat(cmd string) string {
	matches := dndStatRegex.FindStringSubmatch(cmd)
	if len(matches) < 2 {
		return "无效的dnd指令格式，正确格式：.dnd 属性名"
	}

	stat := strings.ToUpper(matches[1])
	roll := rand.Intn(6) + rand.Intn(6) + rand.Intn(6) + 3
	return fmt.Sprintf("DND属性 %s: %d", stat, roll)
}

func (h *MessageHandler) processDNDInit(cmd string) string {
	matches := dndInitRegex.FindStringSubmatch(cmd)
	if len(matches) < 2 {
		return "无效的init指令格式，正确格式：.init 先攻值"
	}

	initValue, err := strconv.Atoi(matches[1])
	if err != nil {
		return "先攻值必须是整数"
	}

	roll := rand.Intn(20) + 1
	total := roll + initValue
	return fmt.Sprintf("先攻检定: 1D20(%d) + %d = %d", roll, initValue, total)
}

func (h *MessageHandler) processDNDSave(cmd string) string {
	matches := dndSaveRegex.FindStringSubmatch(cmd)
	if len(matches) < 2 {
		return "无效的save指令格式，正确格式：.save 豁免类型"
	}

	saveType := strings.ToUpper(matches[1])
	roll := rand.Intn(20) + 1
	return fmt.Sprintf("%s豁免检定: 1D20 = %d", saveType, roll)
}

func (h *MessageHandler) processDNDCheck(cmd string) string {
	matches := dndCheckRegex.FindStringSubmatch(cmd)
	if len(matches) < 2 {
		return "无效的check指令格式，正确格式：.check 技能名"
	}

	skill := strings.ToUpper(matches[1])
	roll := rand.Intn(20) + 1
	return fmt.Sprintf("%s技能检定: 1D20 = %d", skill, roll)
}

func (h *MessageHandler) processDNDAttack(cmd string) string {
	matches := dndAttackRegex.FindStringSubmatch(cmd)
	if len(matches) < 2 {
		return "无效的attack指令格式，正确格式：.attack 攻击加值"
	}

	attackBonus, err := strconv.Atoi(matches[1])
	if err != nil {
		return "攻击加值必须是整数"
	}

	roll := rand.Intn(20) + 1
	total := roll + attackBonus
	result := fmt.Sprintf("攻击检定: 1D20(%d) + %d = %d", roll, attackBonus, total)
	
	if roll == 20 {
		result += " 重击！"
	} else if roll == 1 {
		result += " 失手！"
	}
	return result
}

func (h *MessageHandler) processDNDDamage(cmd string) string {
	matches := dndDamageRegex.FindStringSubmatch(cmd)
	if len(matches) < 2 {
		return "无效的damage指令格式，正确格式：.damage 伤害表达式"
	}

	damageExpr := matches[1]
	result, err := evaluateRollExpression(damageExpr)
	if err != nil {
		return err.Error()
	}
	return fmt.Sprintf("伤害检定: %s", result)
}

func (h *MessageHandler) processDNDAdvantage(cmd string) string {
	matches := dndAdvRegex.FindStringSubmatch(cmd)
	if len(matches) < 2 {
		return "无效的adv指令格式，正确格式：.adv 技能名"
	}

	skill := strings.ToUpper(matches[1])
	roll1 := rand.Intn(20) + 1
	roll2 := rand.Intn(20) + 1
	result := math.Max(float64(roll1), float64(roll2))
	return fmt.Sprintf("%s优势检定: 1D20(%d) 1D20(%d) = %d", skill, roll1, roll2, int(result))
}

func (h *MessageHandler) processDNDDisadvantage(cmd string) string {
	matches := dndDisRegex.FindStringSubmatch(cmd)
	if len(matches) < 2 {
		return "无效的dis指令格式，正确格式：.dis 技能名"
	}

	skill := strings.ToUpper(matches[1])
	roll1 := rand.Intn(20) + 1
	roll2 := rand.Intn(20) + 1
	result := math.Min(float64(roll1), float64(roll2))
	return fmt.Sprintf("%s劣势检定: 1D20(%d) 1D20(%d) = %d", skill, roll1, roll2, int(result))
}

func (h *MessageHandler) processDNDHP(cmd string) string {
	matches := dndHPRegex.FindStringSubmatch(cmd)
	if len(matches) < 3 {
		return "无效的hp指令格式，正确格式：.hp 当前生命值/最大生命值"
	}

	currentHP, err := strconv.Atoi(matches[1])
	if err != nil {
		return "当前生命值必须是整数"
	}

	maxHP, err := strconv.Atoi(matches[2])
	if err != nil {
		return "最大生命值必须是整数"
	}

	return fmt.Sprintf("生命值: %d/%d", currentHP, maxHP)
}

func (h *MessageHandler) processDNDSpell(cmd string) string {
	matches := dndSpellRegex.FindStringSubmatch(cmd)
	if len(matches) < 2 {
		return "无效的spell指令格式，正确格式：.spell 法术等级"
	}

	spellLevel, err := strconv.Atoi(matches[1])
	if err != nil || spellLevel < 1 || spellLevel > 9 {
		return "法术等级必须是1-9的整数"
	}

	roll := rand.Intn(20) + 1 + spellLevel
	return fmt.Sprintf("法术攻击检定: 1D20 + %d = %d", spellLevel, roll)
}

func (h *MessageHandler) processDNDInitiative() string {
	initiativeMutex.RLock()
	defer initiativeMutex.RUnlock()

	if len(initiativeList) == 0 {
		return "先攻列表为空"
	}

	var initiatives []string
	for name, value := range initiativeList {
		initiatives = append(initiatives, fmt.Sprintf("%s: %d", name, value))
	}

	sort.Slice(initiatives, func(i, j int) bool {
		valI, _ := strconv.Atoi(strings.Split(initiatives[i], ": ")[1])
		valJ, _ := strconv.Atoi(strings.Split(initiatives[j], ": ")[1])
		return valI > valJ
	})

	return fmt.Sprintf("先攻顺序：\n%s", strings.Join(initiatives, "\n"))
}

func (h *MessageHandler) processDNDCondition(cmd string) string {
	matches := dndConditionRegex.FindStringSubmatch(cmd)
	if len(matches) < 2 {
		return "无效的condition指令格式，正确格式：.condition 角色名 状态"
	}

	parts := strings.Split(matches[1], " ")
	if len(parts) < 2 {
		return "请提供角色名和状态"
	}

	character := parts[0]
	condition := strings.Join(parts[1:], " ")

	conditionMutex.Lock()
	conditionList[character] = append(conditionList[character], condition)
	conditionMutex.Unlock()

	return fmt.Sprintf("已为 %s 添加状态: %s", character, condition)
}

func (h *MessageHandler) getHelp() string {
	return `可用指令：
基础骰子：
.r [表达式] - 投掷骰子
.set[数字] - 设置默认骰子面数
.r [理由] - 理由投掷

COC相关：
.ra [技能值] - 技能检定
.rb [技能值] - 战斗检定  
.rc [技能值] - 驾驶检定
.rh - 暗骰
.sc [成功值]/[失败值] - 理智检定
.en [技能值] - 成长检定
.coc7 - 生成COC7版角色
.st [属性名] [属性值]... - 设置角色属性
.ti - 临时疯狂
.li - 长期疯狂

DND相关：
.dnd [属性名] - 生成DND属性
.init [先攻值] - 先攻检定
.save [豁免类型] - 豁免检定
.check [技能名] - 技能检定
.attack [攻击加值] - 攻击检定
.damage [伤害表达式] - 伤害检定
.adv [技能名] - 优势检定
.dis [技能名] - 劣势检定
.hp [当前]/[最大] - 生命值
.spell [法术等级] - 法术攻击
.initiative - 查看先攻列表
.condition [角色名] [状态] - 添加状态

其他：
.help - 显示此帮助`
}
