package parse

import (
	"fmt"
	"github.com/andrewchambers/cc/cpp"
)

// Anything that can be used to staticially initialize a global variable is
// represented by the Data
type StaticData interface{}

type SymbolicData struct {
	DataDeps []StaticData
	Label    string
	Type     CType
	Sym      string
	Val      int64
}

type ConstantData struct {
	DataDeps []StaticData
	Label    string
	Type     CType
	Val      int64
}

type StringData struct {
	Label string
	Val   string
}

type SeqDataEnt struct {
	offset int64
	E      StaticData
}

type SeqData struct {
	DataDeps []StaticData
	Label    string
	Type     CType
	Ents     []SeqDataEnt
}

func nodeToStaticData(ty CType, n Node) (StaticData, error) {
	switch n := n.(type) {
	case *Constant:
		return constantToStaticData(ty, n)
	case *InitializerList:
		return initializerListToStaticData(ty, n)
	default:
		panic("unimplemented")
	}
	return nil, cpp.ErrWithLoc(fmt.Errorf("unimplemented initializer"), n.GetPos())
}

func constantToStaticData(ty CType, c *Constant) (StaticData, error) {
	return nil, cpp.ErrWithLoc(fmt.Errorf("unimplemented initializer"), c.GetPos())
}

func initializerListToStaticData(ty CType, i *InitializerList) (StaticData, error) {
	if IsCharArr(ty) {
		if len(i.Members) == 1 {
			_, ok := i.Members[0].(*String)
			if !ok {
				e := fmt.Errorf("bad initializer for char array")
				return nil, cpp.ErrWithLoc(e, i.GetPos())
			}
		}
		// e.g. char x[] = {"foobar"};
		s := i.Members[0].(*String)
		return StringData{
			Val: s.Val,
		}, nil
	}
	return nil, cpp.ErrWithLoc(fmt.Errorf("unimplemented initializer"), i.GetPos())
}
