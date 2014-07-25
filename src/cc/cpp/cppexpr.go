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

type cppExprCtx struct {
	isDefined func(string) bool
	nextToken func() *Token
	onError   func(error)
}

func (ctx *cppExprCtx) peek() *Token {
	t := ctx.nextToken()
	oldNext := ctx.nextToken
	ctx.nextToken = func() *Token {
		ctx.nextToken = oldNext
		return t
	}
	return t
}

func parseCPPExprAtom(ctx *cppExprCtx) int64 {
	toCheck := ctx.nextToken()
	if toCheck == nil {
		ctx.onError(fmt.Errorf("expected integer, char, or defined but got nothing"))
		return 0
	}
	switch toCheck.Kind {
	case NOT:
		v := parseCPPExprAtom(ctx)
		if v == 0 {
			return 1
		}
		return 0
	case BNOT:
		v := parseCPPExprAtom(ctx)
		return ^v
	case SUB:
		v := parseCPPExprAtom(ctx)
		return -v
	case ADD:
		v := parseCPPExprAtom(ctx)
		return v
	case LPAREN:
		v := parseCPPExpr(ctx)
		rparen := ctx.nextToken()
		if rparen == nil || rparen.Kind != RPAREN {
			ctx.onError(fmt.Errorf("unclosed parenthesis"))
		}
		return v
	case INT_CONSTANT:
		v, err := strconv.ParseInt(toCheck.Val, 0, 64)
		if err != nil {
			ctx.onError(fmt.Errorf("internal error parsing int constant"))
		}
		return v
	case CHAR_CONSTANT:
		ctx.onError(fmt.Errorf("unimplemented char literal in cpp expression"))
		return 0
	case IDENT:
		if toCheck.Val == "defined" {
			toCheck = ctx.nextToken()
			if toCheck == nil {
				ctx.onError(fmt.Errorf("expected ( or an identifier but got nothing"))
				return 0
			}
			switch toCheck.Kind {
			case LPAREN:
				toCheck = ctx.nextToken()
				rparen := ctx.nextToken()
				if rparen == nil || rparen.Kind != RPAREN {
					ctx.onError(fmt.Errorf("malformed defined check, missing )"))
					return 0
				}
			case IDENT:
				//calls isDefined as intended
			default:
				ctx.onError(fmt.Errorf("malformed defined statement at %s", toCheck.Pos))
				return 0
			}
		}
	default:
		ctx.onError(fmt.Errorf("expected integer, char, or defined but got %s", toCheck.Val))
		return 0
	}
	if toCheck == nil {
		ctx.onError(fmt.Errorf("expected identifier but got nothing"))
		return 0
	}
	if ctx.isDefined(toCheck.Val) {
		return 1
	}
	return 0
}

func evalCPPBinop(ctx *cppExprCtx, k TokenKind, l int64, r int64) int64 {
	switch k {
	case LOR:
		if l != 0 || r != 0 {
			return 1
		}
		return 0
	case LAND:
		if l != 0 && r != 0 {
			return 1
		}
		return 0
	case OR:
		return l | r
	case XOR:
		return l ^ r
	case AND:
		return l & r
	case ADD:
		return l + r
	case SUB:
		return l - r
	case MUL:
		return l * r
	case SHR:
		return l >> uint64(r)
	case SHL:
		return l << uint64(r)
	case QUO:
		if r == 0 {
			ctx.onError(fmt.Errorf("divide by zero in expression"))
			return 0
		}
		return l / r
	case REM:
		if r == 0 {
			ctx.onError(fmt.Errorf("divide by zero in expression"))
			return 0
		}
		return l % r
	case EQL:
		if l == r {
			return 1
		}
		return 0
	case LSS:
		if l < r {
			return 1
		}
		return 0
	case GTR:
		if l > r {
			return 1
		}
		return 0
	case LEQ:
		if l <= r {
			return 1
		}
		return 0
	case GEQ:
		if l >= r {
			return 1
		}
		return 0
	case NEQ:
		if l != r {
			return 1
		}
		return 0
	case COMMA:
		return r
	default:
		ctx.onError(fmt.Errorf("internal error %s", k))
	}
	return 0
}

func createCPPExprParseFunc(term func(*cppExprCtx) int64, kinds []TokenKind) func(*cppExprCtx) int64 {
	return func(ctx *cppExprCtx) int64 {
		l := term(ctx)
		for {
			t := ctx.peek()
			if t == nil {
				break
			}
			match := false
			idx := 0
			for idx = range kinds {
				if t.Kind == kinds[idx] {
					match = true
					break
				}
			}
			if !match {
				break
			}
			ctx.nextToken()
			r := term(ctx)
			l = evalCPPBinop(ctx, kinds[idx], l, r)
		}
		return l
	}
}

func parseCPPTernary(ctx *cppExprCtx) int64 {
	cond := parseCPPComma(ctx)
	t := ctx.peek()
	var a, b int64
	if t != nil && t.Kind == QUESTION {
		ctx.nextToken()
		a = parseCPPTernary(ctx)
		colon := ctx.nextToken()
		if colon == nil || colon.Kind != COLON {
			ctx.onError(fmt.Errorf("ternary without :"))
		}
		b = parseCPPTernary(ctx)

		if cond != 0 {
			return a
		}
		return b
	}
	return cond
}

var parseCPPBinop func(*cppExprCtx) int64
var parseCPPComma func(*cppExprCtx) int64

var cppExprBinopPrecTable = [...][]TokenKind{
	{MUL, REM, QUO},
	{ADD, SUB},
	{SHR, SHL},
	{LSS, GTR, GEQ, LEQ},
	{EQL, NEQ},
	{AND},
	{XOR},
	{OR},
	{LAND},
	{LOR},
}

func init() {
	parseCPPBinop = parseCPPExprAtom
	for idx := range cppExprBinopPrecTable {
		parseCPPBinop = createCPPExprParseFunc(parseCPPBinop, cppExprBinopPrecTable[idx])
	}
	parseCPPComma = createCPPExprParseFunc(parseCPPBinop, []TokenKind{COMMA})
}

func parseCPPExpr(ctx *cppExprCtx) int64 {
	result := parseCPPTernary(ctx)
	return result
}

func evalIfExpr(isDefined func(string) bool, nextToken func() *Token, onError func(error)) int64 {
	ctx := &cppExprCtx{isDefined: isDefined, nextToken: nextToken, onError: onError}
	ret := parseCPPExpr(ctx)
	t := ctx.nextToken()
	if t != nil {
		ctx.onError(fmt.Errorf("stray token %s", t.Val))
	}
	return ret
}
