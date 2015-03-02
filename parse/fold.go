package parse

import (
	"fmt"
)

type ConstantValue interface{}

type ConstantGPtr struct {
	Label string
	Off   int64
	Type  CType
}

type ConstantPtr struct {
	Val  ConstantValue
	Off  int64
	Type CType
}

type ConstantString struct {
	Val   string
	Type  CType
	Label string
}

type ConstantArr struct {
	Inits map[int]ConstantValue
	Type  CType
}

type ConstantInt struct {
	Val  int64
	Type CType
}

func Fold(n Node) (ConstantValue, error) {
	switch n := n.(type) {
	case *Constant:
		return &ConstantInt{
			Type: n.Type,
			Val:  n.Val,
		}, nil
	case *String:
		return &ConstantString{
			Type:  &Ptr{PointsTo: CChar},
			Val:   n.Val,
			Label: n.Label,
		}, nil
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
			return &ConstantGPtr{
				Off:   0,
				Label: gsym.Label,
				Type:  n.Type,
			}, nil
		}
	default:

	}

	return nil, fmt.Errorf("not a valid constant value")
}
