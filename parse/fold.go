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
	if n == nil {
		return nil, fmt.Errorf("not a valid constant expression")
	}
	switch n := n.(type) {
	case *Constant:
		return &FoldedConstant{
			Val:   n.Val,
			Label: "",
			Type:  n.Type,
		}, nil
	default:
		return nil, fmt.Errorf("not a valid constant expression")
	}
}
