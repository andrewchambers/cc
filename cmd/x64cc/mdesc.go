package main

import (
	"github.com/andrewchambers/cc/parse"
)

var primSizeTab = [...]int{
	parse.CVoid:   0,
	parse.CChar:   1,
	parse.CUChar:  1,
	parse.CShort:  2,
	parse.CUShort: 2,
	parse.CInt:    4,
	parse.CUInt:   4,
	parse.CLong:   8,
	parse.CULong:  8,
	parse.CLLong:  8,
	parse.CULLong: 8,
}

var primAlignTab = [...]int{
	parse.CVoid:   0,
	parse.CBool:   1,
	parse.CChar:   1,
	parse.CUChar:  1,
	parse.CShort:  2,
	parse.CUShort: 2,
	parse.CInt:    4,
	parse.CUInt:   4,
	parse.CLong:   8,
	parse.CULong:  8,
	parse.CLLong:  8,
	parse.CULLong: 8,
}

func getSize(t parse.CType) int {
	switch t := t.(type) {
	case *parse.Array:
		return t.Dim * getSize(t.MemberType)
	case *parse.Ptr:
		return 8
	case parse.Primitive:
		return primSizeTab[t]
	}
	panic(t)
}

func getAlign(t parse.CType) int {
	switch t := t.(type) {
	case *parse.Array:
		return 8
	case *parse.Ptr:
		return 8
	case parse.Primitive:
		return primAlignTab[t]
	}
	panic(t)
}
