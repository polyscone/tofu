package i18n

type Node interface {
	node()
}

type exprNode struct{}

func (exprNode) node() {}

type RootNode struct {
	exprNode
	Fragments []Node
}

type IdentNode struct {
	exprNode
	Name Token
}

type LiteralNode struct {
	exprNode
	Token
}

type UnaryNode struct {
	exprNode
	Op  Token
	RHS Node
}

type BinaryNode struct {
	exprNode
	Op  Token
	LHS Node
	RHS Node
}

type IndexNode struct {
	exprNode
	LHS  Node
	Expr Node
}

type SliceNode struct {
	exprNode
	LHS   Node
	Start Node
	End   Node
}

type OptNode struct {
	exprNode
	Match  Node
	Result Node
}

type SelectNode struct {
	exprNode
	Value Node
	Opts  []*OptNode
}

type CallNode struct {
	exprNode
	LHS  Node
	Args []Node
}

type InvalidNode struct {
	exprNode
	Start Token
	End   Token
}
