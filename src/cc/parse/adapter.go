package parse

import (
	"cc/cpp"
	"fmt"
)

//adapter converts the lexer to the format expected by go yacc.
type adapter struct {
	tokenStream chan *cpp.Token
	lastToken   *cpp.Token
}

//This is needed so the lexer doesn't depend on the parser.
func cppTok2yaccTok(t cpp.TokenKind) int {
	switch t {
	case cpp.IDENT:
		return IDENTIFIER
	case cpp.INT_CONSTANT, cpp.FLOAT_CONSTANT, cpp.CHAR_CONSTANT:
		return CONSTANT
	case cpp.STRING:
		return STRING_LITERAL
	case cpp.LPAREN:
		return '('
	case cpp.RPAREN:
		return ')'
		/*
			operator_beg
			// Operators and delimiters
			ADD      // +
			SUB      // -
			MUL      // *
			QUO      // /
			REM      // %
			QUESTION // ?

			AND // &
			OR  // |
			XOR // ^
			SHL // <<
			SHR // >>

			ADD_ASSIGN // +=
			SUB_ASSIGN // -=
			MUL_ASSIGN // *=
			QUO_ASSIGN // /=
			REM_ASSIGN // %=

			AND_ASSIGN // &=
			OR_ASSIGN  // |=
			XOR_ASSIGN // ^=
			SHL_ASSIGN // <<=
			SHR_ASSIGN // >>=

			LAND  // &&
			LOR   // ||
			ARROW // ->
			INC   // ++
			DEC   // --

			EQL    // ==
			LSS    // <
			GTR    // >
			ASSIGN // =
			NOT    // !
			BNOT   // ~

			NEQ      // !=
			LEQ      // <=
			GEQ      // >=
			ELLIPSIS // ...

			LPAREN // (
			LBRACK // [
			LBRACE // {
			COMMA  // ,
			PERIOD // .

			RPAREN    // )
			RBRACK    // ]
			RBRACE    // }
			SEMICOLON // ;
			COLON     // :
			operator_end

			keyword_beg
			// Keywords
			BREAK
			CASE
			DO
			CONST
			CONTINUE
			DEFAULT
			ELSE
			FOR
			WHILE
			GOTO
			IF
			RETURN
			STRUCT
			SWITCH
			TYPEDEF
			SIZEOF
			VOID
			CHAR
			INT
			FLOAT
			DOUBLE
			SIGNED
			UNSIGNED
			LONG
		*/
	}
	panic("Internal error!")
}

func newAdapter(ts chan *cpp.Token) yyLexer {
	return &adapter{ts, nil}
}

func (a *adapter) Lex(lval *yySymType) int {
	t := <-a.tokenStream
	a.lastToken = t
	return cppTok2yaccTok(t.Kind)
}

func (a *adapter) Error(e string) {
	fmt.Println(fmt.Sprintf("%s at %s", e, a.lastToken))
}
