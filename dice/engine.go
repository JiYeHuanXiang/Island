package dice

import (
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"sync"
)

// CoC7 属性列表
var CoC7Attributes = [...]string{"STR", "CON", "SIZ", "DEX", "APP", "INT", "POW", "EDU", "LUK"}

// Engine 骰子引擎
type Engine struct {
	mu              sync.RWMutex
	defaultDiceSides int
}

// New 创建新的骰子引擎
func New() *Engine {
	return &Engine{
		defaultDiceSides: 100,
	}
}

// SetDefaultSides 设置默认骰子面数
func (e *Engine) SetDefaultSides(sides int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.defaultDiceSides = sides
}

// Roll 执行基础掷骰
func (e *Engine) Roll(expression string) string {
	// 这里会调用 parser 模块进行解析
	// 目前简化处理
	result := e.simpleRoll(expression)
	return result
}

// simpleRoll 简单掷骰实现
func (e *Engine) simpleRoll(expr string) string {
	expr = strings.TrimSpace(expr)
	
	// 解析简单的 XdY 格式
	var count, sides int
	parts := strings.Split(expr, "d")
	
	if len(parts) != 2 {
		return fmt.Sprintf("无效的骰子表达式: %s", expr)
	}
	
	// 解析数量
	if parts[0] == "" {
		count = 1
	} else {
		c, err := strconv.Atoi(parts[0])
		if err != nil {
			return fmt.Sprintf("无效的骰子数量: %s", parts[0])
		}
		count = c
	}
	
	// 解析面数
	sidePart := strings.Split(parts[1], "+")[0]
	sidePart = strings.Split(sidePart, "-")[0]
	s, err := strconv.Atoi(sidePart)
	if err != nil {
		return fmt.Sprintf("无效的骰子面数: %s", sidePart)
	}
	sides = s
	
	// 执行掷骰
	if count > 100 {
		return fmt.Sprintf("骰子数量过多: %d", count)
	}
	if sides > 1000 {
		return fmt.Sprintf("骰子面数过多: %d", sides)
	}
	
	var total int
	var rolls []int
	for i := 0; i < count; i++ {
		r := rand.Intn(sides) + 1
		rolls = append(rolls, r)
		total += r
	}
	
	if count == 1 {
		return fmt.Sprintf("掷骰结果: 1D%d=%d", sides, total)
	}
	return fmt.Sprintf("掷骰结果: %dD%d=%d (详情: %v)", count, sides, total, rolls)
}

// RollWithModifier 执行带修正值的掷骰
func (e *Engine) RollWithModifier(expression string) string {
	// 解析修正值
	var modifier int
	var dicePart string
	
	if idx := strings.IndexAny(expression, "+-"); idx != -1 {
		op := expression[idx]
		dicePart = expression[:idx]
		modStr := expression[idx+1:]
		if m, err := strconv.Atoi(modStr); err == nil {
			if op == '-' {
				modifier = -m
			} else {
				modifier = m
			}
		}
	} else {
		dicePart = expression
	}
	
	// 执行掷骰
	result := e.simpleRoll(dicePart)
	
	// 添加修正值
	if modifier != 0 {
		// 从结果中提取数字
		var total int
		fmt.Sscanf(result, "掷骰结果: %*[^=]=%d", &total)
		total += modifier
		
		if modifier > 0 {
			return fmt.Sprintf("%s+%d=%d", dicePart, modifier, total)
		} else {
			return fmt.Sprintf("%s%d=%d", dicePart, modifier, total)
		}
	}
	
	return result
}

// CoC7RollAttributes 生成 CoC7 属性
func (e *Engine) CoC7RollAttributes() string {
	var attrs []string
	for _, attr := range CoC7Attributes {
		roll := rand.Intn(6) + rand.Intn(6) + rand.Intn(6) + 3
		value := roll * 5
		attrs = append(attrs, fmt.Sprintf("%s: %d", attr, value))
	}
	return "COC7版角色属性：\n" + strings.Join(attrs, "\n")
}

// CoC7SkillCheck 技能检定
func (e *Engine) CoC7SkillCheck(skillValue int) string {
	if skillValue < 1 || skillValue > 100 {
		return "技能值必须在1-100之间"
	}
	
	roll := rand.Intn(100) + 1
	result := fmt.Sprintf("技能检定 %d → %d", skillValue, roll)
	
	if roll <= skillValue {
		if roll <= 5 {
			result += " 大成功！"
		} else if roll == 1 {
			result += " 成功"
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

// CoC7SanCheck 理智检定
func (e *Engine) CoC7SanCheck(successValue, failValue int) string {
	if successValue < 1 || successValue > 100 || failValue < 1 || failValue > 100 {
		return "理智值必须在1-100之间"
	}
	
	roll := rand.Intn(100) + 1
	result := fmt.Sprintf("理智检定 sc %d/%d → %d", successValue, failValue, roll)
	
	if roll <= successValue {
		result += " 成功"
	} else if roll >= failValue {
		result += " 失败"
	} else {
		result += " 普通"
	}
	
	return result
}

// CoC7GrowthCheck 成长检定
func (e *Engine) CoC7GrowthCheck(skillValue int) string {
	if skillValue < 1 || skillValue > 100 {
		return "技能值必须在1-100之间"
	}
	
	roll := rand.Intn(100) + 1
	if roll > skillValue {
		increase := rand.Intn(10) + 1
		newSkill := skillValue + increase
		if newSkill > 100 {
			newSkill = 100
		}
		return fmt.Sprintf("成长检定 en %d → %d (失败)，技能提升到 %d", skillValue, roll, newSkill)
	}
	return fmt.Sprintf("成长检定 en %d → %d (成功)，技能未提升", skillValue, roll)
}

// CoC7TempInsanity 临时疯狂
func (e *Engine) CoC7TempInsanity() string {
	effects := []string{
		"1. 失忆 - 你忘记了之前发生的事情",
		"2. 被收容 - 你被送往精神病院",
		"3. 偏执 - 你认为有人要伤害你",
		"4. 狂躁 - 你变得极度兴奋和活跃",
		"5. 恐惧 - 你感到极度的恐惧",
		"6. 幻觉 - 你看到了不存在的东西",
		"7. 失语 - 你无法说话",
		"8. 失明 - 你暂时失明",
		"9. 失聪 - 你暂时失聪",
		"10. 疯狂 - 你陷入疯狂状态",
	}
	
	roll := rand.Intn(10) + 1
	return fmt.Sprintf("临时疯狂 ti → %d\n%s", roll, effects[roll-1])
}

// CoC7LongInsanity 长期疯狂
func (e *Engine) CoC7LongInsanity() string {
	effects := []string{
		"1. 失忆 - 你失去了所有记忆",
		"2. 假性失忆 - 你编造了一段虚假记忆",
		"3. 偏执 - 持续的被迫害妄想",
		"4. 人格分裂 - 出现了第二人格",
		"5. 恐惧 - 对某种事物的恐惧",
		"6. 狂热 - 对某种理念的狂热",
		"7. 社交恐惧 - 害怕与人交往",
		"8. 抑郁 - 深深的绝望感",
		"9. 躁狂 - 过度活跃和冲动",
		"10. 麻木 - 情感完全丧失",
	}
	
	roll := rand.Intn(10) + 1
	return fmt.Sprintf("长期疯狂 li → %d\n%s", roll, effects[roll-1])
}

// DnD5ERollAttribute 生成 DnD 属性
func (e *Engine) DnD5ERollAttribute(stat string) string {
	rolls := []int{
		rand.Intn(6) + 1,
		rand.Intn(6) + 1,
		rand.Intn(6) + 1,
		rand.Intn(6) + 1,
	}
	
	// 去掉最小的
	minIdx := 0
	for i := 1; i < len(rolls); i++ {
		if rolls[i] < rolls[minIdx] {
			minIdx = i
		}
	}
	rolls = append(rolls[:minIdx], rolls[minIdx+1:]...)
	
	total := 0
	for _, r := range rolls {
		total += r
	}
	
	mod := (total - 10) / 2
	modStr := ""
	if mod >= 0 {
		modStr = fmt.Sprintf("+%d", mod)
	} else {
		modStr = fmt.Sprintf("%d", mod)
	}
	
	return fmt.Sprintf("DND %s: %d (修正值 %s)", strings.ToUpper(stat), total, modStr)
}

// DnD5EAttack 攻击检定
func (e *Engine) DnD5EAttack(attackBonus int) string {
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

// DnD5EInitiative 先攻检定
func (e *Engine) DnD5EInitiative(dexMod int) string {
	roll := rand.Intn(20) + 1
	total := roll + dexMod
	return fmt.Sprintf("先攻检定: 1D20(%d) + %d = %d", roll, dexMod, total)
}

// DnD5ESave 豁免检定
func (e *Engine) DnD5ESave(saveDC int) string {
	roll := rand.Intn(20) + 1
	total := roll + saveDC
	result := fmt.Sprintf("豁免检定: 1D20(%d) + %d = %d", roll, saveDC, total)
	
	if roll == 20 {
		result += " 成功！"
	} else if roll == 1 {
		result += " 失败！"
	}
	
	return result
}

// DnD5ECheck 技能检定
func (e *Engine) DnD5ECheck(skillName string, profBonus int) string {
	roll := rand.Intn(20) + 1
	total := roll + profBonus
	result := fmt.Sprintf("%s技能检定: 1D20(%d) + %d = %d", skillName, roll, profBonus, total)
	
	if roll == 20 {
		result += " 大成功！"
	} else if roll == 1 {
		result += " 大失败！"
	}
	
	return result
}

// AdvantageRoll 优势掷骰
func (e *Engine) AdvantageRoll() (int, int) {
	r1 := rand.Intn(20) + 1
	r2 := rand.Intn(20) + 1
	return r1, r2
}

// DisadvantageRoll 劣势掷骰
func (e *Engine) DisadvantageRoll() (int, int) {
	r1 := rand.Intn(20) + 1
	r2 := rand.Intn(20) + 1
	return r1, r2
}

// GetAdvantageResult 获取优势掷骰结果
func (e *Engine) GetAdvantageResult(skillName string, modifier int) string {
	r1, r2 := e.AdvantageRoll()
	best := int(math.Max(float64(r1), float64(r2)))
	total := best + modifier
	return fmt.Sprintf("%s优势检定: 1D20(%d) 1D20(%d) = %d + %d = %d", 
		skillName, r1, r2, best, modifier, total)
}

// GetDisadvantageResult 获取劣势掷骰结果
func (e *Engine) GetDisadvantageResult(skillName string, modifier int) string {
	r1, r2 := e.DisadvantageRoll()
	worst := int(math.Min(float64(r1), float64(r2)))
	total := worst + modifier
	return fmt.Sprintf("%s劣势检定: 1D20(%d) 1D20(%d) = %d + %d = %d", 
		skillName, r1, r2, worst, modifier, total)
}
