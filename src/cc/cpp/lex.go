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
	ls.stream = make(chan *Token)
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
	ls.bol = false
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
			break
		}
		switch {
		case isAlpha(first) || first == '_':
			ls.unreadRune()
			ls.readIdentOrKeyword()
		case isNumeric(first):
			ls.unreadRune()
			ls.readConstantInt()
		case isWhiteSpace(first):
			ls.unreadRune()
			ls.skipWhiteSpace()
		default:
			switch first {
			case '#':
				if ls.isAtLineStart() {
					ls.sendTok(HASH, "#")
					ls.readDirective()
				} else {
					ls.sendTok(HASH, "#")
				}
			case '!':
				ls.sendTok(NOT, "!")
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
				ls.sendTok(LSS, "<")
			case '>':
				ls.sendTok(GTR, ">")
			case '+':
				second, _ := ls.readRune()
				if second == '+' {
					ls.sendTok(INC, "++")
					break
				}
				ls.unreadRune()
				ls.sendTok(ADD, "+")
			case '.':
				ls.sendTok(PERIOD, ".")
			case '~':
				ls.sendTok(BNOT, "~")
			case '^':
				ls.sendTok(XOR, "^")
			case '-':
				second, _ := ls.readRune()
				if second == '>' {
					ls.sendTok(ARROW, "->")
					break
				}
				ls.unreadRune()
				ls.sendTok(SUB, "-")
			case ',':
				ls.sendTok(COMMA, ",")
			case '*':
				ls.sendTok(MUL, "*")
			case '\\':
				r, _ := ls.readRune()
				if r == '\n' {
					break
				}
				ls.lexError("misplaced '\\'.")
			case '/':
				second, _ := ls.readRune()
				if second == '*' { // C comment.
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
				} else if second == '/' { // C++ comment.
					for {
						c, _ := ls.readRune()
						if c == '\n' {
							break
						}
					}
				} else {
					ls.unreadRune()
					ls.sendTok(QUO, "/")
				}
			case '%':
				ls.sendTok(REM, "%")
			case '|':
				second, _ := ls.readRune()
				if second == '|' {
					ls.sendTok(LOR, "||")
					break
				}
				ls.unreadRune()
				ls.sendTok(OR, "|")
			case '&':
				second, _ := ls.readRune()
				if second == '&' {
					ls.sendTok(LAND, "&&")
					break
				}
				ls.unreadRune()
				ls.sendTok(AND, "&")
			case '=':
				second, _ := ls.readRune()
				if second == '=' {
					ls.sendTok(EQL, "==")
					break
				}
				ls.unreadRune()
				ls.sendTok(ASSIGN, "=")
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
		ls.lexError("Empty directive")
	}
	var buff bytes.Buffer
	ls.markPos()
	directiveChar, eof := ls.readRune()
	if eof {
		ls.lexError("end of file in directive.")
	}
	if isAlpha(directiveChar) {
		for isAlpha(directiveChar) {
			buff.WriteRune(directiveChar)
			directiveChar, eof = ls.readRune()
		}
		if !eof {
			ls.unreadRune()
		}
		directive := buff.String()
		ls.sendTok(DIRECTIVE, directive)
		if directive == "include" {
			ls.readHeaderInclude()
		}
	} else {
		//wasnt a directive, error will be caught by
		//cpp or parser.
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
	}
}

func (ls *lexerState) readConstantInt() {
	var buff bytes.Buffer
	ls.markPos()
	const (
		START = iota
		SECOND
		HEX
		DEC
		FLOAT
		TAIL
		END
	)
	var tokType TokenKind = INT_CONSTANT
	state := START
	for state != END {
		r, eof := ls.readRune()
		if eof {
			state = END
			break
		}
		switch state {
		case START:
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
			} else {
				state = END
			}
		case DEC:
			if !isNumeric(r) {
				switch r {
				case 'l', 'L', 'u', 'U':
					state = TAIL
					buff.WriteRune(r)
				case '.':
					state = FLOAT
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
		case FLOAT:
			if !isNumeric(r) {
				switch r {
				default:
					if isValidIdentStart(r) {
						ls.lexError("invalid floating point constant.")
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
					state = TAIL
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
		case TAIL:
			switch r {
			case 'l', 'L', 'u', 'U':
				buff.WriteRune(r)
			default:
				state = END
			}
		default:
			ls.lexError("internal error.")
		}
	}
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
			case 'n', 'r', 't':
				buff.WriteRune('\\')
				buff.WriteRune(r)
				state = MID
			case '\r':
				// empty
			case '\n':
				state = MID
			default:
				ls.lexError(fmt.Sprintf("unknown escape char %c", r))
			}
		}
	}
	ls.sendTok(STRING, buff.String())
}

func (ls *lexerState) readCChar() {
	/*
		var buff bytes.Buffer
		ls.markPos()
		first, _ := ls.readRune()
		if first != '\'' {
			ls.lexError("internal error")
		}
		buff.WriteRune('\'')
		escaped := false
		for {
			b, eof := ls.readRune()
			if eof {
				ls.lexError("unterminated char literal.")
			}
			if b == '\'' && !escaped {
				break
			}
			if b == '\n' {
				ls.lexError("unterminated char literal")
			}
			if !escaped {
				if b == '\\' {
					escaped = true
				}
			} else {
				escaped = false
			}
			buff.WriteRune(b)
		}
		buff.WriteRune('\'')
		ls.sendTok(CHAR_CONSTANT, buff.String())
	*/
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

func isAlphaNumeric(b rune) bool {
	return isNumeric(b) || isAlpha(b)
}
