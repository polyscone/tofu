package i18n

import (
	"fmt"
	"io"
	"slices"
	"strings"
)

type Error struct {
	Token   Token
	Message string
}

type Errors []Error

func (e Errors) Error() string {
	var sb strings.Builder

	for _, err := range e {
		sb.WriteString(fmt.Sprintf("%v\n", err.Message))
	}

	return strings.TrimSpace(sb.String())
}

const (
	powerNone = iota
	powerLogicalTerm
	powerLogicalFactor
	powerComparison
	powerTerm
	powerFactor
	powerSlice
	powerSelect
	powerUnary
	powerCall
)

type Rule struct {
	nud   func() Node
	led   func(Node) Node
	power int
}

type Parser struct {
	scanner *Scanner
	curr    Token
	next    Token
	errs    Errors
	rules   map[TokenKind]Rule
}

func NewParser() *Parser {
	p := &Parser{scanner: NewScanner()}

	p.rules = map[TokenKind]Rule{
		KindInt:    {nud: p.parseLiteral},
		KindFloat:  {nud: p.parseLiteral},
		KindString: {nud: p.parseLiteral},

		KindIdent: {nud: p.parseIdent},

		KindBang: {nud: p.parseUnary},

		KindOr: {led: p.parseBinary, power: powerLogicalTerm},

		KindAnd: {led: p.parseBinary, power: powerLogicalFactor},

		KindEqualEqual:   {led: p.parseBinary, power: powerComparison},
		KindBangEqual:    {led: p.parseBinary, power: powerComparison},
		KindGreater:      {led: p.parseBinary, power: powerComparison},
		KindGreaterEqual: {led: p.parseBinary, power: powerComparison},
		KindLess:         {led: p.parseBinary, power: powerComparison},
		KindLessEqual:    {led: p.parseBinary, power: powerComparison},

		KindPlus:  {nud: p.parseUnary, led: p.parseBinary, power: powerTerm},
		KindMinus: {nud: p.parseUnary, led: p.parseBinary, power: powerTerm},

		KindAsterisk: {led: p.parseBinary, power: powerFactor},
		KindSlash:    {led: p.parseBinary, power: powerFactor},
		KindPercent:  {led: p.parseBinary, power: powerFactor},

		KindColon: {nud: p.parseSliceImplicitStart, led: p.parseBinary, power: powerSlice},

		KindEqualGreater: {led: p.parseSelect, power: powerSelect},

		KindLBrack: {led: p.parseIndexOrSlice, power: powerCall},
		KindLParen: {nud: p.parseGroup, led: p.parseCall, power: powerCall},
	}

	return p
}

func (p *Parser) load(src io.ByteScanner) {
	p.scanner.Load(src)

	p.curr = Token{}
	p.next = Token{}
	p.errs = nil

	p.consume()
	p.consume()
}

func (p *Parser) Parse(src io.ByteScanner) (*RootNode, error) {
	p.load(src)

	var root RootNode
	for !p.match(KindEOF) {
		root.Fragments = append(root.Fragments, p.parseFragment())
	}

	if p.errs != nil {
		return &root, p.errs
	}

	return &root, nil
}

func (p *Parser) parseText() *LiteralNode {
	text := p.consume(KindText)

	return &LiteralNode{Token: text}
}

func (p *Parser) parseLiteral() Node {
	tok := p.consume(KindInt, KindFloat, KindString)

	return &LiteralNode{Token: tok}
}

func (p *Parser) parseIdent() Node {
	ident := p.consume(KindIdent)

	return &IdentNode{Name: ident}
}

func (p *Parser) parseUnary() Node {
	op := p.consume(KindPlus, KindMinus, KindBang)
	rhs := p.parseExpr(powerUnary)

	return &UnaryNode{
		Op:  op,
		RHS: rhs,
	}
}

func (p *Parser) parseBinary(lhs Node) Node {
	isBinary := p.match(
		KindOr, KindAnd,
		KindPlus, KindMinus,
		KindAsterisk, KindSlash, KindPercent,
		KindEqualEqual, KindBangEqual,
		KindGreater, KindGreaterEqual,
		KindLess, KindLessEqual,
		KindColon,
	)
	if !isBinary {
		p.error(p.curr, fmt.Sprintf("expected binary operator, found %q", p.curr.Lexeme))
	}

	op := p.consume()
	rbp := p.rules[op.Kind].power

	var rhs Node
	if op.Kind != KindColon || !p.match(KindRBrack) {
		rhs = p.parseExpr(rbp)
	}

	return &BinaryNode{
		Op:  op,
		LHS: lhs,
		RHS: rhs,
	}
}

func (p *Parser) parseGroup() Node {
	p.consume(KindLParen)

	expr := p.parseExpr(powerNone)

	p.consume(KindRParen)

	return expr
}

func (p *Parser) parseSliceImplicitStart() Node {
	lhs := &LiteralNode{Token: Token{
		Kind:   KindInt,
		Lexeme: "0",
	}}

	return p.parseBinary(lhs)
}

func (p *Parser) parseIndexOrSlice(lhs Node) Node {
	p.consume(KindLBrack)

	expr := p.parseExpr(powerNone)

	p.consume(KindRBrack)

	if bin, ok := expr.(*BinaryNode); ok && bin.Op.Kind == KindColon {
		return &SliceNode{
			LHS:   lhs,
			Start: bin.LHS,
			End:   bin.RHS,
		}
	}

	return &IndexNode{
		LHS:  lhs,
		Expr: expr,
	}
}

func (p *Parser) parseSelect(lhs Node) Node {
	p.consume(KindEqualGreater)
	p.consume(KindLParen)

	var opts []*OptNode
	for !p.match(KindRParen, KindEOF) {
		match := p.parseExpr(powerNone)

		p.consume(KindEqual)

		result := p.parseExpr(powerNone)

		opts = append(opts, &OptNode{
			Match:  match,
			Result: result,
		})

		if !p.match(KindRParen) {
			p.consume(KindComma)
		}
	}

	p.consume(KindRParen)

	return &SelectNode{
		Value: lhs,
		Opts:  opts,
	}
}

func (p *Parser) parseCall(lhs Node) Node {
	p.consume(KindLParen)

	var args []Node
	for !p.match(KindRParen, KindEOF) {
		args = append(args, p.parseExpr(powerNone))

		if !p.match(KindRParen) {
			p.consume(KindComma)
		}
	}

	p.consume(KindRParen)

	return &CallNode{
		LHS:  lhs,
		Args: args,
	}
}

func (p *Parser) parseExpr(rbp int) Node {
	rule := p.rules[p.curr.Kind]
	if rule.nud == nil {
		return p.parseInvalidUntil(KindEOF)
	}

	lhs := rule.nud()
	rule = p.rules[p.curr.Kind]
	power := rule.power

	for !p.match(KindEOF) && rbp < power {
		if rule.led == nil {
			return lhs
		}

		lhs = rule.led(lhs)
		rule = p.rules[p.curr.Kind]
	}

	return lhs
}

func (p *Parser) parseExprFragment() Node {
	p.consume(KindEnterExpr)

	expr := p.parseExpr(powerNone)

	p.consume(KindLeaveExpr)

	return expr
}

func (p *Parser) parseFragment() Node {
	switch p.curr.Kind {
	case KindText:
		return p.parseText()

	case KindEnterExpr:
		return p.parseExprFragment()

	default:
		return p.parseInvalidUntil(KindEOF)
	}
}

func (p *Parser) parseInvalidUntil(stop ...TokenKind) *InvalidNode {
	start := p.consume()

	p.error(start, fmt.Sprintf("unexpected %q", start.Lexeme))
	p.consumeUntil(stop...)

	end := p.curr

	return &InvalidNode{
		Start: start,
		End:   end,
	}
}

func (p *Parser) error(tok Token, message string) {
	p.errs = append(p.errs, Error{Token: tok, Message: message})
}

func (p *Parser) match(kinds ...TokenKind) bool {
	return slices.Contains(kinds, p.curr.Kind)
}

func (p *Parser) peek(kinds ...TokenKind) bool {
	return slices.Contains(kinds, p.next.Kind)
}

func (p *Parser) expect(expected ...TokenKind) bool {
	match := p.match(expected...)
	if !match {
		names := make([]string, len(expected))
		for i, kind := range expected {
			names[i] = kind.String()
		}

		lexeme := p.curr.Lexeme
		if p.curr.Kind == KindEOF {
			lexeme = "eof"
		}

		p.error(p.curr, fmt.Sprintf("expected %v, found %q", orList(names), lexeme))
	}

	return match
}

func (p *Parser) consume(expected ...TokenKind) Token {
	if expected != nil {
		p.expect(expected...)
	}

	tok, err := p.scanner.Consume()
	if err != nil {
		p.error(tok, err.Error())
	}

	prev := p.curr
	p.curr = p.next
	p.next = tok

	return prev
}

func (p *Parser) consumeUntil(stop ...TokenKind) Token {
	if len(stop) == 0 {
		panic("consume until must be called with at least one target token")
	}

	for !p.match(KindEOF) {
		if p.match(stop...) {
			break
		}

		p.consume()
	}

	return p.curr
}

func orList(strs []string) string {
	quotes := make([]string, len(strs))
	for i, str := range strs {
		quotes[i] = fmt.Sprintf("%q", str)
	}

	switch n := len(quotes); n {
	case 0:
		return ""

	case 1:
		return quotes[0]

	case 2:
		return strings.Join(quotes, " or ")

	default:
		first, last := quotes[:n-1], quotes[n-1]

		return strings.Join(first, ", ") + " or " + last
	}
}
