package parse

type CType interface {
	GetSize() int
	GetAlign() int
}

type Array struct {
	MemberType CType
	Dim        int
}

func (a *Array) GetSize() int  { return a.MemberType.GetSize() * a.Dim }
func (a *Array) GetAlign() int { return a.MemberType.GetAlign() }

type Ptr struct {
	PointsTo CType
}

func (p *Ptr) GetSize() int  { return 8 }
func (p *Ptr) GetAlign() int { return 8 }

// Struct or union.
type Struct struct {
	Fields []struct {
		Name string
		Type CType
	}
	IsUnion bool
}

func (s *Struct) GetSize() int  { return 8 }
func (s *Struct) GetAlign() int { return 8 }

type FunctionType struct {
	RetType  CType
	ArgTypes []CType
	ArgNames []string
	IsVarArg bool
}

func (f *FunctionType) GetSize() int  { panic("internal error") }
func (f *FunctionType) GetAlign() int { panic("internal error") }

type ForwardedType struct {
	Type CType
}

func (f *ForwardedType) GetSize() int  { return f.Type.GetSize() }
func (f *ForwardedType) GetAlign() int { return f.Type.GetAlign() }

// All the primitive C types.

type Primitive int

// *NOTE* order is significant.
const (
	CVoid Primitive = iota
	CEnum
	// Signed
	CChar
	CShort
	CInt
	CLong
	CLLong
	// Unsigned
	CBool
	CUChar
	CUShort
	CUInt
	CULong
	CULLong
	// Floats
	CFloat
	CDouble
	CLDouble
)

var primSizeTab = [...]int{
	CVoid:   0,
	CChar:   1,
	CUChar:  1,
	CShort:  2,
	CUShort: 2,
	CInt:    4,
	CUInt:   4,
	CLong:   8,
	CULong:  8,
	CLLong:  8,
	CULLong: 8,
}

var primAlignTab = [...]int{
	CVoid:   0,
	CBool:   1,
	CChar:   1,
	CUChar:  1,
	CShort:  2,
	CUShort: 2,
	CInt:    4,
	CUInt:   4,
	CLong:   8,
	CULong:  8,
	CLLong:  8,
	CULLong: 8,
}

func (p Primitive) GetAlign() int {
	return primAlignTab[p]
}

func (p Primitive) GetSize() int {
	return primSizeTab[p]
}

func IsPtrType(t CType) bool {
	_, ok := t.(*Ptr)
	return ok
}

func IsIntType(t CType) bool {
	prim, ok := t.(Primitive)
	if !ok {
		return false
	}
	return prim >= CEnum && prim < CFloat
}

func IsScalarType(t CType) bool {
	return IsPtrType(t) || IsIntType(t)
}
