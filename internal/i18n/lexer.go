package i18n

import (
	"errors"
	"fmt"
	"io"
	"strings"
)

type predicate func(byte) bool

const eof = 0xFF

var errorEOF = Token{Kind: KindEOF, Lexeme: "lexer error"}

type mode byte

const (
	modeText mode = iota
	modeExpr
)

type Lexer struct {
	scanner   io.ByteScanner
	enterExpr Token
	mode      mode
}

func NewLexer() *Lexer {
	return &Lexer{}
}

func (l *Lexer) Load(scanner io.ByteScanner) {
	l.scanner = scanner
	l.enterExpr.Kind = KindUnknown
	l.mode = modeText
}

func (l *Lexer) Consume() (Token, error) {
	next, err := l.peek()
	if err != nil {
		return l.newError("initial peek: %w", err)
	}

	if isEOF(next) {
		return l.newToken(KindEOF, "")
	}

	if l.mode == modeText {
		tok, err := l.consumeText()
		if tok.Kind == KindEnterExpr {
			l.mode = modeExpr
		}

		return tok, err
	}

	tok, err := l.consumeExpr()
	if tok.Kind == KindLeaveExpr {
		l.mode = modeText
	}

	return tok, err
}

func (l *Lexer) consumeText() (Token, error) {
	next, err := l.peek()
	if err != nil {
		return l.newError("initial peek: %w", err)
	}

	switch {
	case isEnterExpr(next):
		lexeme := string(next)

		return l.newTokenSkip(KindEnterExpr, lexeme, len(lexeme))

	default:
		lexeme, err := l.readStringUntil('{')
		if err != nil {
			return l.newError("read while not expr start: %w", err)
		}

		return l.newToken(KindText, lexeme)
	}
}

func (l *Lexer) consumeExpr() (Token, error) {
	if _, err := l.readWhile(isWhitespace); err != nil {
		return l.newError("skip expr whitespace: %w", err)
	}

	next, err := l.peek()
	if err != nil {
		return l.newError("initial peek: %w", err)
	}

	switch {
	case isLeaveExpr(next):
		lexeme := string(next)

		return l.newTokenSkip(KindLeaveExpr, lexeme, len(lexeme))

	case isStringStart(next):
		delim, err := l.read()
		if err != nil {
			return l.newError("read string delimiter: %w", err)
		}

		lexeme, err := l.readStringUntil(delim)
		if err != nil {
			return l.newError("read string: %w", err)
		}

		// Remove escape characters
		lexeme = strings.ReplaceAll(lexeme, `\`+string(delim), string(delim))

		// Skip is 1 for the ending string delimiter
		return l.newTokenSkip(KindString, lexeme, 1)

	case isIdentStart(next):
		lexeme, err := l.readWhile(isIdent)
		if err != nil {
			return l.newError("read while ident: %w", err)
		}

		if kind, ok := operators[lexeme]; ok {
			return l.newToken(kind, lexeme)
		}

		return l.newToken(KindIdent, lexeme)

	case isDigit(next):
		lexeme, err := l.readWhile(isInteger)
		if err != nil {
			return errorEOF, fmt.Errorf("read while integer: %w", err)
		}

		next, err := l.peek()
		if err != nil {
			return errorEOF, fmt.Errorf("peek digit: %w", err)
		}
		if next == '.' {
			lexeme += string(next)
			if err := l.skip(1); err != nil {
				return errorEOF, fmt.Errorf("discard digit dot: %w", err)
			}

			fraction, err := l.readWhile(isInteger)
			if err != nil {
				return errorEOF, fmt.Errorf("read while fraction integer: %w", err)
			}
			if fraction == "" {
				unexpected, err := l.readWhile(notWhitespace)
				if err != nil {
					return errorEOF, fmt.Errorf("read while fraction unexpected: %w", err)
				}

				lexeme += unexpected

				return l.newToken(KindUnexpected, lexeme)
			}

			lexeme += fraction

			return l.newToken(KindFloat, lexeme)
		}

		return l.newToken(KindInt, lexeme)

	default:
		next1, next2, err := l.peek2()
		if err != nil {
			return l.newError("default expr peek2: %w", err)
		}

		if kind, ok := operators[next2]; ok {
			return l.newTokenSkip(kind, next2, len(next2))
		}
		if kind, ok := operators[next1]; ok {
			return l.newTokenSkip(kind, next1, len(next1))
		}

		lexeme, err := l.readWhile(notWhitespace)
		if err != nil {
			return l.newError("read unexpected expr: %w", err)
		}

		return l.newToken(KindUnexpected, lexeme)
	}
}

func (l *Lexer) newToken(kind TokenKind, lexeme string) (Token, error) {
	tok := Token{
		Kind:   kind,
		Lexeme: lexeme,
	}

	return tok, nil
}

func (l *Lexer) newTokenSkip(kind TokenKind, lexeme string, n int) (Token, error) {
	if err := l.skip(n); err != nil {
		return l.newError("skip: %w", err)
	}

	return l.newToken(kind, lexeme)
}

func (l *Lexer) newError(format string, a ...any) (Token, error) {
	return errorEOF, fmt.Errorf(format, a...)
}

func (l *Lexer) read() (byte, error) {
	b, err := l.scanner.ReadByte()
	if errors.Is(err, io.EOF) {
		return eof, nil
	}
	if err != nil {
		return b, err
	}

	return b, nil
}

func (l *Lexer) unread() error {
	return l.scanner.UnreadByte()
}

func (l *Lexer) peek() (byte, error) {
	b, err := l.read()
	if b == eof {
		return b, nil
	}
	if err != nil {
		return b, fmt.Errorf("read: %w", err)
	}

	return b, l.unread()
}

func (l *Lexer) peek2() (string, string, error) {
	curr, err := l.read()
	if err != nil {
		return "", "", fmt.Errorf("default expr initial read: %w", err)
	}

	next, err := l.peek()
	if err != nil {
		return "", "", fmt.Errorf("default expr next read: %w", err)
	}

	lexeme1 := string(curr)
	lexeme2 := lexeme1 + string(next)

	return lexeme1, lexeme2, l.unread()
}

func (l *Lexer) skip(num int) error {
	for i := 0; i < num; i++ {
		if _, err := l.read(); err != nil {
			return err
		}
	}

	return nil
}

func (l *Lexer) readWhile(valid predicate) (string, error) {
	var sb strings.Builder
	for {
		b, err := l.read()
		switch {
		case err != nil:
			return "", fmt.Errorf("read: %w", err)

		case b == eof:
			return sb.String(), nil

		case !valid(b):
			return sb.String(), l.unread()
		}

		sb.WriteByte(b)
	}
}

func (l *Lexer) readStringUntil(delim byte) (string, error) {
	var sb strings.Builder
	var escape bool
	for {
		b, err := l.read()
		switch {
		case err != nil:
			return "", fmt.Errorf("read: %w", err)

		case b == eof:
			return sb.String(), nil

		case b == delim && !escape:
			return sb.String(), l.unread()

		case b == '\\' && !escape:
			escape = true

			continue
		}

		escape = false

		sb.WriteByte(b)
	}
}

func isEOF(b byte) bool {
	return b == eof
}

func isWhitespace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\r' || b == '\n'
}

func notWhitespace(b byte) bool {
	return !isWhitespace(b)
}

func isEnterExpr(b byte) bool {
	return b == '{'
}

func notEnterExpr(b byte) bool {
	return !isEnterExpr(b)
}

func isLeaveExpr(b byte) bool {
	return b == '}'
}

func notLeaveExpr(b byte) bool {
	return !isLeaveExpr(b)
}

func isAlpha(b byte) bool {
	return b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z'
}

func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

func isInteger(b byte) bool {
	return isDigit(b) || b == '_'
}

func isStringStart(b byte) bool {
	return b == '\'' || b == '"' || b == '`'
}

func isIdentStart(b byte) bool {
	return isAlpha(b) || b == '_'
}

func isIdent(b byte) bool {
	return isIdentStart(b) || isDigit(b)
}
