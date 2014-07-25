package cpp

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
)

type lexerState struct {
	pos       FilePos
	markedPos FilePos
	brdr      *bufio.Reader
	lastChar  rune
	oldCol    int
	//At the beginning on line not including whitespace
	bol bool
	// Set to true if we have hit the end of file
	eof bool
	//Set to true if we are currently reading a # directive line
	inDirective bool
	//If this channel recieves a value, the lexing goroutines should close
	//its output channel and its error channel and terminate.
	cancel chan struct{}
	stream chan *Token
}

type breakout struct {
}

//Lex starts a goroutine which lexes the contents of the reader.
//fname is used for error messages when showing the source location.
//No preprocessing is done, this is just pure reading of the unprocessed
//source file.
//The goroutine will not stop until all tokens are read
func Lex(fname string, r io.Reader) chan *Token {
	ls := new(lexerState)
	ls.pos.File = fname
	ls.pos.Line = 1
	ls.pos.Col = 1
	ls.stream = make(chan *Token, 1024)
	ls.brdr = bufio.NewReader(r)
	ls.bol = true
	go ls.lex()
	return ls.stream
}

func (ls *lexerState) markPos() {
	ls.markedPos = ls.pos
}

func (ls *lexerState) sendTok(kind TokenKind, val string) {
	var tok Token
	tok.Kind = kind
	tok.Val = val
	tok.Pos = ls.markedPos
	//XXX This might slow things down.
	//Not all tokens need a hideset.
	tok.hs = newHideSet()
	switch kind {
	case END_DIRECTIVE:
		//Do nothing as this is a pseudo directive.
	default:
		ls.bol = false
	}
	ls.stream <- &tok
}

func (ls *lexerState) unreadRune() {
	if ls.eof {
		return
	}
	switch ls.lastChar {
	case '\n':
		ls.pos.Line -= 1
		ls.pos.Col = ls.oldCol
		ls.bol = false
	case '\t':
		ls.pos.Col -= 4 // Is this ok?
	default:
		ls.pos.Col -= 1
	}
	ls.brdr.UnreadRune()
}

func (ls *lexerState) readRune() (rune, bool) {
	r, _, err := ls.brdr.ReadRune()
	if err != nil {
		if err == io.EOF {
			ls.eof = true
			ls.lastChar = 0
			return 0, true
		}
		ls.lexError(err.Error())
	}
	switch r {
	case '\n':
		ls.pos.Line += 1
		ls.oldCol = ls.pos.Col
		ls.pos.Col = 1
		ls.bol = true
	case '\t':
		ls.pos.Col += 4 // Is this ok?
	default:
		ls.pos.Col += 1
	}
	ls.lastChar = r
	return r, false
}

func (ls *lexerState) lexError(e string) {
	eWithPos := fmt.Sprintf("Error while reading %s. %s", ls.pos, e)
	ls.sendTok(ERROR, eWithPos)
	close(ls.stream)
	//recover exits the lexer cleanly
	panic(&breakout{})
}

func (ls *lexerState) lex() {

	//This recovery happens if lexError is called.
	defer func() {
		//XXX is this correct way to retrigger non breakout?
		if e := recover(); e != nil {
			_ = e.(*breakout) // Will re-panic if not a parse error.
		}

	}()
	ls.markPos()
	first, eof := ls.readRune()
	for {
		if eof {
			if ls.inDirective {
				ls.sendTok(END_DIRECTIVE, "")
			}
			break
		}
		switch {
		case isAlpha(first) || first == '_':
			ls.unreadRune()
			ls.readIdentOrKeyword()
		case isNumeric(first):
			ls.unreadRune()
			ls.readConstantIntOrFloat(false)
		case isWhiteSpace(first):
			ls.unreadRune()
			ls.skipWhiteSpace()
		default:
			switch first {
			case '#':
				if ls.isAtLineStart() {
					ls.readDirective()
				} else {
					ls.sendTok(HASH, "#")
				}
			case '!':
				second, _ := ls.readRune()
				switch second {
				case '=':
					ls.sendTok(NEQ, "!=")
				default:
					ls.unreadRune()
					ls.sendTok(NOT, "!")
				}
			case '?':
				ls.sendTok(QUESTION, "?")
			case ':':
				ls.sendTok(COLON, ":")
			case '\'':
				ls.unreadRune()
				ls.readCChar()
			case '"':
				ls.unreadRune()
				ls.readCString()
			case '(':
				ls.sendTok(LPAREN, "(")
			case ')':
				ls.sendTok(RPAREN, ")")
			case '{':
				ls.sendTok(LBRACE, "{")
			case '}':
				ls.sendTok(RBRACE, "}")
			case '[':
				ls.sendTok(LBRACK, "[")
			case ']':
				ls.sendTok(RBRACE, "]")
			case '<':
				second, _ := ls.readRune()
				switch second {
				case '<':
					ls.sendTok(SHL, "<<")
				case '=':
					ls.sendTok(LEQ, "<=")
				default:
					ls.unreadRune()
					ls.sendTok(LSS, "<")
				}
			case '>':
				second, _ := ls.readRune()
				switch second {
				case '>':
					ls.sendTok(SHR, ">>")
				case '=':
					ls.sendTok(GEQ, ">=")
				default:
					ls.unreadRune()
					ls.sendTok(GTR, ">")
				}
			case '+':
				second, _ := ls.readRune()
				switch second {
				case '+':
					ls.sendTok(INC, "++")
				case '=':
					ls.sendTok(ADD_ASSIGN, "+=")
				default:
					ls.unreadRune()
					ls.sendTok(ADD, "+")
				}
			case '.':
				second, _ := ls.readRune()
				ls.unreadRune()
				if isNumeric(second) {
					ls.readConstantIntOrFloat(true)
				} else {
					ls.sendTok(PERIOD, ".")
				}
			case '~':
				ls.sendTok(BNOT, "~")
			case '^':
				second, _ := ls.readRune()
				switch second {
				case '=':
					ls.sendTok(XOR_ASSIGN, "-=")
				default:
					ls.unreadRune()
					ls.sendTok(XOR, "^")
				}
			case '-':
				second, _ := ls.readRune()
				switch second {
				case '>':
					ls.sendTok(ARROW, "->")
				case '-':
					ls.sendTok(DEC, "--")
				case '=':
					ls.sendTok(SUB_ASSIGN, "-=")
				default:
					ls.unreadRune()
					ls.sendTok(SUB, "-")
				}
			case ',':
				ls.sendTok(COMMA, ",")
			case '*':
				second, _ := ls.readRune()
				switch second {
				case '=':
					ls.sendTok(MUL_ASSIGN, "*=")
				default:
					ls.unreadRune()
					ls.sendTok(MUL, "*")
				}
			case '\\':
				r, _ := ls.readRune()
				if r == '\n' {
					break
				}
				ls.lexError("misplaced '\\'.")
			case '/':
				second, _ := ls.readRune()
				switch second {
				case '*':
					for {
						c, eof := ls.readRune()
						if eof {
							ls.lexError("unclosed comment.")
						}
						if c == '*' {
							closeBar, eof := ls.readRune()
							if eof {
								ls.lexError("unclosed comment.")
							}
							if closeBar == '/' {
								break
							}
							//Unread so that we dont lose newlines.
							ls.unreadRune()
						}
					}
				case '/':
					for {
						c, _ := ls.readRune()
						if c == '\n' {
							break
						}
					}
				case '=':
					ls.sendTok(QUO_ASSIGN, "/")
				default:
					ls.unreadRune()
					ls.sendTok(QUO, "/")
				}
			case '%':
				second, _ := ls.readRune()
				switch second {
				case '=':
					ls.sendTok(REM_ASSIGN, "%=")
				default:
					ls.unreadRune()
					ls.sendTok(REM, "%")
				}
			case '|':
				second, _ := ls.readRune()
				switch second {
				case '|':
					ls.sendTok(LOR, "||")
				case '=':
					ls.sendTok(OR_ASSIGN, "|=")
				default:
					ls.unreadRune()
					ls.sendTok(OR, "|")
				}
			case '&':
				second, _ := ls.readRune()
				switch second {
				case '&':
					ls.sendTok(LAND, "&&")
				case '=':
					ls.sendTok(AND_ASSIGN, "&=")
				default:
					ls.unreadRune()
					ls.sendTok(AND, "&")
				}
			case '=':
				second, _ := ls.readRune()
				switch second {
				case '=':
					ls.sendTok(EQL, "==")
				default:
					ls.unreadRune()
					ls.sendTok(ASSIGN, "=")
				}
			case ';':
				ls.sendTok(SEMICOLON, ";")
			default:
				ls.lexError(fmt.Sprintf("Internal Error - bad char code '%d'", first))
			}
		}
		ls.markPos()
		first, eof = ls.readRune()
	}
	close(ls.stream)
}

func (ls *lexerState) readDirective() {
	directiveLine := ls.pos.Line
	ls.skipWhiteSpace()
	if ls.pos.Line != directiveLine {
		return
	}
	var buff bytes.Buffer
	directiveChar, eof := ls.readRune()
	if eof {
		ls.lexError("end of file in directive.")
	}
	if isAlpha(directiveChar) {
		ls.inDirective = true
		for isAlpha(directiveChar) {
			buff.WriteRune(directiveChar)
			directiveChar, eof = ls.readRune()
		}
		if !eof {
			ls.unreadRune()
		}
		directive := buff.String()
		ls.sendTok(DIRECTIVE, directive)
		switch directive {
		case "include":
			ls.readHeaderInclude()
		case "define":
			ls.readDefine()
		default:
		}
	} else {
		//wasnt a directive, error will be caught by
		//cpp or parser.
		ls.unreadRune()
	}

}

func (ls *lexerState) readDefine() {
	line := ls.pos.Line
	ls.skipWhiteSpace()
	if ls.pos.Line != line {
		ls.lexError("No identifier after define")
	}
	ls.readIdentOrKeyword()
	r, eof := ls.readRune()
	if eof {
		ls.lexError("End of File in #efine")
	}
	//Distinguish between a funclike macro
	//and a regular macro.
	if r == '(' {
		ls.sendTok(FUNCLIKE_DEFINE, "")
		ls.unreadRune()
	}

}

func (ls *lexerState) readHeaderInclude() {
	var buff bytes.Buffer
	line := ls.pos.Line
	ls.skipWhiteSpace()
	if ls.pos.Line != line {
		ls.lexError("No header after include.")
	}
	ls.markPos()
	opening, _ := ls.readRune()
	var terminator rune
	if opening == '"' {
		terminator = '"'
	} else if opening == '<' {
		terminator = '>'
	} else {
		ls.lexError("bad start to header include.")
	}
	buff.WriteRune(opening)
	for {
		c, eof := ls.readRune()
		if eof {
			ls.lexError("EOF encountered in header include.")
		}
		if c == '\n' {
			ls.lexError("new line in header include.")
		}
		buff.WriteRune(c)
		if c == terminator {
			break
		}
	}
	ls.sendTok(HEADER, buff.String())
}

func (ls *lexerState) readIdentOrKeyword() {
	var buff bytes.Buffer
	ls.markPos()
	first, _ := ls.readRune()
	if !isValidIdentStart(first) {
		panic("internal error")
	}
	buff.WriteRune(first)
	for {
		b, _ := ls.readRune()
		if isValidIdentTail(b) {
			buff.WriteRune(b)
		} else {
			ls.unreadRune()
			str := buff.String()
			tokType, ok := keywordLUT[str]
			if !ok {
				tokType = IDENT
			}
			ls.sendTok(tokType, str)
			break
		}
	}
}

func (ls *lexerState) skipWhiteSpace() {
	for {
		r, _ := ls.readRune()
		if !isWhiteSpace(r) {
			ls.unreadRune()
			break
		}
		if r == '\n' {
			if ls.inDirective {
				ls.sendTok(END_DIRECTIVE, "")
				ls.inDirective = false
			}
		}
	}
}

// Due to the 1 character lookahead we need this bool
func (ls *lexerState) readConstantIntOrFloat(startedWithPeriod bool) {
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
		r, eof := ls.readRune()
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
				ls.lexError("internal error")
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
						ls.lexError("invalid constant int")
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
						ls.lexError("invalid constant int")
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
						ls.lexError("invalid floating point constant.")
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
				ls.lexError("invalid float constant - expected number or signed after e")
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
						ls.lexError("invalid float constant")
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
					ls.lexError("invalid float constant")
				}
				state = END
			}
		default:
			ls.lexError("internal error.")
		}
	}
	ls.unreadRune()
	ls.sendTok(tokType, buff.String())
}

func (ls *lexerState) readCString() {
	const (
		START = iota
		MID
		ESCAPED
		END
	)
	var buff bytes.Buffer
	var state int
	ls.markPos()
	for state != END {
		r, eof := ls.readRune()
		if eof {
			ls.lexError("eof in string literal")
		}
		switch state {
		case START:
			if r != '"' {
				ls.lexError("internal error")
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
	ls.sendTok(STRING, buff.String())
}

func (ls *lexerState) readCChar() {
	const (
		START = iota
		MID
		ESCAPED
		END
	)
	var buff bytes.Buffer
	var state int
	ls.markPos()
	for state != END {
		r, eof := ls.readRune()
		if eof {
			ls.lexError("eof in char literal")
		}
		switch state {
		case START:
			if r != '\'' {
				ls.lexError("internal error")
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
	ls.sendTok(CHAR_CONSTANT, buff.String())

}

func (ls *lexerState) isAtLineStart() bool {
	return ls.bol
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
