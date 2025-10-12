package parser

// YySymType represents the semantic value for the parser
type YySymType struct {
	yys      int
	Num      int
	Str      string
	StmtList []Stmt
	Stmt     Stmt
	Expr     Expr
}

type yySymType = YySymType

// 辅助函数，将表达式的值转为整数
func EvaluateToInt(expr Expr) int {
	if expr == nil {
		return 0
	}

	result := expr.Evaluate()

	switch v := result.(type) {
	case int:
		return v
	case float64:
		return int(v)
	default:
		return 0
	}
}
