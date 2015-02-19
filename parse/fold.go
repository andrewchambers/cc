package parse

import (
	"fmt"
)

type FoldedConstant struct {
	Val   int64
	Label string
	Type  CType
}

func Fold(n Node) (*FoldedConstant, error) {
	switch n := n.(type) {
	case *Unop:
		switch n.Op {
		case '&':
			ident, ok := n.Operand.(*Ident)
			if !ok {
				// XXX &foo[CONST] is valid.
				return nil, fmt.Errorf("'&' requires a valid identifier")
			}
			gsym, ok := ident.Sym.(*GSymbol)
			if !ok {
				return nil, fmt.Errorf("'&' requires a static or global identifier")
			}
			return &FoldedConstant{
				Val:   0,
				Label: gsym.Label,
				Type:  n.Type,
			}, nil
		case '-':
			operand, err := Fold(n.Operand)
			if err != nil {
				return nil, err
			}
			if IsIntType(operand.Type) {
				return &FoldedConstant{
					Val:  -operand.Val,
					Type: operand.Type,
				}, nil
			}
		case '+':
			operand, err := Fold(n.Operand)
			if err != nil {
				return nil, err
			}
			if IsIntType(operand.Type) {
				return operand, nil
			}
		}
	case *Binop:
		l, err := Fold(n.L)
		if err != nil {
			return nil, err
		}
		r, err := Fold(n.R)
		if err != nil {
			return nil, err
		}
		switch n.Op {
		case '+':
			if IsIntType(l.Type) && IsIntType(r.Type) {
				if l.Label != "" || r.Label != "" {
					panic("internal error.")
				}
				return &FoldedConstant{
					Val:  l.Val + r.Val,
					Type: l.Type,
				}, nil
			}
		case '*':
			if IsIntType(l.Type) && IsIntType(r.Type) {
				if l.Label != "" || r.Label != "" {
					panic("internal error.")
				}
				return &FoldedConstant{
					Val:  l.Val * r.Val,
					Type: l.Type,
				}, nil
			}
		case '/':
			if IsIntType(l.Type) && IsIntType(r.Type) {
				if l.Label != "" || r.Label != "" {
					panic("internal error.")
				}
				if r.Val == 0 {
					return nil, fmt.Errorf("division by zero.")
				}
				return &FoldedConstant{
					Val:  l.Val / r.Val,
					Type: l.Type,
				}, nil
			}
		case '-':
			if IsIntType(l.Type) && IsIntType(r.Type) {
				if l.Label != "" || r.Label != "" {
					panic("internal error.")
				}
				return &FoldedConstant{
					Val:  l.Val - r.Val,
					Type: l.Type,
				}, nil
			}
		}
	case *Constant:
		return &FoldedConstant{
			Val:   n.Val,
			Label: "",
			Type:  n.Type,
		}, nil
	default:
		return nil, fmt.Errorf("not a valid constant expression")
	}
	panic("internal error.")
}
