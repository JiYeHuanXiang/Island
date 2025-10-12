%{
package parser

import "fmt" 

var _ = fmt.Println
%}

%token <Num> NUMBER
%token <Str> STRING IDENT

// Define token for parentheses
%token LPAREN RPAREN

// 骰子操作符
%token D H L F A C P B K Q M

// 特殊操作符
%token MAX MIN

// 基本操作符
%token ADD SUB MUL DIV MOD RPAREN LBRACKET RBRACKET BITAND BITOR
%token COMMA GT LT GE LE EQ NEQ QUESTION COLON SEMICOLON HASH LBRACE RBRACE CIRCUMFLEX ASSIGN

// 组合操作符 kh, kl
%token KL KH

%left GT LT GE LE EQ NEQ
%left BITAND BITOR
%left ADD SUB
%left MUL DIV MOD
%right CIRCUMFLEX
%left MAX MIN KL KH
%left D F A C P B
%right QUESTION COLON
%right UMINUS

%type <StmtList> program stmtlist
%type <Stmt> stmt
%type <Expr> expr term factor dice_expr fate_expr bonus_expr infinite_pool_expr double_cross_expr array_expr array_item array_items high_low_expr
%type <Expr> bitwise_expr compare_expr condition_expr additive_expr mult_expr power_expr special_expr unary_expr

%start program

%%

program: stmtlist
    {
        var result interface{}
        var process string = ""
        for _, stmt := range $1 {
            if stmt != nil {
                // 获取表达式结果
                result = stmt.Evaluate()
                
                // 获取投掷过程
                if exprStmt, ok := stmt.(*ExprStmt); ok {
                    process = exprStmt.GetProcess()
                }
            }
        }
        SetResult(result, process)  // 保存计算结果和过程
        $$ = $1
    }
;

stmtlist: stmt
    {
        $$ = []Stmt{$1}
    }
    | stmtlist SEMICOLON stmt
    {
        $$ = append($1, $3)
    }
;

stmt: expr
    {
        $$ = &ExprStmt{$1}
    }
;

expr: condition_expr
    {
        $$ = $1
    }
;

condition_expr: bitwise_expr
    {
        $$ = $1
    }
    | expr QUESTION expr COLON expr
    {
        $$ = &TernaryExpr{$1, $3, $5}
    }
;

bitwise_expr: compare_expr
    {
        $$ = $1
    }
    | bitwise_expr BITAND compare_expr
    {
        $$ = &BitwiseExpr{$1, $3, true}
    }
    | bitwise_expr BITOR compare_expr
    {
        $$ = &BitwiseExpr{$1, $3, false}
    }
;

compare_expr: additive_expr
    {
        $$ = $1
    }
    | compare_expr GT additive_expr
    {
        $$ = &CompareExpr{$1, $3, true}
    }
    | compare_expr LT additive_expr
    {
        $$ = &CompareExpr{$1, $3, false}
    }
;

additive_expr: mult_expr
    {
        $$ = $1
    }
    | additive_expr ADD mult_expr
    {
        $$ = &BinaryExpr{$1, ADD, $3}
    }
    | additive_expr SUB mult_expr
    {
        $$ = &BinaryExpr{$1, SUB, $3}
    }
;

mult_expr: power_expr
    {
        $$ = $1
    }
    | mult_expr MUL power_expr
    {
        $$ = &BinaryExpr{$1, MUL, $3}
    }
    | mult_expr DIV power_expr
    {
        $$ = &BinaryExpr{$1, DIV, $3}
    }
    | mult_expr MOD power_expr
    {
        $$ = &BinaryExpr{$1, MOD, $3}
    }
;

power_expr: special_expr
    {
        $$ = $1
    }
    | power_expr CIRCUMFLEX special_expr
    {
        $$ = &BinaryExpr{$1, CIRCUMFLEX, $3}
    }
;

special_expr: term
    {
        $$ = $1
    }
    | special_expr MAX term
    {
        $$ = &MaxMinExpr{$1, $3, true}
    }
    | special_expr MIN term
    {
        $$ = &MaxMinExpr{$1, $3, false}
    }
;

term: dice_expr 
    { 
        $$ = $1
    }
    | fate_expr 
    { 
        $$ = $1
    }
    | bonus_expr 
    { 
        $$ = $1
    }
    | infinite_pool_expr 
    { 
        $$ = $1
    }
    | double_cross_expr 
    { 
        $$ = $1
    }
    | high_low_expr 
    { 
        $$ = $1
    }
    | unary_expr
    { 
        $$ = $1
    }
    | term KH factor
    {
        $$ = &HighLowSelectExpr{$1, $3, true, true}
    }
    | term KL factor
    {
        $$ = &HighLowSelectExpr{$1, $3, false, true}
    }
;

dice_expr: factor D factor
    {
        fmt.Printf("解析骰点表达式: %v d %v\n", $1, $3)
        $$ = &DiceExpr{Count: $1, Sides: $3, Drop: nil, Keep: nil, process: "", rolls: nil}
    }
    | dice_expr D factor
    {
        fmt.Printf("解析连续骰点表达式: %v d %v\n", $1, $3)
        $$ = &DiceExpr{Count: $1, Sides: $3, Drop: nil, Keep: nil, process: "", rolls: nil}
    }
    | factor D factor A factor
    {
        $$ = &DiceExpr{Count: $1, Sides: $3, Drop: $5, Keep: nil, process: "", rolls: nil}
    }
    | factor D factor K factor
    {
        $$ = &DiceExpr{Count: $1, Sides: $3, Drop: nil, Keep: $5, process: "", rolls: nil}
    }
    | factor D factor Q factor
    {
        $$ = &DiceExpr{Count: $1, Sides: $3, Drop: nil, Keep: nil, process: "", rolls: nil}
    }
    | factor D factor P factor
    {
        $$ = &PenaltyBonusDiceExpr{IsBonus: false, Count: $5}
    }
    | factor D factor B factor
    {
        $$ = &PenaltyBonusDiceExpr{IsBonus: true, Count: $5}
    }
;

fate_expr: factor F
    {
        $$ = &FateDiceExpr{Count: $1, process: "", rolls: nil}
    }
    | factor IDENT
    {
        // 处理 df 操作符
        if $2 == "df" {
            $$ = &FateDiceExpr{Count: $1, process: "", rolls: nil}
        } else {
            yylex.Error("非预期的标识符: " + $2)
        }
    }
;

bonus_expr: factor P factor
    {
        $$ = &PenaltyBonusDiceExpr{IsBonus: false, Count: $3}
    }
    | factor B factor
    {
        $$ = &PenaltyBonusDiceExpr{IsBonus: true, Count: $3}
    }
;

infinite_pool_expr: factor A factor
    {
        $$ = &InfinitePoolDiceExpr{
            Initial:            $1,
            AddLine:            $3,
            SuccessLine:        &NumberExpr{Value: 8}, // 默认成功线为8
            ReverseSuccessLine: nil,
            Sides:              &NumberExpr{Value: 10}, // 默认面数为10
        }
    }
    | factor A factor K factor
    {
        $$ = &InfinitePoolDiceExpr{
            Initial:            $1,
            AddLine:            $3,
            SuccessLine:        $5,
            ReverseSuccessLine: nil,
            Sides:              &NumberExpr{Value: 10},
        }
    }
    | factor A factor K factor M factor
    {
        $$ = &InfinitePoolDiceExpr{
            Initial:            $1,
            AddLine:            $3,
            SuccessLine:        $5,
            ReverseSuccessLine: nil,
            Sides:              $7,
        }
    }
    | factor A factor Q factor
    {
        $$ = &InfinitePoolDiceExpr{
            Initial:            $1,
            AddLine:            $3,
            SuccessLine:        &NumberExpr{Value: 8},
            ReverseSuccessLine: $5,
            Sides:              &NumberExpr{Value: 10},
        }
    }
    | factor A factor Q factor M factor
    {
        $$ = &InfinitePoolDiceExpr{
            Initial:            $1,
            AddLine:            $3,
            SuccessLine:        &NumberExpr{Value: 8},
            ReverseSuccessLine: $5,
            Sides:              $7,
        }
    }
    | factor A factor K factor Q factor
    {
        $$ = &InfinitePoolDiceExpr{
            Initial:            $1,
            AddLine:            $3,
            SuccessLine:        $5,
            ReverseSuccessLine: $7,
            Sides:              &NumberExpr{Value: 10},
        }
    }
    | factor A factor K factor Q factor M factor
    {
        $$ = &InfinitePoolDiceExpr{
            Initial:            $1,
            AddLine:            $3,
            SuccessLine:        $5,
            ReverseSuccessLine: $7,
            Sides:              $9,
        }
    }
    | factor A factor M factor
    {
        $$ = &InfinitePoolDiceExpr{
            Initial:            $1,
            AddLine:            $3,
            SuccessLine:        &NumberExpr{Value: 8},
            ReverseSuccessLine: nil,
            Sides:              $5,
        }
    }
;

double_cross_expr: factor C factor
    {
        $$ = &DoubleCrossDiceExpr{
            Initial: $1,
            AddLine: $3,
            Sides:   &NumberExpr{Value: 10}, // 默认面数为10
        }
    }
    | factor C factor M factor
    {
        $$ = &DoubleCrossDiceExpr{
            Initial: $1,
            AddLine: $3,
            Sides:   $5,
        }
    }
;

high_low_expr: factor KH factor
    {
        $$ = &HighLowSelectExpr{
            Expr:     $1,
            Count:    $3,
            KeepHigh: true,
            KeepLeft: true,
        }
    }
    | factor KL factor
    {
        $$ = &HighLowSelectExpr{
            Expr:     $1,
            Count:    $3,
            KeepHigh: false,
            KeepLeft: true,
        }
    }
;

unary_expr: factor
    {
        $$ = $1
    }
;

factor: NUMBER
    {
        $$ = &NumberExpr{$1}
    }
    | IDENT
    {
        $$ = &IdentExpr{$1}
    }
    | LPAREN expr RPAREN
    {
        $$ = $2
    }
    | array_expr
    {
        $$ = $1
    }
;

array_expr: LBRACKET array_items RBRACKET
    {
        if arr, ok := $2.(*ArrayExpr); ok {
            $$ = arr
        } else {
            $$ = &ArrayExpr{[]Expr{$2}}
        }
    }
;

array_items: array_item
    {
        $$ = &ArrayExpr{[]Expr{$1}}
    }
    | array_items COMMA array_item
    {
        if arr, ok := $1.(*ArrayExpr); ok {
            arr.Elements = append(arr.Elements, $3)
            $$ = arr
        } else {
            $$ = &ArrayExpr{[]Expr{$1, $3}}
        }
    }
;

array_item: expr
    {
        $$ = $1
    }
;

%%

func init() {
    // 初始化解析器相关设置
}