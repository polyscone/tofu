package i18n

const (
	KindUnknown TokenKind = iota
	KindUnexpected
	KindEOF
	KindEnterExpr
	KindLeaveExpr
	KindInt
	KindFloat
	KindString
	KindText
	KindIdent
	KindOr
	KindAnd
	KindBang
	KindComma
	KindColon
	KindLParen
	KindRParen
	KindLBrack
	KindRBrack
	KindPlus
	KindMinus
	KindAsterisk
	KindSlash
	KindPercent
	KindEqual
	KindEqualEqual
	KindBangEqual
	KindGreater
	KindGreaterEqual
	KindLess
	KindLessEqual
	KindEqualGreater
)

var operators = map[string]TokenKind{
	"or":  KindOr,
	"and": KindAnd,
	"!":   KindBang,
	",":   KindComma,
	":":   KindColon,
	"(":   KindLParen,
	")":   KindRParen,
	"[":   KindLBrack,
	"]":   KindRBrack,
	"+":   KindPlus,
	"-":   KindMinus,
	"*":   KindAsterisk,
	"/":   KindSlash,
	"%":   KindPercent,
	"=":   KindEqual,
	"==":  KindEqualEqual,
	"!=":  KindBangEqual,
	">":   KindGreater,
	">=":  KindGreaterEqual,
	"<":   KindLess,
	"<=":  KindLessEqual,
	"=>":  KindEqualGreater,
}

var names = [...]string{
	KindUnknown:      "unknown",
	KindUnexpected:   "unexpected",
	KindEOF:          "eof",
	KindEnterExpr:    "enter expr",
	KindLeaveExpr:    "leave expr",
	KindInt:          "int",
	KindFloat:        "float",
	KindString:       "string",
	KindText:         "text",
	KindIdent:        "identifier",
	KindOr:           "or",
	KindAnd:          "and",
	KindBang:         "!",
	KindComma:        ",",
	KindColon:        ":",
	KindLParen:       "(",
	KindRParen:       ")",
	KindLBrack:       "[",
	KindRBrack:       "]",
	KindPlus:         "+",
	KindMinus:        "-",
	KindAsterisk:     "*",
	KindSlash:        "/",
	KindPercent:      "%",
	KindEqual:        "=",
	KindEqualEqual:   "==",
	KindBangEqual:    "!=",
	KindGreater:      ">",
	KindGreaterEqual: ">=",
	KindLess:         "<",
	KindLessEqual:    "<=",
	KindEqualGreater: "=>",
}

type TokenKind int

func (k TokenKind) String() string {
	return names[k]
}

type Token struct {
	Kind   TokenKind
	Lexeme string
}
