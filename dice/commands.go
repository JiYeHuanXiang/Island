package dice

import (
	"fmt"
	"regexp"
	"strings"
)

// CommandHandler 指令处理器接口
type CommandHandler interface {
	GetName() string
	GetHelp() string
	Match(cmd string) bool
	Process(ctx *CommandContext) string
}

// CommandContext 命令执行上下文
type CommandContext struct {
	PlayerID int64
	GroupID  int64
	Args     string
	Engine   *Engine
}

// BaseCommand 基础指令结构
type BaseCommand struct {
	name string
	help string
	regex *regexp.Regexp
}

// GetName 获取指令名称
func (c *BaseCommand) GetName() string {
	return c.name
}

// GetHelp 获取帮助文本
func (c *BaseCommand) GetHelp() string {
	return c.help
}

// RollCommand .r 指令
type RollCommand struct {
	BaseCommand
}

func NewRollCommand() *RollCommand {
	return &RollCommand{
		BaseCommand: BaseCommand{
			name: "r",
			help: ".r [表达式] - 投掷骰子，例如 .r 3d6+5",
			regex: regexp.MustCompile(`^r\s*(.+)$`),
		},
	}
}

func (c *RollCommand) Match(cmd string) bool {
	return c.regex.MatchString(cmd)
}

func (c *RollCommand) Process(ctx *CommandContext) string {
	matches := c.regex.FindStringSubmatch(ctx.Args)
	if len(matches) < 2 {
		return "用法: .r [骰子表达式]"
	}
	return ctx.Engine.Roll(matches[1])
}

// RACheckCommand .ra 指令 (技能检定)
type RACheckCommand struct {
	BaseCommand
}

func NewRACheckCommand() *RACheckCommand {
	return &RACheckCommand{
		BaseCommand: BaseCommand{
			name: "ra",
			help: ".ra [技能值] - 技能检定",
			regex: regexp.MustCompile(`^ra\s+(\d+)$`),
		},
	}
}

func (c *RACheckCommand) Match(cmd string) bool {
	return c.regex.MatchString(cmd)
}

func (c *RACheckCommand) Process(ctx *CommandContext) string {
	matches := c.regex.FindStringSubmatch(ctx.Args)
	if len(matches) < 2 {
		return "用法: .ra [技能值]"
	}
	skillValue := 0
	fmt.Sscanf(matches[1], "%d", &skillValue)
	return ctx.Engine.CoC7SkillCheck(skillValue)
}

// RBCheckCommand .rb 指令 (战斗检定)
type RBCheckCommand struct {
	BaseCommand
}

func NewRBCheckCommand() *RBCheckCommand {
	return &RBCheckCommand{
		BaseCommand: BaseCommand{
			name: "rb",
			help: ".rb [技能值] - 战斗检定",
			regex: regexp.MustCompile(`^rb\s+(\d+)$`),
		},
	}
}

func (c *RBCheckCommand) Match(cmd string) bool {
	return c.regex.MatchString(cmd)
}

func (c *RBCheckCommand) Process(ctx *CommandContext) string {
	matches := c.regex.FindStringSubmatch(ctx.Args)
	if len(matches) < 2 {
		return "用法: .rb [技能值]"
	}
	skillValue := 0
	fmt.Sscanf(matches[1], "%d", &skillValue)
	return ctx.Engine.CoC7SkillCheck(skillValue)
}

// RCCheckCommand .rc 指令 (驾驶检定)
type RCCheckCommand struct {
	BaseCommand
}

func NewRCCheckCommand() *RCCheckCommand {
	return &RCCheckCommand{
		BaseCommand: BaseCommand{
			name: "rc",
			help: ".rc [技能值] - 驾驶检定",
			regex: regexp.MustCompile(`^rc\s+(\d+)$`),
		},
	}
}

func (c *RCCheckCommand) Match(cmd string) bool {
	return c.regex.MatchString(cmd)
}

func (c *RCCheckCommand) Process(ctx *CommandContext) string {
	matches := c.regex.FindStringSubmatch(ctx.Args)
	if len(matches) < 2 {
		return "用法: .rc [技能值]"
	}
	skillValue := 0
	fmt.Sscanf(matches[1], "%d", &skillValue)
	return ctx.Engine.CoC7SkillCheck(skillValue)
}

// SCCheckCommand .sc 指令 (理智检定)
type SCCheckCommand struct {
	BaseCommand
}

func NewSCCheckCommand() *SCCheckCommand {
	return &SCCheckCommand{
		BaseCommand: BaseCommand{
			name: "sc",
			help: ".sc [成功值]/[失败值] - 理智检定",
			regex: regexp.MustCompile(`^sc\s+(\d+)/(\d+)$`),
		},
	}
}

func (c *SCCheckCommand) Match(cmd string) bool {
	return c.regex.MatchString(cmd)
}

func (c *SCCheckCommand) Process(ctx *CommandContext) string {
	matches := c.regex.FindStringSubmatch(ctx.Args)
	if len(matches) < 3 {
		return "用法: .sc [成功值]/[失败值]"
	}
	successValue := 0
	failValue := 0
	fmt.Sscanf(matches[1], "%d", &successValue)
	fmt.Sscanf(matches[2], "%d", &failValue)
	return ctx.Engine.CoC7SanCheck(successValue, failValue)
}

// ENCheckCommand .en 指令 (成长检定)
type ENCheckCommand struct {
	BaseCommand
}

func NewENCheckCommand() *ENCheckCommand {
	return &ENCheckCommand{
		BaseCommand: BaseCommand{
			name: "en",
			help: ".en [技能值] - 成长检定",
			regex: regexp.MustCompile(`^en\s+(\d+)$`),
		},
	}
}

func (c *ENCheckCommand) Match(cmd string) bool {
	return c.regex.MatchString(cmd)
}

func (c *ENCheckCommand) Process(ctx *CommandContext) string {
	matches := c.regex.FindStringSubmatch(ctx.Args)
	if len(matches) < 2 {
		return "用法: .en [技能值]"
	}
	skillValue := 0
	fmt.Sscanf(matches[1], "%d", &skillValue)
	return ctx.Engine.CoC7GrowthCheck(skillValue)
}

// COC7Command .coc7 指令
type COC7Command struct {
	BaseCommand
}

func NewCOC7Command() *COC7Command {
	return &COC7Command{
		BaseCommand: BaseCommand{
			name: "coc7",
			help: ".coc7 - 生成COC7版角色",
			regex: regexp.MustCompile(`^coc7$`),
		},
	}
}

func (c *COC7Command) Match(cmd string) bool {
	return c.regex.MatchString(cmd)
}

func (c *COC7Command) Process(ctx *CommandContext) string {
	return ctx.Engine.CoC7RollAttributes()
}

// TICommand .ti 指令 (临时疯狂)
type TICommand struct {
	BaseCommand
}

func NewTICommand() *TICommand {
	return &TICommand{
		BaseCommand: BaseCommand{
			name: "ti",
			help: ".ti - 临时疯狂表",
			regex: regexp.MustCompile(`^ti$`),
		},
	}
}

func (c *TICommand) Match(cmd string) bool {
	return c.regex.MatchString(cmd)
}

func (c *TICommand) Process(ctx *CommandContext) string {
	return ctx.Engine.CoC7TempInsanity()
}

// LICommand .li 指令 (长期疯狂)
type LICommand struct {
	BaseCommand
}

func NewLICommand() *LICommand {
	return &LICommand{
		BaseCommand: BaseCommand{
			name: "li",
			help: ".li - 长期疯狂表",
			regex: regexp.MustCompile(`^li$`),
		},
	}
}

func (c *LICommand) Match(cmd string) bool {
	return c.regex.MatchString(cmd)
}

func (c *LICommand) Process(ctx *CommandContext) string {
	return ctx.Engine.CoC7LongInsanity()
}

// DNDStatCommand .dnd 指令
type DNDStatCommand struct {
	BaseCommand
}

func NewDNDStatCommand() *DNDStatCommand {
	return &DNDStatCommand{
		BaseCommand: BaseCommand{
			name: "dnd",
			help: ".dnd [属性名] - 生成DND属性",
			regex: regexp.MustCompile(`^dnd\s+(\w+)$`),
		},
	}
}

func (c *DNDStatCommand) Match(cmd string) bool {
	return c.regex.MatchString(cmd)
}

func (c *DNDStatCommand) Process(ctx *CommandContext) string {
	matches := c.regex.FindStringSubmatch(ctx.Args)
	if len(matches) < 2 {
		return "用法: .dnd [属性名]，例如: .dnd str"
	}
	stat := strings.ToUpper(matches[1])
	return ctx.Engine.DnD5ERollAttribute(stat)
}

// DNDInitCommand .init 指令
type DNDInitCommand struct {
	BaseCommand
}

func NewDNDInitCommand() *DNDInitCommand {
	return &DNDInitCommand{
		BaseCommand: BaseCommand{
			name: "init",
			help: ".init [先攻加值] - 先攻检定",
			regex: regexp.MustCompile(`^init\s+(\d+)$`),
		},
	}
}

func (c *DNDInitCommand) Match(cmd string) bool {
	return c.regex.MatchString(cmd)
}

func (c *DNDInitCommand) Process(ctx *CommandContext) string {
	matches := c.regex.FindStringSubmatch(ctx.Args)
	if len(matches) < 2 {
		return "用法: .init [先攻加值]"
	}
	mod := 0
	fmt.Sscanf(matches[1], "%d", &mod)
	return ctx.Engine.DnD5EInitiative(mod)
}

// DNDAttackCommand .attack 指令
type DNDAttackCommand struct {
	BaseCommand
}

func NewDNDAttackCommand() *DNDAttackCommand {
	return &DNDAttackCommand{
		BaseCommand: BaseCommand{
			name: "attack",
			help: ".attack [攻击加值] - 攻击检定",
			regex: regexp.MustCompile(`^attack\s+(\d+)$`),
		},
	}
}

func (c *DNDAttackCommand) Match(cmd string) bool {
	return c.regex.MatchString(cmd)
}

func (c *DNDAttackCommand) Process(ctx *CommandContext) string {
	matches := c.regex.FindStringSubmatch(ctx.Args)
	if len(matches) < 2 {
		return "用法: .attack [攻击加值]"
	}
	bonus := 0
	fmt.Sscanf(matches[1], "%d", &bonus)
	return ctx.Engine.DnD5EAttack(bonus)
}

// CommandRegistry 指令注册表
type CommandRegistry struct {
	commands []CommandHandler
}

// NewCommandRegistry 创建指令注册表
func NewCommandRegistry() *CommandRegistry {
	r := &CommandRegistry{
		commands: make([]CommandHandler, 0),
	}
	
	// 注册所有内置指令
	r.commands = append(r.commands, NewRollCommand())
	r.commands = append(r.commands, NewRACheckCommand())
	r.commands = append(r.commands, NewRBCheckCommand())
	r.commands = append(r.commands, NewRCCheckCommand())
	r.commands = append(r.commands, NewSCCheckCommand())
	r.commands = append(r.commands, NewENCheckCommand())
	r.commands = append(r.commands, NewCOC7Command())
	r.commands = append(r.commands, NewTICommand())
	r.commands = append(r.commands, NewLICommand())
	r.commands = append(r.commands, NewDNDStatCommand())
	r.commands = append(r.commands, NewDNDInitCommand())
	r.commands = append(r.commands, NewDNDAttackCommand())
	
	return r
}

// Process 处理命令
func (r *CommandRegistry) Process(cmd string, ctx *CommandContext) string {
	// 去除前导点号和空格
	cmd = strings.TrimPrefix(cmd, ".")
	cmd = strings.TrimSpace(cmd)
	ctx.Args = cmd
	
	// 匹配指令
	for _, c := range r.commands {
		if c.Match(cmd) {
			return c.Process(ctx)
		}
	}
	
	// 未匹配的指令
	return "未知指令，请输入 .help 查看帮助"
}

// GetHelp 获取所有指令的帮助
func (r *CommandRegistry) GetHelp() string {
	var lines []string
	lines = append(lines, "可用指令：")
	lines = append(lines, "")
	lines = append(lines, "基础骰子：")
	lines = append(lines, "  .r [表达式] - 投掷骰子")
	lines = append(lines, "")
	lines = append(lines, "COC7相关：")
	lines = append(lines, "  .coc7 - 生成COC7版角色")
	lines = append(lines, "  .ra [技能值] - 技能检定")
	lines = append(lines, "  .rb [技能值] - 战斗检定")
	lines = append(lines, "  .rc [技能值] - 驾驶检定")
	lines = append(lines, "  .sc [成功]/[失败] - 理智检定")
	lines = append(lines, "  .en [技能值] - 成长检定")
	lines = append(lines, "  .ti - 临时疯狂表")
	lines = append(lines, "  .li - 长期疯狂表")
	lines = append(lines, "")
	lines = append(lines, "DND5E相关：")
	lines = append(lines, "  .dnd [属性] - 生成DND属性")
	lines = append(lines, "  .init [加值] - 先攻检定")
	lines = append(lines, "  .attack [加值] - 攻击检定")
	
	return strings.Join(lines, "\n")
}
