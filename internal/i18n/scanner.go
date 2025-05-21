package i18n

import (
	"errors"
	"fmt"
	"io"
	"strings"
)

type predicate func(byte) bool

const eof = 0xFF

var errorEOF = Token{Kind: KindEOF, Lexeme: "scanner error"}

type mode byte

const (
	modeText mode = iota
	modeExpr
)

type Scanner struct {
	scanner   io.ByteScanner
	enterExpr Token
	mode      mode
}

func NewScanner() *Scanner {
	return &Scanner{}
}

func (s *Scanner) Load(scanner io.ByteScanner) {
	s.scanner = scanner
	s.enterExpr.Kind = KindUnknown
	s.mode = modeText
}

func (s *Scanner) Consume() (Token, error) {
	next, err := s.peek()
	if err != nil {
		return s.newError("initial peek: %w", err)
	}

	if isEOF(next) {
		return s.newToken(KindEOF, "")
	}

	if s.mode == modeText {
		tok, err := s.consumeText()
		if tok.Kind == KindEnterExpr {
			s.mode = modeExpr
		}

		return tok, err
	}

	tok, err := s.consumeExpr()
	if tok.Kind == KindLeaveExpr {
		s.mode = modeText
	}

	return tok, err
}

func (s *Scanner) consumeText() (Token, error) {
	next, err := s.peek()
	if err != nil {
		return s.newError("initial peek: %w", err)
	}

	switch {
	case isEnterExpr(next):
		lexeme := string(next)

		return s.newTokenSkip(KindEnterExpr, lexeme, len(lexeme))

	default:
		lexeme, err := s.readStringUntil('{')
		if err != nil {
			return s.newError("read while not expr start: %w", err)
		}

		return s.newToken(KindText, lexeme)
	}
}

func (s *Scanner) consumeExpr() (Token, error) {
	if _, err := s.readWhile(isWhitespace); err != nil {
		return s.newError("skip expr whitespace: %w", err)
	}

	next, err := s.peek()
	if err != nil {
		return s.newError("initial peek: %w", err)
	}

	switch {
	case isLeaveExpr(next):
		lexeme := string(next)

		return s.newTokenSkip(KindLeaveExpr, lexeme, len(lexeme))

	case isStringStart(next):
		delim, err := s.read()
		if err != nil {
			return s.newError("read string delimiter: %w", err)
		}

		lexeme, err := s.readStringUntil(delim)
		if err != nil {
			return s.newError("read string: %w", err)
		}

		// Remove escape characters
		lexeme = strings.ReplaceAll(lexeme, `\`+string(delim), string(delim))

		// Skip is 1 for the ending string delimiter
		return s.newTokenSkip(KindString, lexeme, 1)

	case isIdentStart(next):
		lexeme, err := s.readWhile(isIdent)
		if err != nil {
			return s.newError("read while ident: %w", err)
		}

		if kind, ok := operators[lexeme]; ok {
			return s.newToken(kind, lexeme)
		}

		return s.newToken(KindIdent, lexeme)

	case isDigit(next):
		lexeme, err := s.readWhile(isInteger)
		if err != nil {
			return errorEOF, fmt.Errorf("read while integer: %w", err)
		}

		next, err := s.peek()
		if err != nil {
			return errorEOF, fmt.Errorf("peek digit: %w", err)
		}
		if next == '.' {
			lexeme += string(next)
			if err := s.skip(1); err != nil {
				return errorEOF, fmt.Errorf("discard digit dot: %w", err)
			}

			fraction, err := s.readWhile(isInteger)
			if err != nil {
				return errorEOF, fmt.Errorf("read while fraction integer: %w", err)
			}
			if fraction == "" {
				unexpected, err := s.readWhile(notWhitespace)
				if err != nil {
					return errorEOF, fmt.Errorf("read while fraction unexpected: %w", err)
				}

				lexeme += unexpected

				return s.newToken(KindUnexpected, lexeme)
			}

			lexeme += fraction

			return s.newToken(KindFloat, lexeme)
		}

		return s.newToken(KindInt, lexeme)

	default:
		next1, next2, err := s.peek2()
		if err != nil {
			return s.newError("default expr peek2: %w", err)
		}

		if kind, ok := operators[next2]; ok {
			return s.newTokenSkip(kind, next2, len(next2))
		}
		if kind, ok := operators[next1]; ok {
			return s.newTokenSkip(kind, next1, len(next1))
		}

		lexeme, err := s.readWhile(notWhitespace)
		if err != nil {
			return s.newError("read unexpected expr: %w", err)
		}

		return s.newToken(KindUnexpected, lexeme)
	}
}

func (s *Scanner) newToken(kind TokenKind, lexeme string) (Token, error) {
	tok := Token{
		Kind:   kind,
		Lexeme: lexeme,
	}

	return tok, nil
}

func (s *Scanner) newTokenSkip(kind TokenKind, lexeme string, n int) (Token, error) {
	if err := s.skip(n); err != nil {
		return s.newError("skip: %w", err)
	}

	return s.newToken(kind, lexeme)
}

func (s *Scanner) newError(format string, a ...any) (Token, error) {
	return errorEOF, fmt.Errorf(format, a...)
}

func (s *Scanner) read() (byte, error) {
	b, err := s.scanner.ReadByte()
	if errors.Is(err, io.EOF) {
		return eof, nil
	}
	if err != nil {
		return b, err
	}

	return b, nil
}

func (s *Scanner) unread() error {
	return s.scanner.UnreadByte()
}

func (s *Scanner) peek() (byte, error) {
	b, err := s.read()
	if b == eof {
		return b, nil
	}
	if err != nil {
		return b, fmt.Errorf("read: %w", err)
	}

	return b, s.unread()
}

func (s *Scanner) peek2() (string, string, error) {
	curr, err := s.read()
	if err != nil {
		return "", "", fmt.Errorf("default expr initial read: %w", err)
	}

	next, err := s.peek()
	if err != nil {
		return "", "", fmt.Errorf("default expr next read: %w", err)
	}

	lexeme1 := string(curr)
	lexeme2 := lexeme1 + string(next)

	return lexeme1, lexeme2, s.unread()
}

func (s *Scanner) skip(num int) error {
	for i := 0; i < num; i++ {
		if _, err := s.read(); err != nil {
			return err
		}
	}

	return nil
}

func (s *Scanner) readWhile(valid predicate) (string, error) {
	var sb strings.Builder
	for {
		b, err := s.read()
		switch {
		case err != nil:
			return "", fmt.Errorf("read: %w", err)

		case b == eof:
			return sb.String(), nil

		case !valid(b):
			return sb.String(), s.unread()
		}

		sb.WriteByte(b)
	}
}

func (s *Scanner) readStringUntil(delim byte) (string, error) {
	var sb strings.Builder
	var escape bool
	for {
		b, err := s.read()
		switch {
		case err != nil:
			return "", fmt.Errorf("read: %w", err)

		case b == eof:
			return sb.String(), nil

		case b == delim && !escape:
			return sb.String(), s.unread()

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
