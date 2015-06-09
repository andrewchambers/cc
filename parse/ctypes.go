package parse

type TargetSizeDesc struct {
	GetSize  func(CType) int
	GetAlign func(CType) int
}

type CType interface{}

type Array struct {
	MemberType CType
	Dim        int
}

type Ptr struct {
	PointsTo CType
}

// Struct or union.
type CStruct struct {
	Names   []string
	Types   []CType
	IsUnion bool
}

func (s *CStruct) FieldType(n string) CType {
	for idx, v := range s.Names {
		if v == n {
			return s.Types[idx]
		}
	}
	return nil
}

type FunctionType struct {
	RetType  CType
	ArgTypes []CType
	ArgNames []string
	IsVarArg bool
}

type ForwardedType struct {
	Type CType
}

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

func IsSignedIntType(t CType) bool {
	prim, ok := t.(Primitive)
	if !ok {
		return false
	}
	return prim >= CEnum && prim <= CLLong
}

func IsScalarType(t CType) bool {
	return IsPtrType(t) || IsIntType(t)
}

func IsArrType(t CType) bool {
	_, ok := t.(*Array)
	return ok
}

func IsCharType(t CType) bool {
	prim, ok := t.(Primitive)
	if !ok {
		return false
	}
	return prim == CChar
}

func IsCharArr(t CType) bool {
	return IsArrType(t) && IsCharType(t)
}

func IsStruct(t CType) bool {
	_, ok := t.(*CStruct)
	return ok
}

func IsCFunction(t CType) bool {
	_, ok := t.(*FunctionType)
	return ok
}
