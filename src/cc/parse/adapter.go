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
	case cpp.LBRACE:
		return '{'
	case cpp.RBRACE:
		return '}'
	case cpp.SEMICOLON:
		return ';'
	case cpp.ADD:
		return '+'
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
	*/

	case cpp.BREAK:
		return BREAK
	case cpp.CASE:
		return CASE
	case cpp.DO:
		return DO
	case cpp.CONST:
		return CONST
	case cpp.CONTINUE:
		return CONTINUE
	case cpp.DEFAULT:
		return DEFAULT
	case cpp.ELSE:
		return ELSE
	case cpp.FOR:
		return FOR
	case cpp.WHILE:
		return WHILE
	case cpp.GOTO:
		return GOTO
	case cpp.IF:
		return IF
	case cpp.RETURN:
		return RETURN
	case cpp.STRUCT:
		return STRUCT
	case cpp.SWITCH:
		return SWITCH
	case cpp.TYPEDEF:
		return TYPEDEF
	case cpp.SIZEOF:
		return SIZEOF
	case cpp.VOID:
		return VOID
	case cpp.CHAR:
		return CHAR
	case cpp.INT:
		return INT
	case cpp.FLOAT:
		return FLOAT
	case cpp.DOUBLE:
		return DOUBLE
	case cpp.SIGNED:
		return SIGNED
	case cpp.UNSIGNED:
		return UNSIGNED
	case cpp.LONG:
		return LONG

	}
	panic("Internal error - unhandled case " + t.String())
}

func newAdapter(ts chan *cpp.Token) yyLexer {
	return &adapter{ts, nil}
}

func (a *adapter) Lex(lval *yySymType) int {
	t := <-a.tokenStream
	a.lastToken = t
	if t == nil {
		return 0
	}
	return cppTok2yaccTok(t.Kind)
}

func (a *adapter) Error(e string) {
	fmt.Println(fmt.Sprintf("%s at %s", e, a.lastToken))
}
