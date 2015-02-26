package parse

type CType interface {
	GetSize() int
	GetAlign() int
}

type PrimitiveKind int

const (
	Void PrimitiveKind = iota // type is invalid
	Bool
	Char
	Short
	Int
	Long
	LLong
	Float
	Double
	LDouble
	Enum
)

type Primitive struct {
	Kind     PrimitiveKind
	Size     int
	Align    int
	Unsigned bool
}

func (p *Primitive) GetSize() int  { return p.Size }
func (p *Primitive) GetAlign() int { return p.Align }

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

// Misc
var CVoid *Primitive = &Primitive{Void, 0, 0, false}
var CEnum *Primitive = &Primitive{Enum, 4, 4, false}

// Signed
var CChar *Primitive = &Primitive{Char, 1, 1, false}
var CShort *Primitive = &Primitive{Short, 2, 2, false}
var CInt *Primitive = &Primitive{Int, 4, 4, false}
var CLong *Primitive = &Primitive{Long, 8, 8, false}
var CLLong *Primitive = &Primitive{LLong, 8, 8, false}

// Unsigned
var CBool *Primitive = &Primitive{Bool, 1, 1, true}
var CUChar *Primitive = &Primitive{Char, 1, 1, true}
var CUShort *Primitive = &Primitive{Short, 2, 2, true}
var CUInt *Primitive = &Primitive{Int, 4, 4, true}
var CULong *Primitive = &Primitive{Long, 8, 8, true}
var CULLong *Primitive = &Primitive{LLong, 8, 8, true}

// Floats
var CFloat *Primitive = &Primitive{Float, 4, 4, false}
var CDouble *Primitive = &Primitive{Double, 8, 8, false}
var CLDouble *Primitive = &Primitive{LDouble, 8, 8, false}

func IsPtrType(t CType) bool {
	_, ok := t.(*Ptr)
	return ok
}

func IsIntType(t CType) bool {
	prim, ok := t.(*Primitive)
	if !ok {
		return false
	}
	switch prim.Kind {
	case Bool, Short, Int, Long, LLong:
		return true
	default:
		return false
	}
}

func IsScalarType(t CType) bool {
	return IsPtrType(t) || IsIntType(t)
}
