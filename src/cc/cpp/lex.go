package cpp

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
)

type lexerState struct {
	pos  FilePos
	brdr *bufio.Reader
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

	first, _, err := ls.brdr.ReadRune()

	for {
		if err == io.EOF {
			break
		}
		if err != nil {
			ls.lexError(err.Error())
		}
		switch {
		case isAlpha(first) || first == '_':
			ls.brdr.UnreadRune()
			ls.readIdentOrKeyword()
		case isNumeric(first):
			ls.brdr.UnreadRune()
			ls.readConstantInt()
		case isWhiteSpace(first):
			ls.brdr.UnreadRune()
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
				ls.sendTok('?', "?")
			case ':':
				ls.sendTok(COLON, ":")
			case '"':
				ls.brdr.UnreadRune()
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
				second, _, _ := ls.brdr.ReadRune()
				if second == '+' {
					ls.sendTok(INC, "++")
					break
				}
				ls.brdr.UnreadRune()
				ls.sendTok(ADD, "+")
			case '.':
				ls.sendTok(PERIOD, ".")
			case '-':
				second, _, _ := ls.brdr.ReadRune()
				if second == '>' {
					ls.sendTok(ARROW, "->")
					break
				}
				ls.brdr.UnreadRune()
				ls.sendTok(SUB, "-")
			case ',':
				ls.sendTok(COMMA, ",")
			case '*':
				ls.sendTok(MUL, "*")
			case '/':
				second, _, _ := ls.brdr.ReadRune()
				if second == '*' { // C comment.
					for {
						c, _, err := ls.brdr.ReadRune()
						if err == io.EOF {
							ls.lexError("unclosed comment.")
						}
						if err != nil {
							ls.lexError(err.Error())
						}
						if c == '*' {
							closeBar, _, _ := ls.brdr.ReadRune()
							if closeBar == '/' {
								break
							}
							//Unread so that we dont lose newlines.
							ls.brdr.UnreadRune()
						}
						if c == '\n' {
							ls.pos.Col = 1
							ls.pos.Line += 1
						}
					}
				} else if second == '/' { // C++ comment.
					for {
						c, _, err := ls.brdr.ReadRune()
						if err != nil {
							break
						}
						if c == '\n' {
							ls.pos.Col = 1
							ls.pos.Line += 1
							break
						}
					}
				} else {
					ls.sendTok(QUO, "/")
					ls.brdr.UnreadRune()
				}
			case '|':
				second, _, _ := ls.brdr.ReadRune()
				if second == '|' {
					ls.sendTok(LOR, "||")
					break
				}
				ls.brdr.UnreadRune()
				ls.sendTok(OR, "|")
			case '&':
				second, _, _ := ls.brdr.ReadRune()
				if second == '&' {
					ls.sendTok(LAND, "&&")
					break
				}
				ls.brdr.UnreadRune()
				ls.sendTok(AND, "&")
			case '=':
				second, _, _ := ls.brdr.ReadRune()
				if second == '=' {
					ls.sendTok(EQL, "==")
					break
				}
				ls.brdr.UnreadRune()
				ls.sendTok(ASSIGN, "=")
			case ';':
				ls.sendTok(SEMICOLON, ";")
			default:
				//ls.lexError(fmt.Errorf("Internal Error - bad char code '%d'", first))
			}
		}
		first, _, err = ls.brdr.ReadRune()
	}
	close(ls.stream)
}

func (ls *lexerState) sendTok(kind TokenKind, val string) {
	var tok Token
	tok.Kind = kind
	tok.Val = val
	tok.Pos.Line = ls.pos.Line
	tok.Pos.Col = ls.pos.Col
	tok.Pos.File = ls.pos.File
	ls.bol = false
	ls.stream <- &tok
	ls.pos.Col += len(val)
}

func (ls *lexerState) readDirective() {
	directiveLine := ls.pos.Line
	ls.skipWhiteSpace()
	if ls.pos.Line != directiveLine {
		ls.lexError("Empty directive")
	}
	var buff bytes.Buffer
	directiveChar, _, err := ls.brdr.ReadRune()
	if err != nil && err != io.EOF {
		ls.lexError(fmt.Sprintf("Error reading directive %s", err.Error()))
	}
	if isAlpha(directiveChar) {
		for isAlpha(directiveChar) {
			buff.WriteRune(directiveChar)
			directiveChar, _, err = ls.brdr.ReadRune()
		}
		ls.brdr.UnreadRune()
		directive := buff.String()
		ls.sendTok(DIRECTIVE, directive)
		if directive == "include" {
			ls.readHeaderInclude()
		}
	} else {
		//wasnt a directive, error will be caught by
		//cpp or parser.
		ls.brdr.UnreadRune()
	}

}

func (ls *lexerState) readHeaderInclude() {
	var buff bytes.Buffer
	line := ls.pos.Line
	ls.skipWhiteSpace()
	if ls.pos.Line != line {
		ls.lexError("No header after include.")
	}
	opening, _, _ := ls.brdr.ReadRune()
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
		c, _, err := ls.brdr.ReadRune()
		if err == io.EOF {
			ls.lexError("EOF encountered in header include.")
		}
		if err != nil {
			ls.lexError(err.Error())
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
	first, _, _ := ls.brdr.ReadRune()
	if !isValidIdentStart(first) {
		panic("internal error")
	}
	buff.WriteRune(first)
	for {
		b, _, err := ls.brdr.ReadRune()
		if isAlphaNumeric(b) || b == '_' {
			buff.WriteRune(b)
		} else {
			if err == nil {
				ls.brdr.UnreadByte()
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
		r, _, err := ls.brdr.ReadRune()
		if !isWhiteSpace(r) {
			if err == nil {
				ls.brdr.UnreadRune()
			}
			break
		}
		if r == '\n' {
			ls.pos.Line += 1
			ls.pos.Col = 1
			ls.bol = true
		} else {
			ls.pos.Col += 1
		}
	}
}

func (ls *lexerState) readConstantInt() {
	var buff bytes.Buffer
	for {
		r, _, err := ls.brdr.ReadRune()
		if isNumeric(r) {
			buff.WriteRune(r)
		} else {
			if err == nil {
				ls.brdr.UnreadRune()
			}
			str := buff.String()
			ls.sendTok(INT_CONSTANT, str)
			break
		}
	}
}

func (ls *lexerState) readCString() {
	var buff bytes.Buffer
	first, _, err := ls.brdr.ReadRune()
	if err != nil {
		ls.lexError(err.Error())
	}

	if first != '"' {
		ls.lexError("internal error")
	}
	buff.WriteRune('"')

	escaped := false
	for {
		b, _, err := ls.brdr.ReadRune()
		if err == io.EOF {
			ls.lexError("Unterminated string literal.")
		}
		if err != nil {
			ls.lexError(err.Error())
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

func (ls *lexerState) isAtLineStart() bool {
	return ls.bol
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
	return b == ' ' || b == '\r' || b == '\n' || b == '\t'
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
