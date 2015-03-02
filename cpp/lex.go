package cpp

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
)

// TODO:
// Prefer this to not use a goroutine to allow proper garbage collection of lexers.
//
// Prefer tokens had flags indicating whitespace and newlines, instead of special tokens.
// The preprocessor needs this info to correctly identify directives etc.

type Lexer struct {
	brdr      *bufio.Reader
	pos       FilePos
	lastPos   FilePos
	markedPos FilePos
	lastChar  rune
	// At the beginning on line not including whitespace.
	bol bool
	// Set to true if we have hit the end of file.
	eof bool
	// Set to true if we are currently reading a # directive line
	inDirective bool
	stream      chan *Token

	err error
}

type breakout struct{}

// Lex starts a goroutine which lexes the contents of the reader.
// fname is used for error messages when showing the source location.
// No preprocessing is done, this is just pure reading of the unprocessed
// source file.
// The goroutine will not stop until all tokens are read
func Lex(fname string, r io.Reader) *Lexer {
	lx := new(Lexer)
	lx.pos.File = fname
	lx.pos.Line = 1
	lx.pos.Col = 1
	lx.markedPos = lx.pos
	lx.lastPos = lx.pos
	lx.stream = make(chan *Token, 4096)
	lx.brdr = bufio.NewReader(r)
	lx.bol = true
	go lx.lex()
	return lx
}

func (lx *Lexer) Next() (*Token, error) {
	tok := <-lx.stream
	if tok == nil {
		return &Token{Kind: EOF, Pos: lx.pos}, nil
	}
	if tok.Kind == ERROR {
		return tok, lx.err
	}
	return tok, nil
}

func (lx *Lexer) markPos() {
	lx.markedPos = lx.pos
}

func (lx *Lexer) sendTok(kind TokenKind, val string) {
	var tok Token
	tok.Kind = kind
	tok.Val = val
	tok.Pos = lx.markedPos
	tok.hs = emptyHS
	switch kind {
	case END_DIRECTIVE:
		//Do nothing as this is a pseudo directive.
	default:
		lx.bol = false
	}
	lx.stream <- &tok
}

func (lx *Lexer) unreadRune() {
	lx.pos = lx.lastPos
	if lx.lastChar == '\n' {
		lx.bol = false
	}
	if lx.eof {
		return
	}
	lx.brdr.UnreadRune()
}

func (lx *Lexer) readRune() (rune, bool) {
	r, _, err := lx.brdr.ReadRune()
	lx.lastPos = lx.pos
	if err != nil {
		if err == io.EOF {
			lx.eof = true
			lx.lastChar = 0
			return 0, true
		}
		lx.Error(err.Error())
	}
	switch r {
	case '\n':
		lx.pos.Line += 1
		lx.pos.Col = 1
		lx.bol = true
	case '\t':
		lx.pos.Col += 4
	default:
		lx.pos.Col += 1
	}
	lx.lastChar = r
	return r, false
}

func (lx *Lexer) Error(e string) {
	eWithPos := ErrWithLoc(errors.New(e), lx.pos)
	lx.err = eWithPos
	lx.sendTok(ERROR, eWithPos.Error())
	close(lx.stream)
	//recover exits the lexer cleanly
	panic(&breakout{})
}

func (lx *Lexer) lex() {

	defer func() {
		if e := recover(); e != nil {
			_ = e.(*breakout) // Will re-panic if not a breakout.
		}

	}()
	for {
		lx.markPos()
		first, eof := lx.readRune()
		if eof {
			if lx.inDirective {
				lx.sendTok(END_DIRECTIVE, "")
			}
			lx.sendTok(EOF, "")
			break
		}
		switch {
		case isAlpha(first) || first == '_':
			lx.unreadRune()
			lx.readIdentOrKeyword()
		case isNumeric(first):
			lx.unreadRune()
			lx.readConstantIntOrFloat(false)
		case isWhiteSpace(first):
			lx.unreadRune()
			lx.skipWhiteSpace()
		default:
			switch first {
			case '#':
				if lx.isAtLineStart() {
					lx.readDirective()
				} else {
					lx.sendTok(HASH, "#")
				}
			case '!':
				second, _ := lx.readRune()
				switch second {
				case '=':
					lx.sendTok(NEQ, "!=")
				default:
					lx.unreadRune()
					lx.sendTok(NOT, "!")
				}
			case '?':
				lx.sendTok(QUESTION, "?")
			case ':':
				lx.sendTok(COLON, ":")
			case '\'':
				lx.unreadRune()
				lx.readCChar()
			case '"':
				lx.unreadRune()
				lx.readCString()
			case '(':
				lx.sendTok(LPAREN, "(")
			case ')':
				lx.sendTok(RPAREN, ")")
			case '{':
				lx.sendTok(LBRACE, "{")
			case '}':
				lx.sendTok(RBRACE, "}")
			case '[':
				lx.sendTok(LBRACK, "[")
			case ']':
				lx.sendTok(RBRACK, "]")
			case '<':
				second, _ := lx.readRune()
				switch second {
				case '<':
					lx.sendTok(SHL, "<<")
				case '=':
					lx.sendTok(LEQ, "<=")
				default:
					lx.unreadRune()
					lx.sendTok(LSS, "<")
				}
			case '>':
				second, _ := lx.readRune()
				switch second {
				case '>':
					lx.sendTok(SHR, ">>")
				case '=':
					lx.sendTok(GEQ, ">=")
				default:
					lx.unreadRune()
					lx.sendTok(GTR, ">")
				}
			case '+':
				second, _ := lx.readRune()
				switch second {
				case '+':
					lx.sendTok(INC, "++")
				case '=':
					lx.sendTok(ADD_ASSIGN, "+=")
				default:
					lx.unreadRune()
					lx.sendTok(ADD, "+")
				}
			case '.':
				second, _ := lx.readRune()
				lx.unreadRune()
				if isNumeric(second) {
					lx.readConstantIntOrFloat(true)
				} else {
					lx.sendTok(PERIOD, ".")
				}
			case '~':
				lx.sendTok(BNOT, "~")
			case '^':
				second, _ := lx.readRune()
				switch second {
				case '=':
					lx.sendTok(XOR_ASSIGN, "-=")
				default:
					lx.unreadRune()
					lx.sendTok(XOR, "^")
				}
			case '-':
				second, _ := lx.readRune()
				switch second {
				case '>':
					lx.sendTok(ARROW, "->")
				case '-':
					lx.sendTok(DEC, "--")
				case '=':
					lx.sendTok(SUB_ASSIGN, "-=")
				default:
					lx.unreadRune()
					lx.sendTok(SUB, "-")
				}
			case ',':
				lx.sendTok(COMMA, ",")
			case '*':
				second, _ := lx.readRune()
				switch second {
				case '=':
					lx.sendTok(MUL_ASSIGN, "*=")
				default:
					lx.unreadRune()
					lx.sendTok(MUL, "*")
				}
			case '\\':
				r, _ := lx.readRune()
				if r == '\n' {
					break
				}
				lx.Error("misplaced '\\'.")
			case '/':
				second, _ := lx.readRune()
				switch second {
				case '*':
					for {
						c, eof := lx.readRune()
						if eof {
							lx.Error("unclosed comment.")
						}
						if c == '*' {
							closeBar, eof := lx.readRune()
							if eof {
								lx.Error("unclosed comment.")
							}
							if closeBar == '/' {
								break
							}
							//Unread so that we dont lose newlines.
							lx.unreadRune()
						}
					}
				case '/':
					for {
						c, eof := lx.readRune()
						if c == '\n' || eof {
							break
						}
					}
				case '=':
					lx.sendTok(QUO_ASSIGN, "/")
				default:
					lx.unreadRune()
					lx.sendTok(QUO, "/")
				}
			case '%':
				second, _ := lx.readRune()
				switch second {
				case '=':
					lx.sendTok(REM_ASSIGN, "%=")
				default:
					lx.unreadRune()
					lx.sendTok(REM, "%")
				}
			case '|':
				second, _ := lx.readRune()
				switch second {
				case '|':
					lx.sendTok(LOR, "||")
				case '=':
					lx.sendTok(OR_ASSIGN, "|=")
				default:
					lx.unreadRune()
					lx.sendTok(OR, "|")
				}
			case '&':
				second, _ := lx.readRune()
				switch second {
				case '&':
					lx.sendTok(LAND, "&&")
				case '=':
					lx.sendTok(AND_ASSIGN, "&=")
				default:
					lx.unreadRune()
					lx.sendTok(AND, "&")
				}
			case '=':
				second, _ := lx.readRune()
				switch second {
				case '=':
					lx.sendTok(EQL, "==")
				default:
					lx.unreadRune()
					lx.sendTok(ASSIGN, "=")
				}
			case ';':
				lx.sendTok(SEMICOLON, ";")
			default:
				lx.Error(fmt.Sprintf("Internal Error - bad char code '%d'", first))
			}
		}
	}
	close(lx.stream)
}

func (lx *Lexer) readDirective() {
	directiveLine := lx.pos.Line
	lx.skipWhiteSpace()
	if lx.pos.Line != directiveLine {
		return
	}
	var buff bytes.Buffer
	directiveChar, eof := lx.readRune()
	if eof {
		lx.Error("end of file in directive.")
	}
	if isAlpha(directiveChar) {
		lx.inDirective = true
		for isAlpha(directiveChar) {
			buff.WriteRune(directiveChar)
			directiveChar, eof = lx.readRune()
		}
		if !eof {
			lx.unreadRune()
		}
		directive := buff.String()
		lx.sendTok(DIRECTIVE, directive)
		switch directive {
		case "include":
			lx.readHeaderInclude()
		case "define":
			lx.readDefine()
		default:
		}
	} else {
		//wasnt a directive, error will be caught by
		//cpp or parser.
		lx.unreadRune()
	}

}

func (lx *Lexer) readDefine() {
	line := lx.pos.Line
	lx.skipWhiteSpace()
	if lx.pos.Line != line {
		lx.Error("No identifier after define")
	}
	lx.readIdentOrKeyword()
	r, eof := lx.readRune()
	if eof {
		lx.Error("End of File in #efine")
	}
	//Distinguish between a funclike macro
	//and a regular macro.
	if r == '(' {
		lx.sendTok(FUNCLIKE_DEFINE, "")
		lx.unreadRune()
	}

}

func (lx *Lexer) readHeaderInclude() {
	var buff bytes.Buffer
	line := lx.pos.Line
	lx.skipWhiteSpace()
	if lx.pos.Line != line {
		lx.Error("No header after include.")
	}
	lx.markPos()
	opening, _ := lx.readRune()
	var terminator rune
	if opening == '"' {
		terminator = '"'
	} else if opening == '<' {
		terminator = '>'
	} else {
		lx.Error("bad start to header include.")
	}
	buff.WriteRune(opening)
	for {
		c, eof := lx.readRune()
		if eof {
			lx.Error("EOF encountered in header include.")
		}
		if c == '\n' {
			lx.Error("new line in header include.")
		}
		buff.WriteRune(c)
		if c == terminator {
			break
		}
	}
	lx.sendTok(HEADER, buff.String())
}

func (lx *Lexer) readIdentOrKeyword() {
	var buff bytes.Buffer
	lx.markPos()
	first, _ := lx.readRune()
	if !isValidIdentStart(first) {
		panic("internal error")
	}
	buff.WriteRune(first)
	for {
		b, _ := lx.readRune()
		if isValidIdentTail(b) {
			buff.WriteRune(b)
		} else {
			lx.unreadRune()
			str := buff.String()
			tokType, ok := keywordLUT[str]
			if !ok {
				tokType = IDENT
			}
			lx.sendTok(tokType, str)
			break
		}
	}
}

func (lx *Lexer) skipWhiteSpace() {
	for {
		r, _ := lx.readRune()
		if !isWhiteSpace(r) {
			lx.unreadRune()
			break
		}
		if r == '\n' {
			if lx.inDirective {
				lx.sendTok(END_DIRECTIVE, "")
				lx.inDirective = false
			}
		}
	}
}

// Due to the 1 character lookahead we need this bool
func (lx *Lexer) readConstantIntOrFloat(startedWithPeriod bool) {
	var buff bytes.Buffer
	const (
		START = iota
		SECOND
		HEX
		DEC
		FLOAT_START
		FLOAT_AFTER_E
		FLOAT_AFTER_E_SIGN
		INT_TAIL
		FLOAT_TAIL
		END
	)
	var tokType TokenKind
	var state int
	if startedWithPeriod {
		state = FLOAT_START
		tokType = FLOAT_CONSTANT
		buff.WriteRune('.')
	} else {
		state = START
		tokType = INT_CONSTANT
	}
	for state != END {
		r, eof := lx.readRune()
		if eof {
			state = END
			break
		}
		switch state {
		case START:
			if r == '.' {
				state = FLOAT
				buff.WriteRune(r)
				break
			}
			if !isNumeric(r) {
				lx.Error("internal error")
			}
			buff.WriteRune(r)
			state = SECOND
		case SECOND:
			if r == 'x' || r == 'X' {
				state = HEX
				buff.WriteRune(r)
			} else if isNumeric(r) {
				state = DEC
				buff.WriteRune(r)
			} else if r == 'e' || r == 'E' {
				state = FLOAT_AFTER_E
				tokType = FLOAT_CONSTANT
				buff.WriteRune(r)
			} else if r == '.' {
				state = FLOAT_START
				tokType = FLOAT_CONSTANT
				buff.WriteRune(r)
			} else {
				state = END
			}
		case DEC:
			if !isNumeric(r) {
				switch r {
				case 'l', 'L', 'u', 'U':
					state = INT_TAIL
					buff.WriteRune(r)
				case 'e', 'E':
					state = FLOAT_AFTER_E
					tokType = FLOAT_CONSTANT
					buff.WriteRune(r)
				case '.':
					state = FLOAT_START
					tokType = FLOAT_CONSTANT
					buff.WriteRune(r)
				default:
					if isValidIdentStart(r) {
						lx.Error("invalid constant int")
					}
					state = END
				}
			} else {
				buff.WriteRune(r)
			}
		case HEX:
			if !isHexDigit(r) {
				switch r {
				case 'l', 'L', 'u', 'U':
					state = INT_TAIL
					buff.WriteRune(r)
				default:
					if isValidIdentStart(r) {
						lx.Error("invalid constant int")
					}
					state = END
				}
			} else {
				buff.WriteRune(r)
			}
		case INT_TAIL:
			switch r {
			case 'l', 'L', 'u', 'U':
				buff.WriteRune(r)
			default:
				state = END
			}
		case FLOAT_START:
			if !isNumeric(r) {
				switch r {
				case 'e', 'E':
					state = FLOAT_AFTER_E
					buff.WriteRune(r)
				default:
					if isValidIdentStart(r) {
						lx.Error("invalid floating point constant.")
					}
					state = END
				}
			} else {
				buff.WriteRune(r)
			}
		case FLOAT_AFTER_E:
			if r == '-' || r == '+' {
				state = FLOAT_AFTER_E_SIGN
				buff.WriteRune(r)
			} else if isNumeric(r) {
				state = FLOAT_AFTER_E_SIGN
				buff.WriteRune(r)
			} else {
				lx.Error("invalid float constant - expected number or signed after e")
			}
		case FLOAT_AFTER_E_SIGN:
			if isNumeric(r) {
				buff.WriteRune(r)
			} else {
				switch r {
				case 'l', 'L', 'f', 'F':
					buff.WriteRune(r)
					state = FLOAT_TAIL
				default:
					if isValidIdentStart(r) {
						lx.Error("invalid float constant")
					} else {
						state = END
					}
				}
			}
		case FLOAT_TAIL:
			switch r {
			case 'l', 'L', 'f', 'F':
				buff.WriteRune(r)
			default:
				if isValidIdentStart(r) {
					lx.Error("invalid float constant")
				}
				state = END
			}
		default:
			lx.Error("internal error.")
		}
	}
	lx.unreadRune()
	lx.sendTok(tokType, buff.String())
}

func (lx *Lexer) readCString() {
	const (
		START = iota
		MID
		ESCAPED
		END
	)
	var buff bytes.Buffer
	var state int
	lx.markPos()
	for state != END {
		r, eof := lx.readRune()
		if eof {
			lx.Error("eof in string literal")
		}
		switch state {
		case START:
			if r != '"' {
				lx.Error("internal error")
			}
			buff.WriteRune(r)
			state = MID
		case MID:
			switch r {
			case '\\':
				state = ESCAPED
			case '"':
				buff.WriteRune(r)
				state = END
			default:
				buff.WriteRune(r)
			}
		case ESCAPED:
			switch r {
			case '\r':
				// empty
			case '\n':
				state = MID
			default:
				buff.WriteRune('\\')
				buff.WriteRune(r)
				state = MID
			}
		}
	}
	lx.sendTok(STRING, buff.String())
}

func (lx *Lexer) readCChar() {
	const (
		START = iota
		MID
		ESCAPED
		END
	)
	var buff bytes.Buffer
	var state int
	lx.markPos()
	for state != END {
		r, eof := lx.readRune()
		if eof {
			lx.Error("eof in char literal")
		}
		switch state {
		case START:
			if r != '\'' {
				lx.Error("internal error")
			}
			buff.WriteRune(r)
			state = MID
		case MID:
			switch r {
			case '\\':
				state = ESCAPED
			case '\'':
				buff.WriteRune(r)
				state = END
			default:
				buff.WriteRune(r)
			}
		case ESCAPED:
			switch r {
			case '\r':
				// empty
			case '\n':
				state = MID
			default:
				buff.WriteRune('\\')
				buff.WriteRune(r)
				state = MID
			}
		}
	}
	lx.sendTok(CHAR_CONSTANT, buff.String())

}

func (lx *Lexer) isAtLineStart() bool {
	return lx.bol
}

func isValidIdentTail(b rune) bool {
	return isValidIdentStart(b) || isNumeric(b) || b == '$'
}

func isValidIdentStart(b rune) bool {
	return b == '_' || isAlpha(b)
}

func isAlpha(b rune) bool {
	if b >= 'a' && b <= 'z' {
		return true
	}
	if b >= 'A' && b <= 'Z' {
		return true
	}
	return false
}

func isWhiteSpace(b rune) bool {
	return b == ' ' || b == '\r' || b == '\n' || b == '\t' || b == '\f'
}

func isNumeric(b rune) bool {
	if b >= '0' && b <= '9' {
		return true
	}
	return false
}

func isHexDigit(b rune) bool {
	return isNumeric(b) || (b >= 'a' && b <= 'f') || (b >= 'A' && b <= 'F')
}
