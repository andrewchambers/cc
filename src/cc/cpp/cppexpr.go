package cpp

import (
	"fmt"
	"strconv"
)

/*
Implements the expression parsing and evaluation for #if statements


Note that "defined name" and "define(name)" are handled before this part of code.

#if expression
    controlled text
#endif

expression may be:

Integer constants.

Character constants, which are interpreted as they would be in normal code.

Arithmetic operators for addition, subtraction, multiplication, division,
bitwise operations, shifts, comparisons, and logical operations (&& and ||).

Identifiers that are not macros, which are all considered to be the number zero.

*/

func parseCPPExprAtom(isDefined func(string) bool, nextToken func() *Token) (int64, error) {
	toCheck := nextToken()
	if toCheck == nil {
		return 0, fmt.Errorf("expected integer, char, or defined but got nothing")
	}
	switch toCheck.Kind {
	case INT_CONSTANT:
		return strconv.ParseInt(toCheck.Val, 0, 64)
	case CHAR_CONSTANT:
		return 0, fmt.Errorf("unimplemented char literal in cpp expression")
	case IDENT:
		if toCheck.Val == "defined" {
			toCheck = nextToken()
			if toCheck == nil {
				return 0, fmt.Errorf("expected ( or an identifier but got nothing")
			}
			switch toCheck.Kind {
			case LPAREN:
				toCheck = nextToken()
				rparen := nextToken()
				if rparen == nil || rparen.Kind != RPAREN {
					return 0, fmt.Errorf("malformed defined check, missing )")
				}
			case IDENT:
				//calls isDefined as intended
			default:
				fmt.Errorf("malformed defined statement at %s", toCheck.Pos)
			}
		}
	default:
		return 0, fmt.Errorf("expected integer, char, or defined but got %s", toCheck.Val)
	}
	if toCheck == nil {
		return 0, fmt.Errorf("expected identifier but got nothing")
	}
	if isDefined(toCheck.Val) {
		return 1, nil
	}
	return 0, nil
}

//Eval the expression using precedence climbing
func evalIfExpr(isDefined func(string) bool, tokens *tokenList) (int64, error) {
	e := tokens.front()
	nextToken := func() *Token {
		if e == nil {
			return nil
		}
		e = e.Next()
		r := e.Value.(*Token)
		return r
	}
	return parseCPPExpr(isDefined, nextToken, 0)
}

func parseCPPExpr(isDefined func(string) bool, nextToken func() *Token, min_prec int) (int64, error) {
	result, err := parseCPPExprAtom(isDefined, nextToken)
	//while cur token is a binary operator with precedence >= min_prec:
	//prec, assoc = precedence and associativity of current token
	//if assoc is left:
	//  next_min_prec = prec + 1
	//else:
	// next_min_prec = prec
	//rhs = compute_expr(next_min_prec)
	//result = compute operator(result, rhs)
	return result, err
}
