package cpp

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
)

type lexerState struct {
	file   string
	brdr   *bufio.Reader
	line   int
	col    int
	stream chan *Token
}

//Lex starts a goroutine which lexes the contents of the reader.
//fname is used for error messages when showing the source location.
//No preprocessing is done, this is just pure reading of the unprocessed
//source file.
func Lex(fname string, r io.Reader) chan *Token {
	ls := new(lexerState)
	ls.file = fname
	ls.line = 1
	ls.col = 1
	ls.stream = make(chan *Token)
	ls.brdr = bufio.NewReader(r)
	go ls.lex()
	return ls.stream
}

func (ls *lexerState) lexError(e error) {
	fmt.Fprintf(os.Stderr, "Error while reading file %s line %d column %d. %s", ls.file, ls.line, ls.col, e.Error())
	os.Exit(1)
}

func (ls *lexerState) lex() {
	first, _, err := ls.brdr.ReadRune()
	for {
		if err == io.EOF {
			break
		}
		if err != nil {
			ls.lexError(err)
			break
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
				break
			}
		}
		first, _, err = ls.brdr.ReadRune()
	}
	close(ls.stream)
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
