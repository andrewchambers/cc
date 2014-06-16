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
	//If this channel recieves a value, the lexing goroutines should close
	//its output channel and its error channel and terminate.
	cancel chan struct{}
	errors chan error
	stream chan *Token
}

//Lex starts a goroutine which lexes the contents of the reader.
//fname is used for error messages when showing the source location.
//No preprocessing is done, this is just pure reading of the unprocessed
//source file. returns 2 channels, tokens and errors.
//On error the goroutine will return.
//Otherwise the goroutine will not stop until all tokens are read
//and nil is returned from the channel.
func Lex(fname string, r io.Reader) (chan *Token, chan error) {
	ls := new(lexerState)
	ls.pos.File = fname
	ls.pos.Line = 1
	ls.pos.Col = 1
	ls.stream = make(chan *Token)
	ls.errors = make(chan error)
	ls.brdr = bufio.NewReader(r)
	go ls.lex()
	return ls.stream, ls.errors
}

func (ls *lexerState) lexError(e error) {
	eWithPos := fmt.Errorf("Error while reading %s. %s", ls.pos, e.Error())
	ls.errors <- eWithPos
	//recover exits the lexer cleanly
	panic("error while lexing.")
}

func (ls *lexerState) lex() {

	//This recovery happens if lexError is called.
	defer func() {
		recover()
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
				ls.sendTok('#', "#")
			case '!':
				ls.sendTok('!', "!")
			case '?':
				ls.sendTok('?', "?")
			case ':':
				ls.sendTok(':', ":")
			case '"':
				ls.brdr.UnreadByte()
				ls.readCString()
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
			case '.':
				ls.sendTok('.', ".")
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
}

func (ls *lexerState) sendTok(kind TokenKind, val string) {
	var tok Token
	tok.Kind = kind
	tok.Val = val
	tok.Pos.Line = ls.pos.Line
	tok.Pos.Col = ls.pos.Col
	tok.Pos.File = ls.pos.File
	ls.stream <- &tok
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
			ls.pos.Col += len(str)
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
			ls.sendTok(TOK_CONSTANT_INT, str)
			ls.pos.Col += len(str)
			break
		}
	}
}

func (ls *lexerState) readCString() {
	first, _, err := ls.brdr.ReadRune()

	if err != nil {
		ls.lexError(err)
	}

	if first != '"' {
		ls.lexError(fmt.Errorf("internal error"))
	}

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
	ls.sendTok(TOK_STRING, "")
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
