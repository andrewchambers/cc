package cpp

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
)

type lexerState struct {
	file string
	brdr *bufio.Reader
	line int
	col  int
	//If this channel recieves a value, the lexing goroutines should close
	//its output channel and its error channel and terminate.
	cancel chan struct{}
	errors chan error
	stream chan *Token
}

//Lex starts a goroutine which lexes the contents of the reader.
//fname is used for error messages when showing the source location.
//No preprocessing is done, this is just pure reading of the unprocessed
//source file. An error will be sent to the error channel in the event of
//a lexing error, otherwise tokens will be sent to the Token channel
func Lex(fname string, r io.Reader, cancel chan struct{}) (chan error, chan *Token) {
	ls := new(lexerState)
	ls.file = fname
	ls.line = 1
	ls.col = 1
	ls.stream = make(chan *Token)
	ls.errors = make(chan error)
	ls.brdr = bufio.NewReader(r)
	ls.cancel = cancel
	go ls.lex()
	return ls.errors, ls.stream
}

func (ls *lexerState) lexError(e error) {
	eWithPos := fmt.Errorf("Error while reading file %s line %d column %d. %s", ls.file, ls.line, ls.col, e.Error())
	close(ls.stream)
	ls.errors <- eWithPos
	close(ls.errors)
	//Easy way to quit from an error.
	panic("error while lexing.")
}

func (ls *lexerState) lex() {

	//This recovery happens if lexError is called.
	defer func() {
		recover()
	}()

	first, _, err := ls.brdr.ReadRune()

	for {

		//If we get cancelled, quit as if we reached EOF
		select {
		case <-ls.cancel:
			ls.cancelErr()
		default:
		}

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
				ls.sendTok('#', "#")
			case '!':
				ls.sendTok('!', "!")
			case '?':
				ls.sendTok('?', "?")
			case ':':
				ls.sendTok(':', ":")
			//case '"':
			//	source.UnreadByte()
			//	ls.stream <- readCString(source)
			case '(':
				ls.sendTok('(', "(")
			case ')':
				ls.sendTok(')', ")")
			case '{':
				ls.sendTok('{', "{")
			case '}':
				ls.sendTok('}', "}")
			case '[':
				ls.sendTok('[', "[")
			case ']':
				ls.sendTok(']', "]")
			case '<':
				ls.sendTok('<', "<")
			case '>':
				ls.sendTok('>', ">")
			case '+':
				second, _, _ := ls.brdr.ReadRune()
				if second == '+' {
					ls.sendTok(TOK_INC_OP, "++")
					break
				}
				ls.brdr.UnreadRune()
				ls.sendTok('+', "+")
			case '-':
				second, _, _ := ls.brdr.ReadRune()
				if second == '>' {
					ls.sendTok(TOK_PTR_OP, "->")
					break
				}
				ls.brdr.UnreadRune()
				ls.sendTok('-', "-")
			case ',':
				ls.sendTok(',', ",")
			case '*':
				ls.sendTok('*', "*")
			case '/':
				second, _, _ := ls.brdr.ReadRune()
				if second == '*' { // C comment
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
				} else {
					ls.sendTok('/', "/")
				}
			case '|':
				second, _, _ := ls.brdr.ReadRune()
				if second == '|' {
					ls.sendTok(TOK_OR_OP, "||")
					break
				}
				ls.brdr.UnreadRune()
				ls.sendTok('|', "|")
			case '&':
				second, _, _ := ls.brdr.ReadRune()
				if second == '&' {
					ls.sendTok(TOK_AND_OP, "&&")
					break
				}
				ls.brdr.UnreadRune()
				ls.sendTok('&', "&")
			case '=':
				second, _, _ := ls.brdr.ReadRune()
				if second == '=' {
					ls.sendTok(TOK_EQ_OP, "==")
					break
				}
				ls.brdr.UnreadRune()
				ls.sendTok('=', "=")
			case ';':
				ls.sendTok(';', ";")
			default:
				ls.lexError(fmt.Errorf("Internal Error - bad char code '%d'", first))
			}
		}
		first, _, err = ls.brdr.ReadRune()
	}
	close(ls.stream)
	close(ls.errors)
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

func (ls *lexerState) sendTok(kind TokenKind, val string) {
	var tok Token
	tok.Kind = kind
	tok.Val = val
	tok.Pos.Line = ls.line
	tok.Pos.Col = ls.col
	tok.Pos.File = ls.file
	select {
	case <-ls.cancel:
		ls.cancelErr()
	case ls.stream <- &tok:
		break
	}
}

func (ls *lexerState) cancelErr() {
	ls.lexError(fmt.Errorf("Lexing cancelled"))
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
				tokType = TOK_IDENTIFIER
			}

			ls.sendTok(tokType, str)
			ls.col += len(str)
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
			ls.line += 1
			ls.col = 1
		} else {
			ls.col += 1
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
			ls.sendTok(TOK_CONSTANT_INT, str)
			ls.col += len(str)
			break
		}
	}
}

/*func readCString(source *bufio.Reader) *cparse.Token {
	first, _ := source.ReadByte()
	if first != '"' {
		panic("internal error")
	}

	escaped := false
	for {
		b, _ := source.ReadByte()
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
	//XXX
	return makeTok(cparse.STRING_LITERAL, "", 0)
}*/
