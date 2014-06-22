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

func (ls *lexerState) lexError(e error) {
	eWithPos := fmt.Sprintf("Error while reading %s. %s", ls.pos, e.Error())
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
			_ = e.(breakout) // Will re-panic if not a parse error.
		}

	}()

	first, _, err := ls.brdr.ReadRune()

	for {
		if err == io.EOF {
			break
		}
		if err != nil {
			ls.lexError(err)
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
				ls.readDirective()
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
						endChar, _, err := ls.brdr.ReadRune()
						if err == io.EOF {
							ls.lexError(fmt.Errorf("Unclosed comment."))
						}
						if err != nil {
							ls.lexError(err)
						}
						if endChar == '*' {
							closeBar, _, _ := ls.brdr.ReadRune()
							if closeBar == '/' {
								break
							}
						}
					}
				} else if second == '/' { // C++ comment.
					for {
						c, _, err := ls.brdr.ReadRune()
						if err != nil {
							break
						}
						if c == '\n' {
							break
						}
					}
				} else {
					ls.sendTok(QUO, "/")
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

	if !ls.isAtLineStart() {
		ls.lexError(fmt.Errorf("Directive not at beginning of line."))
	}

	//Skip
	for {
		ws, _, err := ls.brdr.ReadRune()
		if err != nil {
			ls.lexError(err)
		}

		if ws == ' ' || ws == '\t' {
			continue
		}

		if ws == '\n' {
			ls.lexError(fmt.Errorf("Empty directive"))
		}

		break
	}

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
		ls.lexError(err)
	}

	if first != '"' {
		ls.lexError(fmt.Errorf("internal error"))
	}
	buff.WriteRune('"')

	escaped := false
	for {
		b, _, err := ls.brdr.ReadRune()
		if err == io.EOF {
			ls.lexError(fmt.Errorf("Unterminated string literal."))
		}
		if err != nil {
			ls.lexError(err)
		}
		if b == '"' && !escaped {
			break
		}
		if !escaped {
			if b == '\\' {
				escaped = true
			}
		} else {
			escaped = false
		}
	}
	ls.sendTok(STRING, "")
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
