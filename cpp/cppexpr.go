package cpp

import (
	"container/list"
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

   Arithmetic operators for most of C

   Identifiers that are not macros, which are all considered to be the number zero.
*/

type cppExprCtx struct {
	e         *list.Element
	isDefined func(string) bool
}

func (ctx *cppExprCtx) nextToken() *Token {
	if ctx.e == nil {
		return nil
	}
	tok := ctx.e.Value.(*Token)
	ctx.e = ctx.e.Next()
	return tok
}

func (ctx *cppExprCtx) peek() *Token {
	if ctx.e == nil {
		return nil
	}
	return ctx.e.Value.(*Token)
}

func parseCPPExprAtom(ctx *cppExprCtx) (int64, error) {
	toCheck := ctx.nextToken()
	if toCheck == nil {
		return 0, fmt.Errorf("expected integer, char, or defined but got nothing")
	}
	switch toCheck.Kind {
	case NOT:
		v, err := parseCPPExprAtom(ctx)
		if v == 0 {
			return 1, nil
		}
		return 0, err
	case BNOT:
		v, err := parseCPPExprAtom(ctx)
		if err != nil {
			return 0, err
		}
		return ^v, nil
	case SUB:
		v, err := parseCPPExprAtom(ctx)
		if err != nil {
			return 0, err
		}
		return -v, nil
	case ADD:
		v, err := parseCPPExprAtom(ctx)
		if err != nil {
			return 0, err
		}
		return v, nil
	case LPAREN:
		v, err := parseCPPExpr(ctx)
		if err != nil {
			return 0, err
		}
		rparen := ctx.nextToken()
		if rparen == nil || rparen.Kind != RPAREN {
			return 0, fmt.Errorf("unclosed parenthesis")
		}
		return v, nil
	case INT_CONSTANT:
		v, err := strconv.ParseInt(toCheck.Val, 0, 64)
		if err != nil {
			return 0, fmt.Errorf("internal error parsing int constant")
		}
		return v, nil
	case CHAR_CONSTANT:

		return 0, fmt.Errorf("unimplemented char literal in cpp expression")
	case IDENT:
		if toCheck.Val == "defined" {
			toCheck = ctx.nextToken()
			if toCheck == nil {
				return 0, fmt.Errorf("expected ( or an identifier but got nothing")
			}
			switch toCheck.Kind {
			case LPAREN:
				toCheck = ctx.nextToken()
				rparen := ctx.nextToken()
				if rparen == nil || rparen.Kind != RPAREN {
					return 0, fmt.Errorf("malformed defined check, missing )")
				}
			case IDENT:
				//calls isDefined as intended
			default:
				return 0, fmt.Errorf("malformed defined statement at %s", toCheck.Pos)
			}
		}
	default:
		return 0, fmt.Errorf("expected integer, char, or defined but got %s", toCheck.Val)
	}
	if toCheck == nil {
		return 0, fmt.Errorf("expected identifier but got nothing")
	}
	if ctx.isDefined(toCheck.Val) {
		return 1, nil
	}
	return 0, nil
}

func evalCPPBinop(ctx *cppExprCtx, k TokenKind, l int64, r int64) (int64, error) {
	switch k {
	case LOR:
		if l != 0 || r != 0 {
			return 1, nil
		}
		return 0, nil
	case LAND:
		if l != 0 && r != 0 {
			return 1, nil
		}
		return 0, nil
	case OR:
		return l | r, nil
	case XOR:
		return l ^ r, nil
	case AND:
		return l & r, nil
	case ADD:
		return l + r, nil
	case SUB:
		return l - r, nil
	case MUL:
		return l * r, nil
	case SHR:
		return l >> uint64(r), nil
	case SHL:
		return l << uint64(r), nil
	case QUO:
		if r == 0 {
			return 0, fmt.Errorf("divide by zero in expression")
		}
		return l / r, nil
	case REM:
		if r == 0 {
			return 0, fmt.Errorf("divide by zero in expression")
		}
		return l % r, nil
	case EQL:
		if l == r {
			return 1, nil
		}
		return 0, nil
	case LSS:
		if l < r {
			return 1, nil
		}
		return 0, nil
	case GTR:
		if l > r {
			return 1, nil
		}
		return 0, nil
	case LEQ:
		if l <= r {
			return 1, nil
		}
		return 0, nil
	case GEQ:
		if l >= r {
			return 1, nil
		}
		return 0, nil
	case NEQ:
		if l != r {
			return 1, nil
		}
		return 0, nil
	case COMMA:
		return r, nil
	default:
		return 0, fmt.Errorf("internal error %s", k)
	}
}

func parseCPPTernary(ctx *cppExprCtx) (int64, error) {
	cond, err := parseCPPBinop(ctx)
	if err != nil {
		return 0, err
	}
	t := ctx.peek()
	var a, b int64
	if t != nil && t.Kind == QUESTION {
		ctx.nextToken()
		a, err = parseCPPExpr(ctx)
		if err != nil {
			return 0, err
		}
		colon := ctx.nextToken()
		if colon == nil || colon.Kind != COLON {
			return 0, fmt.Errorf("ternary without :")
		}
		b, err = parseCPPExpr(ctx)
		if err != nil {
			return 0, err
		}
		if cond != 0 {
			return a, nil
		}
		return b, nil
	}
	return cond, nil
}

func parseCPPComma(ctx *cppExprCtx) (int64, error) {
	v, err := parseCPPTernary(ctx)
	if err != nil {
		return 0, err
	}
	for {
		t := ctx.peek()
		if t == nil || t.Kind != COMMA {
			break
		}
		ctx.nextToken()
		v, err = parseCPPTernary(ctx)
		if err != nil {
			return 0, err
		}
	}
	return v, nil
}

func getPrec(k TokenKind) int {
	switch k {
	case MUL, REM, QUO:
		return 10
	case ADD, SUB:
		return 9
	case SHR, SHL:
		return 8
	case LSS, GTR, GEQ, LEQ:
		return 7
	case EQL, NEQ:
		return 6
	case AND:
		return 5
	case XOR:
		return 4
	case OR:
		return 3
	case LAND:
		return 2
	case LOR:
		return 1
	}
	return -1
}

// This is the precedence climbing algorithm, simplified because
// all the operators are left associative. The CPP doesn't
// deal with assignment operators.
func parseCPPBinop_1(ctx *cppExprCtx, prec int) (int64, error) {
	l, err := parseCPPExprAtom(ctx)
	if err != nil {
		return 0, err
	}
	for {
		t := ctx.peek()
		if t == nil {
			break
		}
		p := getPrec(t.Kind)
		if p == -1 {
			break
		}
		if p < prec {
			break
		}
		ctx.nextToken()
		r, err := parseCPPBinop_1(ctx, p+1)
		if err != nil {
			return 0, err
		}
		l, err = evalCPPBinop(ctx, t.Kind, l, r)
		if err != nil {
			return 0, err
		}
	}
	return l, nil
}

func parseCPPBinop(ctx *cppExprCtx) (int64, error) {
	return parseCPPBinop_1(ctx, 0)
}

func parseCPPExpr(ctx *cppExprCtx) (int64, error) {
	return parseCPPComma(ctx)
}

func evalIfExpr(isDefined func(string) bool, tl *tokenList) (int64, error) {
	ctx := &cppExprCtx{isDefined: isDefined, e: tl.l.Front()}
	ret, err := parseCPPExpr(ctx)
	if err != nil {
		return 0, err
	}
	t := ctx.nextToken()
	if t != nil {
		return 0, fmt.Errorf("stray token %s", t.Val)
	}
	return ret, nil
}
