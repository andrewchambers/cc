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

func parseCPPExprAtom(isDefined func(string) bool, nextToken func() *Token, onError func(error)) int64 {
	toCheck := nextToken()
	if toCheck == nil {
		onError(fmt.Errorf("expected integer, char, or defined but got nothing"))
		return 0
	}
	switch toCheck.Kind {
	case INT_CONSTANT:
		v, err := strconv.ParseInt(toCheck.Val, 0, 64)
		if err != nil {
			onError(nil)
		}
		return v
	case CHAR_CONSTANT:
		onError(fmt.Errorf("unimplemented char literal in cpp expression"))
		return 0
	case IDENT:
		if toCheck.Val == "defined" {
			toCheck = nextToken()
			if toCheck == nil {
				onError(fmt.Errorf("expected ( or an identifier but got nothing"))
				return 0
			}
			switch toCheck.Kind {
			case LPAREN:
				toCheck = nextToken()
				rparen := nextToken()
				if rparen == nil || rparen.Kind != RPAREN {
					onError(fmt.Errorf("malformed defined check, missing )"))
					return 0
				}
			case IDENT:
				//calls isDefined as intended
			default:
				onError(fmt.Errorf("malformed defined statement at %s", toCheck.Pos))
				return 0
			}
		}
	default:
		onError(fmt.Errorf("expected integer, char, or defined but got %s", toCheck.Val))
		return 0
	}
	if toCheck == nil {
		onError(fmt.Errorf("expected identifier but got nothing"))
		return 0
	}
	if isDefined(toCheck.Val) {
		return 1
	}
	return 0
}

//Eval the expression using precedence climbing
func evalIfExpr(isDefined func(string) bool, nextToken func() *Token, onError func(error)) int64 {
	return parseCPPExpr(isDefined, nextToken, onError, 0)
}

func parseCPPExpr(isDefined func(string) bool, nextToken func() *Token, onError func(error), min_prec int) int64 {
	result := parseCPPExprAtom(isDefined, nextToken, onError)
	//while cur token is a binary operator with precedence >= min_prec:
	//prec, assoc = precedence and associativity of current token
	//if assoc is left:
	//  next_min_prec = prec + 1
	//else:
	// next_min_prec = prec
	//rhs = compute_expr(next_min_prec)
	//result = compute operator(result, rhs)
	return result
}
