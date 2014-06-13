package cpp

import (
	"bufio"
	"bytes"
	"io"
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
	ls.stream = make(chan *Token)
	ls.brdr = bufio.NewReader(r)
	go ls.lex()
	return ls.stream
}

func (ls *lexerState) lex() {

	first, _, _ := ls.brdr.ReadRune()

	for {
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
		}
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

func (ls *lexerState) sendTok(kind TokenKind, val string, line, col int) {
	var tok Token
	tok.Kind = kind
	tok.Val = val
	tok.Pos.Line = line
	tok.Pos.Col = col
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

			ls.sendTok(tokType, str, ls.line, ls.col)
			ls.col += len(str)
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
			ls.col = 0
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
			ls.sendTok(TOK_CONSTANT_INT, str, ls.line, ls.col)
			ls.col += len(str)
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
