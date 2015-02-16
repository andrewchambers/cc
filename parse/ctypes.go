package parse

type CType interface {
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

type Array struct {
	MemberType CType
	Dim        int
}

type Ptr struct {
	PointsTo CType
}

// Struct or union.
type Struct struct {
	Fields []struct {
		Name string
		Type CType
	}
	IsUnion bool
}

type FunctionType struct {
	RetType  CType
	ArgTypes []CType
	ArgNames []string
	IsVarArg bool
}

// All the primitive C types.

// Misc
var CVoid *Primitive = &Primitive{Void, 0, 0, false}
var CEnum *Primitive = &Primitive{Enum, 4, 4, false}

// Signed
var CChar *Primitive = &Primitive{Char, 0, 0, false}
var CShort *Primitive = &Primitive{Short, 2, 2, false}
var CInt *Primitive = &Primitive{Int, 4, 4, false}
var CLong *Primitive = &Primitive{Long, 8, 8, false}
var CLLong *Primitive = &Primitive{LLong, 8, 8, false}

// Unsigned
var CBool *Primitive = &Primitive{Void, 1, 1, true}
var CUChar *Primitive = &Primitive{Void, 1, 1, true}
var CUShort *Primitive = &Primitive{Void, 2, 2, true}
var CUInt *Primitive = &Primitive{Void, 4, 4, true}
var CULong *Primitive = &Primitive{Void, 8, 8, true}
var CULLong *Primitive = &Primitive{Void, 8, 8, true}

// Floats
var CFloat *Primitive = &Primitive{Void, 4, 4, false}
var CDouble *Primitive = &Primitive{Void, 8, 8, false}
var CLDouble *Primitive = &Primitive{Void, 8, 8, false}
