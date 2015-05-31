package parse

import (
	"fmt"
	"github.com/andrewchambers/cc/cpp"
)

type ConstantGPtr struct {
	Pos      cpp.FilePos
	PtrLabel string
	Offset   int64
	Type     CType
}

func (c *ConstantGPtr) GetPos() cpp.FilePos {
	return c.Pos
}

// To fold a node means to compute the simplified form which can replace it without
// changing the meaning of the program.
func (p *parser) fold(n Node) (Node, error) {
	switch n := n.(type) {
	case *Constant:
		return n, nil
	case *String:
		return n, nil
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
				Pos:      n.GetPos(),
				Offset:   0,
				PtrLabel: gsym.Label,
				Type:     n.Type,
			}, nil
		}
	default:
	}
	fmt.Printf("%#v\n", n)
	panic("...")
	return nil, fmt.Errorf("not a valid constant value")
}
