package i18n

import (
	"fmt"
	"strings"
)

func SprintNode(node Node) string {
	switch node := node.(type) {
	case *RootNode:
		strs := make([]string, len(node.Fragments))
		for i, fragment := range node.Fragments {
			strs[i] = SprintNode(fragment)
		}

		return strings.Join(strs, " ")

	case *IdentNode:
		return node.Name.Lexeme

	case *LiteralNode:
		if node.Kind == KindString || node.Kind == KindText {
			return fmt.Sprintf("%q", node.Lexeme)
		}

		return node.Lexeme

	case *UnaryNode:
		return fmt.Sprintf("(%v %v)", node.Op.Lexeme, SprintNode(node.RHS))

	case *BinaryNode:
		return fmt.Sprintf("(%v %v %v)", node.Op.Lexeme, SprintNode(node.LHS), SprintNode(node.RHS))

	case *IndexNode:
		return fmt.Sprintf("(index %v %v)", SprintNode(node.LHS), SprintNode(node.Expr))

	case *SliceNode:
		if node.End == nil {
			return fmt.Sprintf("(slice %v %v)", SprintNode(node.LHS), SprintNode(node.Start))
		}

		return fmt.Sprintf("(slice %v %v %v)", SprintNode(node.LHS), SprintNode(node.Start), SprintNode(node.End))

	case *OptNode:
		return fmt.Sprintf("(opt %v %v)", SprintNode(node.Match), SprintNode(node.Result))

	case *SelectNode:
		if len(node.Opts) > 0 {
			strs := make([]string, len(node.Opts))
			for i, arg := range node.Opts {
				strs[i] = SprintNode(arg)
			}

			return fmt.Sprintf("(select %v %v)", SprintNode(node.Value), strings.Join(strs, " "))
		}

		return fmt.Sprintf("(select %v)", SprintNode(node.Value))

	case *CallNode:
		if len(node.Args) > 0 {
			strs := make([]string, len(node.Args))
			for i, arg := range node.Args {
				strs[i] = SprintNode(arg)
			}

			return fmt.Sprintf("(call %v %v)", SprintNode(node.LHS), strings.Join(strs, " "))
		}

		return fmt.Sprintf("(call %v)", SprintNode(node.LHS))

	default:
		return fmt.Sprintf("(? %T)", node)
	}
}
