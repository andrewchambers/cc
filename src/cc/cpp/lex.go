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
		b, eof := ls.readRune()
		if isValidIdentTail(b) {
			buff.WriteRune(b)
		} else {
			if !eof {
				ls.unreadRune()
			}
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
		r, eof := ls.readRune()
		if !isWhiteSpace(r) {
			if !eof {
				ls.unreadRune()
			}
			break
		}
	}
}

func (ls *lexerState) readConstantInt() {
	var buff bytes.Buffer
	ls.markPos()
	for {
		r, eof := ls.readRune()
		if isNumeric(r) {
			buff.WriteRune(r)
		} else {
			if !eof {
				ls.unreadRune()
			}
			str := buff.String()
			ls.sendTok(INT_CONSTANT, str)
			break
		}
	}
}

func (ls *lexerState) readCString() {
	var buff bytes.Buffer
	ls.markPos()
	first, _ := ls.readRune()
	if first != '"' {
		ls.lexError("internal error")
	}
	buff.WriteRune('"')
	escaped := false
	for {
		b, eof := ls.readRune()
		if eof {
			ls.lexError("Unterminated string literal.")
		}
		if b == '"' && !escaped {
			break
		}
		if b == '\n' {
			ls.lexError("unterminated string")
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
	buff.WriteRune('"')
	ls.sendTok(STRING, buff.String())
}

func (ls *lexerState) readCChar() {
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

func isAlphaNumeric(b rune) bool {
	return isNumeric(b) || isAlpha(b)
}
