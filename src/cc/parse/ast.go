package parse

type ASTNode interface {
}

type ASTFunction struct {
}

type ASTBinop struct {
	op          int
	left, right ASTNode
}

type ASTTernary struct {
}

type ASTReturn struct {
}

type ASTIf struct {
}

type ASTConstant struct {
}

type ASTStringLiteral struct {
}
