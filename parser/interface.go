package parser

// LexerWrapper 包装 Lexer 并实现 yyLexer 接口
type LexerWrapper struct {
	lexer *Lexer
}

// NewLexerWrapper 创建一个新的词法分析器包装器
func NewLexerWrapper(l *Lexer) *LexerWrapper {
	return &LexerWrapper{lexer: l}
}

// Error 实现 yyLexer 接口的 Error 方法
func (l *LexerWrapper) Error(s string) {
	l.lexer.Error(s)
}

// Lex 实现 yyLexer 接口的 Lex 方法
func (l *LexerWrapper) Lex(lval *yySymType) int {
	// 直接传递，因为我们已经定义了类型别名
	return l.lexer.Lex(lval)
}

// NewParser 创建一个新的解析器
func NewParser() *yyParserImpl {
	return &yyParserImpl{}
}

var lastResult map[string]interface{}
var lastProcess string // 新增：记录骰子投掷过程

// SetResult 设置最后一次计算的结果
func SetResult(result interface{}, process string) {
	lastResult = map[string]interface{}{
		"结果": result,
		"过程": process,
	}
}

// GetResult 获取最后一次计算的结果
func GetResult() map[string]interface{} {
	return lastResult
}

// Parse 函数是 yyParse 的包装器
func Parse(lexer yyLexer) int {
	// 重置结果
	lastResult = nil
	lastProcess = "" // 重置过程
	return yyParse(lexer)
}
