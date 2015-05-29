package parse

// Anything that can be used to staticially initialize a global variable is
// represented by the Data
type StaticData interface{}

type StaticPtrDerived struct {
	Sz       int
	PtrLabel string
	Sym      string
	Val      int64
}

type StaticConstant struct {
	Label string
	Sz    int
	Val   int64
}

type StaticZero struct {
	Label string
	Sz    int
}

type StaticString struct {
	Label string
	// May be longer than Val.
	// Should never be less.
	Len int
	Val string
}

type StaticArray struct {
	Label   string
	Offsets []int
	Vals    []StaticData
}

// Converts an initializer node static data.
//
// Global and static variables must check that the len of the dynamic inits
// is 0.
func (p *parser) nodeToStatic(ty CType, n Node) []StaticData {
	switch n := n.(type) {
	case *Constant:
		return []StaticData{p.constantToStatic(ty, n)}
	case *InitializerList:
		return p.initializerListToStatic(ty, n)
	}
	panic("unimplemented")
}

func (p *parser) constantToStatic(ty CType, c *Constant) StaticData {
	return &StaticConstant{
		Sz:  8, // XXX get type size.
		Val: c.Val,
	}
}

func (p *parser) initializerListToStatic(ty CType, i *InitializerList) []StaticData {
	if IsCharArr(ty) {
		if len(i.Members) == 1 {
			_, ok := i.Members[0].(*String)
			if !ok {
				p.errorPos(i.GetPos(), "bad initializer for char array")
			}
		}
		// e.g. char x[] = {"foobar"};
		s := i.Members[0].(*String)
		return []StaticData{
			StaticString{
				Val: s.Val,
			}}
	}
	if IsCharPtr(ty) {
		if len(i.Members) == 1 {
			_, ok := i.Members[0].(*String)
			if !ok {
				p.errorPos(i.GetPos(), "bad initializer for char pointer")
			}
		}
		// e.g. char *p = {"foobar"};
		s := i.Members[0].(*String)
		return []StaticData{
			StaticString{
				Val: s.Val,
			},
			StaticPtrDerived{
				PtrLabel: "XXX TODO",
			},
		}
	}
	panic("unimplemented")
}

func (p *parser) nodeToLocalInits(s LSymbol, n Node) ([]Node, []StaticData, error) {
	panic("unimplemented")
}
