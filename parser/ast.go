package parser

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"strings"
)

var (
	MAX_ITERATIONS = 8192
	MAX_ROLLS      = 1038576000
	MAX_DICE_SIDES = 1038576000
	MIN_DICE_SIDES = 1
)

// Interface for expressions that can return both a numeric value and a list of rolls
type MultiValueExpr interface {
	Expr
	EvaluateMulti() []int
}

// FateDiceExpr represents a FATE dice roll (df/f)
type FateDiceExpr struct {
	Count   Expr
	process string
	rolls   []int
}

func (e *FateDiceExpr) GetProcess() string {
	return e.process
}

func (e *FateDiceExpr) Evaluate() interface{} {
	count := 4 // Default is 4 dice
	if e.Count != nil {
		count = e.Count.Evaluate().(int)
	}

	if count > MAX_ROLLS {
		fmt.Println(fmt.Sprintf("Exceeded maximum rolls: %d", MAX_ROLLS))
		return 0
	}

	e.rolls = make([]int, count)
	sum := 0

	for i := 0; i < count; i++ {
		// FATE dice have values of -1, 0, and 1
		roll := rand.Intn(3) - 1
		e.rolls[i] = roll
		sum += roll
	}

	// Convert rolls to symbols for display
	symbols := make([]string, count)
	for i, r := range e.rolls {
		switch r {
		case -1:
			symbols[i] = "-"
		case 0:
			symbols[i] = "0"
		case 1:
			symbols[i] = "+"
		}
	}

	// 记录投掷过程
	e.process = fmt.Sprintf("%df = %s = %d", count, formatRollsCompact(e.rolls), sum)

	return sum
}

// 辅助函数将符号转换为整数进行显示
func symbolsAsInts(symbols []string) []int {
	result := make([]int, len(symbols))
	for i, s := range symbols {
		switch s {
		case "-":
			result[i] = -1
		case "0":
			result[i] = 0
		case "+":
			result[i] = 1
		}
	}
	return result
}

func (e *FateDiceExpr) EvaluateMulti() []int {
	count := 4 // Default is 4 dice
	if e.Count != nil {
		count = e.Count.Evaluate().(int)
	}

	if count > MAX_ROLLS {
		fmt.Println(fmt.Sprintf("Exceeded maximum rolls: %d", MAX_ROLLS))
		return []int{0}
	}

	rolls := make([]int, count)
	for i := 0; i < count; i++ {
		rolls[i] = rand.Intn(3) - 1
	}
	return rolls
}

// PenaltyBonusDiceExpr represents CoC penalty or bonus dice (pb)
type PenaltyBonusDiceExpr struct {
	IsBonus bool
	Count   Expr
}

func (e *PenaltyBonusDiceExpr) Evaluate() interface{} {
	count := 1
	if e.Count != nil {
		count = e.Count.Evaluate().(int)
	}

	if count > MAX_ROLLS {
		fmt.Println(fmt.Sprintf("Exceeded maximum rolls: %d", MAX_ROLLS))
		return 0
	}

	// Roll the tens digit (0-9)
	tensDie := rand.Intn(10)

	// Roll the units digit (0-9)
	unitsDie := rand.Intn(10)

	// Roll the penalty/bonus dice (0-9)
	bonusDice := make([]int, count)
	for i := 0; i < count; i++ {
		bonusDice[i] = rand.Intn(10)
	}

	// Calculate the result based on penalty or bonus
	var result int
	var tensDigit int

	if e.IsBonus {
		// For bonus: use the lowest tens digit
		tensDigit = tensDie
		for _, die := range bonusDice {
			if die < tensDigit {
				tensDigit = die
			}
		}
	} else {
		// For penalty: use the highest tens digit
		tensDigit = tensDie
		for _, die := range bonusDice {
			if die > tensDigit {
				tensDigit = die
			}
		}
	}

	// Special case: handle 00 as 100
	if tensDigit == 0 && unitsDie == 0 {
		result = 100
	} else {
		result = tensDigit*10 + unitsDie
	}

	bonusType := "b"
	if !e.IsBonus {
		bonusType = "p"
	}

	// 使用统一的输出格式
	allRolls := append([]int{tensDie*10 + unitsDie}, bonusDice...)
	fmt.Fprintln(os.Stdout, formatDiceOutput(fmt.Sprintf("1%s%d", bonusType, count), allRolls, result))

	return result
}

// InfinitePoolDiceExpr represents infinite addition dice pool (a)
type InfinitePoolDiceExpr struct {
	Initial            Expr
	AddLine            Expr
	SuccessLine        Expr
	ReverseSuccessLine Expr
	Sides              Expr
}

func (e *InfinitePoolDiceExpr) Evaluate() interface{} {
	initial := e.Initial.Evaluate().(int)
	addLine := e.AddLine.Evaluate().(int)

	successLine := 8 // Default success line is 8
	if e.SuccessLine != nil {
		successLine = e.SuccessLine.Evaluate().(int)
	}

	sides := 10 // Default sides is 10
	if e.Sides != nil {
		sides = e.Sides.Evaluate().(int)
	}

	if sides < MIN_DICE_SIDES || sides > MAX_DICE_SIDES {
		fmt.Println(fmt.Sprintf("Invalid number of sides: %d", sides))
		return 0
	}

	iterations := 0
	totalSuccesses := 0
	allRolls := make([][]int, 0)

	currentPool := initial

	for currentPool > 0 {
		iterations++
		if iterations > MAX_ITERATIONS {
			fmt.Println(fmt.Sprintf("Exceeded maximum iterations: %d", MAX_ITERATIONS))
			break
		}

		// Roll the current pool of dice
		rolls := make([]int, currentPool)
		successes := 0

		for i := 0; i < currentPool; i++ {
			roll := rand.Intn(sides) + 1
			rolls[i] = roll

			// Count successes for next pool
			if roll >= addLine {
				successes++
			}
		}

		allRolls = append(allRolls, rolls)
		currentPool = successes

		// Count successes for final result
		for _, roll := range rolls {
			if e.ReverseSuccessLine != nil {
				reverseSuccessLine := e.ReverseSuccessLine.Evaluate().(int)
				if roll <= reverseSuccessLine {
					totalSuccesses++
				}
			} else if roll >= successLine {
				totalSuccesses++
			}
		}
	}

	// 构建原始表达式
	expr := fmt.Sprintf("%da%d", initial, addLine)
	if e.SuccessLine != nil {
		expr += fmt.Sprintf("k%d", successLine)
	}
	if e.ReverseSuccessLine != nil {
		expr += fmt.Sprintf("q%d", e.ReverseSuccessLine.Evaluate().(int))
	}
	if e.Sides != nil && sides != 10 {
		expr += fmt.Sprintf("m%d", sides)
	}

	// 创建合并的骰子结果
	allDiceResults := make([]int, 0)
	for _, round := range allRolls {
		allDiceResults = append(allDiceResults, round...)
	}

	// 使用统一的输出格式
	fmt.Fprintln(os.Stdout, formatDiceOutput(expr, allDiceResults, totalSuccesses))

	return totalSuccesses
}

// DoubleCrossDiceExpr represents double cross addition dice pool (c)
type DoubleCrossDiceExpr struct {
	Initial Expr
	AddLine Expr
	Sides   Expr
}

func (e *DoubleCrossDiceExpr) Evaluate() interface{} {
	initial := e.Initial.Evaluate().(int)
	addLine := e.AddLine.Evaluate().(int)

	sides := 10 // Default sides is 10
	if e.Sides != nil {
		sides = e.Sides.Evaluate().(int)
	}

	if sides < MIN_DICE_SIDES || sides > MAX_DICE_SIDES {
		fmt.Println(fmt.Sprintf("Invalid number of sides: %d", sides))
		return 0
	}

	iterations := 0
	totalScore := 0
	allRolls := make([][]int, 0)

	currentPool := initial

	for currentPool > 0 {
		iterations++
		if iterations > MAX_ITERATIONS {
			fmt.Println(fmt.Sprintf("Exceeded maximum iterations: %d", MAX_ITERATIONS))
			break
		}

		// Roll the current pool of dice
		rolls := make([]int, currentPool)
		successes := 0

		for i := 0; i < currentPool; i++ {
			roll := rand.Intn(sides) + 1
			rolls[i] = roll

			// Count successes for next pool
			if roll >= addLine {
				successes++
			}
		}

		allRolls = append(allRolls, rolls)
		currentPool = successes

		// Last roll uses highest value, others use sides value
		if currentPool == 0 && len(rolls) > 0 {
			// Last roll - use the highest value
			maxRoll := 0
			for _, roll := range rolls {
				if roll > maxRoll {
					maxRoll = roll
				}
			}
			totalScore += maxRoll
		} else {
			// Not the last roll - each success adds sides value
			totalScore += successes * sides
		}
	}

	// 构建原始表达式
	expr := fmt.Sprintf("%dc%d", initial, addLine)
	if e.Sides != nil && sides != 10 {
		expr += fmt.Sprintf("m%d", sides)
	}

	// 创建合并的骰子结果
	allDiceResults := make([]int, 0)
	for _, round := range allRolls {
		allDiceResults = append(allDiceResults, round...)
	}

	// 使用统一的输出格式
	fmt.Fprintln(os.Stdout, formatDiceOutput(expr, allDiceResults, totalScore))

	return totalScore
}

// ArrayExpr represents an array of expressions [A,B,C,...]
type ArrayExpr struct {
	Elements []Expr
}

func (e *ArrayExpr) Evaluate() interface{} {
	if len(e.Elements) == 0 {
		return 0
	}

	// By default, return the last element's value
	return e.Elements[len(e.Elements)-1].Evaluate()
}

func (e *ArrayExpr) EvaluateMulti() []int {
	results := make([]int, len(e.Elements))
	for i, elem := range e.Elements {
		switch v := elem.Evaluate().(type) {
		case int:
			results[i] = v
		case bool:
			if v {
				results[i] = 1
			} else {
				results[i] = 0
			}
		default:
			results[i] = 0
		}
	}
	return results
}

// HighLowSelectExpr represents kh/kl/dh/dl operators
type HighLowSelectExpr struct {
	Expr     Expr
	Count    Expr
	KeepHigh bool // true for kh/dh, false for kl/dl
	KeepLeft bool // true for kh/kl, false for dh/dl
}

func (e *HighLowSelectExpr) Evaluate() interface{} {
	var values []int

	// Try to get multiple values first
	if multiExpr, ok := e.Expr.(MultiValueExpr); ok {
		values = multiExpr.EvaluateMulti()
	} else {
		// Otherwise, treat it as a single value
		values = []int{e.Expr.Evaluate().(int)}
	}

	count := 1
	if e.Count != nil {
		count = e.Count.Evaluate().(int)
	}

	if count > len(values) {
		count = len(values)
	}

	// Make a copy to avoid modifying the original
	sortedValues := make([]int, len(values))
	copy(sortedValues, values)

	// Sort based on keep high/low
	if e.KeepHigh {
		sort.Sort(sort.Reverse(sort.IntSlice(sortedValues)))
	} else {
		sort.Ints(sortedValues)
	}

	// Select values based on keep left/right
	selected := sortedValues
	if e.KeepLeft {
		selected = sortedValues[:count]
	} else {
		selected = sortedValues[len(sortedValues)-count:]
	}

	// Sum the selected values
	sum := 0
	for _, v := range selected {
		sum += v
	}

	// Output the result
	op := ""
	if e.KeepHigh {
		if e.KeepLeft {
			op = "kh"
		}
	} else {
		if e.KeepLeft {
			op = "kl"
		}
	}

	// 使用统一的输出格式
	exprStr := fmt.Sprintf("%s%s%d", formatExpr(e.Expr), op, count)
	fmt.Fprintln(os.Stdout, formatDiceOutput(exprStr, selected, sum))

	return sum
}

// 添加一个辅助函数用于获取原始表达式
func formatExpr(expr Expr) string {
	switch e := expr.(type) {
	case *DiceExpr:
		count, _ := toInt(e.Count.Evaluate())
		sides, _ := toInt(e.Sides.Evaluate())
		return fmt.Sprintf("%dd%d", count, sides)
	case *ArrayExpr:
		return "array"
	default:
		return "expr"
	}
}

// MaxMinExpr represents max/min constraints
type MaxMinExpr struct {
	Expr  Expr
	Limit Expr
	IsMax bool // true for max, false for min
}

func (e *MaxMinExpr) Evaluate() interface{} {
	val := e.Expr.Evaluate().(int)
	limit := e.Limit.Evaluate().(int)

	op := "max"
	if !e.IsMax {
		op = "min"
	}

	result := val
	if e.IsMax && val > limit {
		result = limit
	} else if !e.IsMax && val < limit {
		result = limit
	}

	// 使用统一的输出格式
	fmt.Fprintln(os.Stdout, fmt.Sprintf("%d %s %d = %d", val, op, limit, result))

	return result
}

// SliceExpr represents the sp operator for array slicing
type SliceExpr struct {
	Array     Expr
	SliceSpec Expr
}

func (e *SliceExpr) Evaluate() interface{} {
	var array []int

	// Try to get multiple values
	if multiExpr, ok := e.Array.(MultiValueExpr); ok {
		array = multiExpr.EvaluateMulti()
	} else {
		// Treat as a single value
		array = []int{e.Array.Evaluate().(int)}
	}

	var start, end, step int

	// Parse the slice specification
	switch spec := e.SliceSpec.(type) {
	case *NumberExpr:
		// Single index
		index := spec.Value
		if index >= 1 && index <= len(array) {
			return array[index-1]
		}
		return 0
	case *ArrayExpr:
		elems := spec.EvaluateMulti()
		if len(elems) == 1 {
			// Single index
			index := elems[0]
			if index >= 1 && index <= len(array) {
				return array[index-1]
			}
			return 0
		} else if len(elems) == 2 {
			// Start:End slice
			start = elems[0]
			end = elems[1]
			step = 1
		} else if len(elems) >= 3 {
			// Start:Step:End slice
			start = elems[0]
			step = elems[1]
			end = elems[2]
		}
	default:
		// Default to a single index
		index := e.SliceSpec.Evaluate().(int)
		if index >= 1 && index <= len(array) {
			return array[index-1]
		}
		return 0
	}

	// Adjust indices to be 0-based and handle bounds
	if start < 1 {
		start = 1
	}
	start--

	if end > len(array) {
		end = len(array)
	}

	// Create the slice
	sliced := make([]int, 0)
	for i := start; (step > 0 && i < end) || (step < 0 && i > end); i += step {
		if i >= 0 && i < len(array) {
			sliced = append(sliced, array[i])
		}
	}

	return &ArrayExpr{
		Elements: intSliceToExprSlice(sliced),
	}
}

func (e *SliceExpr) EvaluateMulti() []int {
	result := e.Evaluate()
	if arr, ok := result.(*ArrayExpr); ok {
		return arr.EvaluateMulti()
	}
	return []int{result.(int)}
}

// ProjectionExpr represents the tp operator
type ProjectionExpr struct {
	Expr Expr
}

func (e *ProjectionExpr) Evaluate() interface{} {
	// Forced projection to array type
	if multiExpr, ok := e.Expr.(MultiValueExpr); ok {
		values := multiExpr.EvaluateMulti()
		return &ArrayExpr{
			Elements: intSliceToExprSlice(values),
		}
	}

	// Otherwise create a single-element array
	return &ArrayExpr{
		Elements: []Expr{
			&NumberExpr{Value: e.Expr.Evaluate().(int)},
		},
	}
}

func (e *ProjectionExpr) EvaluateMulti() []int {
	if multiExpr, ok := e.Expr.(MultiValueExpr); ok {
		return multiExpr.EvaluateMulti()
	}
	return []int{e.Expr.Evaluate().(int)}
}

// BitwiseExpr represents the &/| operators
type BitwiseExpr struct {
	Left  Expr
	Right Expr
	IsAnd bool
}

func (e *BitwiseExpr) Evaluate() interface{} {
	left := e.Left.Evaluate().(int)
	right := e.Right.Evaluate().(int)

	var result int
	if e.IsAnd {
		result = left & right
		fmt.Printf("%d & %d = %d\n", left, right, result)
	} else {
		result = left | right
		fmt.Printf("%d | %d = %d\n", left, right, result)
	}

	return result
}

// CompareExpr represents the >/< operators
type CompareExpr struct {
	Left      Expr
	Right     Expr
	IsGreater bool
}

func (e *CompareExpr) Evaluate() interface{} {
	left := e.Left.Evaluate().(int)
	right := e.Right.Evaluate().(int)

	var result bool
	if e.IsGreater {
		result = left > right
		fmt.Printf("%d > %d = %t\n", left, right, result)
	} else {
		result = left < right
		fmt.Printf("%d < %d = %t\n", left, right, result)
	}

	if result {
		return 1
	}
	return 0
}

// TernaryExpr represents the ternary conditional operator A?B:C
type TernaryExpr struct {
	Condition Expr
	TrueExpr  Expr
	FalseExpr Expr
}

func (e *TernaryExpr) Evaluate() interface{} {
	condValue := e.Condition.Evaluate()

	var result bool
	switch v := condValue.(type) {
	case int:
		result = v != 0
	case bool:
		result = v
	default:
		result = false
	}

	if result {
		return e.TrueExpr.Evaluate()
	} else {
		return e.FalseExpr.Evaluate()
	}
}

// Helper functions
func formatInts(rolls []int) string {
	strRolls := make([]string, len(rolls))
	for i, r := range rolls {
		if r == -1 { // Separator in dice results
			strRolls[i] = "|"
		} else {
			strRolls[i] = strconv.Itoa(r)
		}
	}
	return strings.Join(strRolls, " ")
}

func intSliceToExprSlice(ints []int) []Expr {
	exprs := make([]Expr, len(ints))
	for i, v := range ints {
		exprs[i] = &NumberExpr{Value: v}
	}
	return exprs
}

type Expr interface {
	Evaluate() interface{}
}

type Stmt interface {
	Evaluate() interface{}
}

type NumberExpr struct {
	Value int
}

func (e *NumberExpr) Evaluate() interface{} {
	return e.Value
}

type ExprListStmt struct {
	Exprs []Expr
}

type RepeatBlockStmt struct {
	Count Expr
	Block []Stmt
}

type BinaryExpr struct {
	Left  Expr
	Op    int
	Right Expr
}

func (e *BinaryExpr) Evaluate() interface{} {
	left := e.Left.Evaluate()
	right := e.Right.Evaluate()

	switch e.Op {
	case ADD:
		if lStr, ok := left.(string); ok {
			if rStr, ok := right.(string); ok {
				return lStr + rStr
			}
		}
		return left.(int) + right.(int)
	case SUB:
		if lStr, ok := left.(string); ok {
			if rStr, ok := right.(string); ok {
				return strings.Replace(lStr, rStr, "", 1)
			}
		}
		return left.(int) - right.(int)
	case MUL:
		if lStr, ok := left.(string); ok {
			if rStr, ok := right.(string); ok {
				return longestCommonSubstring(lStr, rStr)
			}
		}
		return left.(int) * right.(int)
	case DIV:
		if lStr, ok := left.(string); ok {
			if rStr, ok := right.(string); ok {
				return strings.ReplaceAll(lStr, rStr, "")
			}
		}
		if right.(int) == 0 {
			fmt.Println("Division by zero")
			return 0
		}
		return left.(int) / right.(int)
	case MOD:
		if right.(int) == 0 {
			fmt.Println("Modulo by zero")
			return 0
		}
		return left.(int) % right.(int)
	case CIRCUMFLEX:
		return int(math.Pow(float64(left.(int)), float64(right.(int))))
	}
	return 0
}

func longestCommonSubstring(s1, s2 string) string {
	m := len(s1)
	n := len(s2)
	longest := 0
	end := 0

	table := make([][]int, m+1)
	for i := range table {
		table[i] = make([]int, n+1)
	}

	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if s1[i-1] == s2[j-1] {
				table[i][j] = table[i-1][j-1] + 1
				if table[i][j] > longest {
					longest = table[i][j]
					end = i
				}
			}
		}
	}
	return s1[end-longest : end]
}

type UnaryExpr struct {
	Op   int
	Expr Expr
}

func (e *UnaryExpr) Evaluate() interface{} {
	value := e.Expr.Evaluate().(int)
	switch e.Op {
	case SUB:
		return -value
	}
	return value
}

// DiceExpr represents a standard dice roll (e.g., 3d6)
type DiceExpr struct {
	Count   Expr
	Sides   Expr
	Drop    Expr
	Keep    Expr
	process string // 保存投掷过程
	rolls   []int  // 保存骰子结果
}

func (d *DiceExpr) GetProcess() string {
	return d.process
}

func (d *DiceExpr) Evaluate() interface{} {
	countVal := EvaluateToInt(d.Count)
	sidesVal := EvaluateToInt(d.Sides)

	// 安全检查
	if countVal <= 0 || countVal > MAX_ROLLS {
		return 0
	}
	if sidesVal < MIN_DICE_SIDES || sidesVal > MAX_DICE_SIDES {
		return 0
	}

	// 投掷骰子
	d.rolls = make([]int, countVal)
	sum := 0
	for i := 0; i < countVal; i++ {
		roll := rand.Intn(sidesVal) + 1
		d.rolls[i] = roll
		sum += roll
	}

	// 记录投掷过程
	d.process = fmt.Sprintf("%dd%d = %s = %d", countVal, sidesVal, formatRollsCompact(d.rolls), sum)

	return sum

}

func (d *DiceExpr) EvaluateMulti() []int {
	// 如果 Evaluate 已经被调用并生成了结果
	if d.rolls != nil {
		return d.rolls
	}

	// 否则重新计算
	count := EvaluateToInt(d.Count)
	if count > MAX_ROLLS {
		fmt.Println(fmt.Sprintf("Exceeded maximum rolls: %d", MAX_ROLLS))
		return []int{0}
	}

	sides := EvaluateToInt(d.Sides)
	if sides < MIN_DICE_SIDES {
		fmt.Println(fmt.Sprintf("Minimum dice sides: %d", MIN_DICE_SIDES))
		return []int{0}
	} else if sides > MAX_DICE_SIDES {
		fmt.Println(fmt.Sprintf("Exceeded maximum dice sides: %d", MAX_DICE_SIDES))
		return []int{0}
	}

	d.rolls = make([]int, count)
	for i := 0; i < count; i++ {
		d.rolls[i] = rand.Intn(sides) + 1
	}
	return d.rolls
}

// Make DiceExpr implement MultiValueExpr
var _ MultiValueExpr = (*DiceExpr)(nil)

type HighestDiceExpr struct {
	Count Expr
	Sides Expr
	Keep  Expr
}

func (e *HighestDiceExpr) Evaluate() interface{} {
	count := e.Count.Evaluate().(int)
	sides := e.Sides.Evaluate().(int)
	keep := e.Keep.Evaluate().(int)

	if count > MAX_ROLLS {
		fmt.Println(fmt.Sprintf("Exceeded maximum rolls: %d", MAX_ROLLS))
		return 0
	}
	if sides < MIN_DICE_SIDES || sides > MAX_DICE_SIDES {
		fmt.Println(fmt.Sprintf("Invalid number of sides: %d", sides))
		return 0
	}
	if keep > count {
		fmt.Println(fmt.Sprintf("Cannot keep more dice than rolled"))
		return 0
	}

	rolls := make([]int, count)
	for i := 0; i < count; i++ {
		rolls[i] = rand.Intn(sides) + 1
	}

	sort.Sort(sort.Reverse(sort.IntSlice(rolls)))
	sum := 0
	for i := 0; i < keep; i++ {
		sum += rolls[i]
	}

	keptRolls := rolls[:keep]
	discardedRolls := rolls[keep:]
	allRolls := append(keptRolls, append([]int{-1}, discardedRolls...)...)
	fmt.Printf("%dd%dh%d = [%s] = [%s] = %d\n",
		count, sides, keep,
		formatRolls(allRolls),
		formatRolls(keptRolls),
		sum)
	return sum
}

func formatRolls(rolls []int) string {
	strRolls := make([]string, len(rolls))
	for i, roll := range rolls {
		if roll == -1 {
			strRolls[i] = "|"
		} else {
			strRolls[i] = strconv.Itoa(roll)
		}
	}

	if len(strRolls) > 10 {
		middle := strings.Join(strRolls[5:len(strRolls)-5], " ")
		if strings.Contains(middle, "|") {
			return strings.Join(strRolls[:5], " ") + " ..|.. " + strings.Join(strRolls[len(strRolls)-5:], " ")
		}
		return strings.Join(strRolls[:5], " ") + " ..... " + strings.Join(strRolls[len(strRolls)-5:], " ")
	}
	return strings.Join(strRolls, " ")
}

type LowestDiceExpr struct {
	Count Expr
	Sides Expr
	Keep  Expr
}

func (e *LowestDiceExpr) Evaluate() interface{} {
	count := e.Count.Evaluate().(int)
	sides := e.Sides.Evaluate().(int)
	keep := e.Keep.Evaluate().(int)

	if count > MAX_ROLLS {
		fmt.Println(fmt.Sprintf("Exceeded maximum rolls: %d", MAX_ROLLS))
		return 0
	}
	if sides < MIN_DICE_SIDES || sides > MAX_DICE_SIDES {
		fmt.Println(fmt.Sprintf("Invalid number of sides: %d", sides))
		return 0
	}
	if keep > count {
		fmt.Println(fmt.Sprintf("Cannot keep more dice than rolled"))
		return 0
	}

	rolls := make([]int, count)
	for i := 0; i < count; i++ {
		rolls[i] = rand.Intn(sides) + 1
	}

	sort.Ints(rolls)
	sum := 0
	for i := 0; i < keep; i++ {
		sum += rolls[i]
	}

	keptRolls := rolls[:keep]
	discardedRolls := rolls[keep:]
	allRolls := append(keptRolls, append([]int{-1}, discardedRolls...)...)
	fmt.Printf("%dd%dl%d = [%s] = [%s] = %d\n",
		count, sides, keep,
		formatRolls(allRolls),
		formatRolls(keptRolls),
		sum)
	return sum
}

type ComparisonExpr struct {
	Left  Expr
	Op    int
	Right Expr
}

func (e *ComparisonExpr) Evaluate() interface{} {
	left := e.Left.Evaluate().(int)
	right := e.Right.Evaluate().(int)

	switch e.Op {
	case EQ:
		return left == right
	case NEQ:
		return left != right
	case GT:
		return left > right
	case LT:
		return left < right
	case GE:
		return left >= right
	case LE:
		return left <= right
	}
	return false
}

type ExprStmt struct {
	Expr Expr
}

func (e *ExprStmt) Evaluate() interface{} {
	return e.Expr.Evaluate()
}

func (e *ExprStmt) GetProcess() string {
	// 尝试获取骰子表达式的过程
	switch expr := e.Expr.(type) {
	case *DiceExpr:
		return expr.GetProcess()
	// 为其他骰子表达式类型添加 case
	// case *FateDiceExpr:
	//     return expr.GetProcess()
	// ... 其他类型
	default:
		// 对于不是骰子表达式的情况，返回结果的字符串表示
		return fmt.Sprintf("%v", e.Expr.Evaluate())
	}
}

type AssignStmt struct {
	Name  string
	Value Expr
}

func (s *AssignStmt) Evaluate() interface{} {
	value := s.Value.Evaluate()
	variables[s.Name] = value
	fmt.Printf("%s = %v\n", s.Name, value)
	return value
}

type IdentExpr struct {
	Name string
}

func (e *IdentExpr) Evaluate() interface{} {
	if value, ok := variables[e.Name]; ok {
		return value
	}
	fmt.Println(fmt.Sprintf("Undefined variable: %s", e.Name))
	return 0
}

var variables = make(map[string]interface{})

// toInt 尝试将 interface{} 值转换为 int
func toInt(val interface{}) (int, bool) {
	switch v := val.(type) {
	case int:
		return v, true
	case float64:
		return int(v), true
	case bool:
		if v {
			return 1, true
		}
		return 0, true
	case string:
		if n, err := strconv.Atoi(v); err == nil {
			return n, true
		}
	}
	return 0, false
}

// formatRollsCompact 格式化骰子结果，对于长结果只显示头尾三个
func formatRollsCompact(rolls []int) string {
	if len(rolls) <= 7 {
		// 如果结果较少，全部显示
		strRolls := make([]string, len(rolls))
		for i, r := range rolls {
			strRolls[i] = strconv.Itoa(r)
		}
		return "[" + strings.Join(strRolls, " ") + "]"
	} else {
		// 如果结果较多，只显示头尾三个
		head := make([]string, 3)
		tail := make([]string, 3)
		for i := 0; i < 3; i++ {
			head[i] = strconv.Itoa(rolls[i])
			tail[i] = strconv.Itoa(rolls[len(rolls)-3+i])
		}
		return "[" + strings.Join(head, " ") + " ... " + strings.Join(tail, " ") + "]"
	}
}

// 统一格式化骰子表达式输出
func formatDiceOutput(expr string, results []int, total int) string {
	return fmt.Sprintf("%s = %s = %d", expr, formatRollsCompact(results), total)
}
