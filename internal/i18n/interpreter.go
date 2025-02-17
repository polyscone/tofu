package i18n

import "fmt"

func Eval(node Node, rt Runtime, locale string, vars Vars) (Value, error) {
	switch node := node.(type) {
	case nil:
		return stringEmpty, nil

	case *RootNode:
		var values Slice
		for _, fragment := range node.Fragments {
			value, err := Eval(fragment, rt, locale, vars)
			if err != nil {
				return stringError, err
			}

			values = append(values, value)
		}

		return values, nil

	case *IdentNode:
		name := node.Name.Lexeme
		if fname, ok := funcNames[name]; ok {
			return fname, nil
		}

		if value, ok := vars[name]; ok {
			return value, nil
		}

		return stringEmpty, nil

	case *LiteralNode:
		str := NewString(node.Lexeme)

		switch node.Kind {
		case KindInt:
			return str.AsInt(), nil

		case KindFloat:
			return str.AsFloat(), nil

		case KindString, KindText:
			return str, nil

		default:
			return str, nil
		}

	case *UnaryNode:
		rhs, err := Eval(node.RHS, rt, locale, vars)
		if err != nil {
			return stringError, err
		}

		switch node.Op.Kind {
		case KindBang:
			return NewBool(!rhs.AsBool().Value), nil

		case KindPlus:
			switch rhs.Type() {
			case TypeBool, TypeString, TypeSlice:
				return rhs.AsInt(), nil

			default:
				return rhs, nil
			}

		case KindMinus:
			switch rhs.Type() {
			case TypeBool, TypeInt, TypeString, TypeSlice:
				return NewInt(-rhs.AsInt().Value), nil

			case TypeFloat:
				return NewFloat(-rhs.AsFloat().Value), nil

			default:
				return rhs, nil
			}

		default:
			return stringEmpty, nil
		}

	case *BinaryNode:
		lhs, err := Eval(node.LHS, rt, locale, vars)
		if err != nil {
			return stringError, err
		}

		rhs, err := Eval(node.RHS, rt, locale, vars)
		if err != nil {
			return stringError, err
		}

		switch node.Op.Kind {
		case KindOr:
			return NewBool(lhs.AsBool().Value || rhs.AsBool().Value), nil

		case KindAnd:
			return NewBool(lhs.AsBool().Value && rhs.AsBool().Value), nil

		case KindEqualEqual:
			return lhs.Equal(rhs), nil

		case KindBangEqual:
			return NewBool(!lhs.Equal(rhs).Value), nil

		case KindGreater:
			return NewBool(!lhs.Less(rhs).Value && !lhs.Equal(rhs).Value), nil

		case KindGreaterEqual:
			return NewBool(!lhs.Less(rhs).Value || lhs.Equal(rhs).Value), nil

		case KindLess:
			return lhs.Less(rhs), nil

		case KindLessEqual:
			return NewBool(lhs.Less(rhs).Value || lhs.Equal(rhs).Value), nil

		case KindPlus:
			return lhs.Add(rhs), nil

		case KindMinus:
			return lhs.Sub(rhs), nil

		case KindAsterisk:
			return lhs.Mul(rhs), nil

		case KindSlash:
			return lhs.Div(rhs), nil

		case KindPercent:
			return lhs.Mod(rhs), nil

		default:
			return stringError, fmt.Errorf("unexpected %q", node.Op.Kind)
		}

	case *IndexNode:
		lhs, err := Eval(node.LHS, rt, locale, vars)
		if err != nil {
			return stringError, err
		}

		expr, err := Eval(node.Expr, rt, locale, vars)
		if err != nil {
			return stringError, err
		}

		index := int(expr.AsInt().Value)
		slice := lhs.AsSlice()
		if index >= len(slice) {
			return stringEmpty, nil
		}
		if index < 0 {
			index = len(slice) + index

			if index < 0 {
				return stringEmpty, nil
			}
		}

		return slice[index], nil

	case *SliceNode:
		lhs, err := Eval(node.LHS, rt, locale, vars)
		if err != nil {
			return stringError, err
		}

		if _type := lhs.Type(); _type != TypeString && _type != TypeSlice {
			return stringError, fmt.Errorf("cannot slice %v", _type)
		}

		var start Value
		if node.Start != nil {
			value, err := Eval(node.Start, rt, locale, vars)
			if err != nil {
				return stringError, err
			}

			if _type := value.Type(); _type != TypeInt {
				return stringError, fmt.Errorf("cannot use %v as the start slice index", _type)
			}

			start = value
		} else {
			start = intZero
		}

		slice := lhs.AsSlice()

		var end Value
		if node.End != nil {
			value, err := Eval(node.End, rt, locale, vars)
			if err != nil {
				return stringError, err
			}

			if _type := value.Type(); _type != TypeInt {
				return stringError, fmt.Errorf("cannot use %v as the end slice index", _type)
			}

			end = value
		} else {
			end = NewInt(int64(len(slice)))
		}

		i := int(start.AsInt().Value)
		j := int(end.AsInt().Value)
		if j < 0 {
			j = len(slice) + j
		}
		if i < 0 || j < i || j > len(slice) {
			return stringEmpty, nil
		}

		return NewSlice(slice[i:j]), nil

	case *SelectNode:
		value, err := Eval(node.Value, rt, locale, vars)
		if err != nil {
			return stringError, err
		}

		for _, opt := range node.Opts {
			var isMatch bool
			if ident, ok := opt.Match.(*IdentNode); ok {
				isMatch = ident.Name.Lexeme == "_"
			}
			if !isMatch {
				match, err := Eval(opt.Match, rt, locale, vars)
				if err != nil {
					return stringError, err
				}

				isMatch = value.Equal(match).Value
			}

			if isMatch {
				result, err := Eval(opt.Result, rt, locale, vars)
				if err != nil {
					return stringError, err
				}

				return result, nil
			}
		}

		return stringEmpty, nil

	case *CallNode:
		lhs, err := Eval(node.LHS, rt, locale, vars)
		if err != nil {
			return stringError, err
		}

		switch lhs.AsString().Value {
		case "len":
			arg0 := arg(node.Args, 0, rt, locale, vars)
			res := rt.Len(arg0)

			return res, nil

		case "join":
			arg0 := arg(node.Args, 0, rt, locale, vars)
			arg1 := arg(node.Args, 1, rt, locale, vars)
			res := rt.Join(arg0, arg1)

			return res, nil

		case "split":
			arg0 := arg(node.Args, 0, rt, locale, vars)
			arg1 := arg(node.Args, 1, rt, locale, vars)
			res := rt.Split(arg0, arg1)

			return res, nil

		case "bold":
			arg0 := arg(node.Args, 0, rt, locale, vars)
			res := rt.Bold(arg0)

			return res, nil

		case "italic":
			arg0 := arg(node.Args, 0, rt, locale, vars)
			res := rt.Italic(arg0)

			return res, nil

		case "link":
			arg0 := arg(node.Args, 0, rt, locale, vars)
			arg1 := arg(node.Args, 1, rt, locale, vars)
			arg2 := arg(node.Args, 2, rt, locale, vars)
			res := rt.Link(arg0, arg1, arg2)

			return res, nil

		case "pad_left":
			arg0 := arg(node.Args, 0, rt, locale, vars)
			arg1 := arg(node.Args, 1, rt, locale, vars)
			arg2 := arg(node.Args, 2, rt, locale, vars)
			res := rt.PadLeft(arg0, arg1, arg2)

			return res, nil

		case "pad_right":
			arg0 := arg(node.Args, 0, rt, locale, vars)
			arg1 := arg(node.Args, 1, rt, locale, vars)
			arg2 := arg(node.Args, 2, rt, locale, vars)
			res := rt.PadRight(arg0, arg1, arg2)

			return res, nil

		case "trim_left":
			arg0 := arg(node.Args, 0, rt, locale, vars)
			arg1 := arg(node.Args, 1, rt, locale, vars)
			res := rt.TrimLeft(arg0, arg1)

			return res, nil

		case "trim_right":
			arg0 := arg(node.Args, 0, rt, locale, vars)
			arg1 := arg(node.Args, 1, rt, locale, vars)
			res := rt.TrimRight(arg0, arg1)

			return res, nil

		case "integer":
			arg0 := arg(node.Args, 0, rt, locale, vars)
			res := rt.Integer(arg0)

			return res, nil

		case "fraction":
			arg0 := arg(node.Args, 0, rt, locale, vars)
			arg1 := arg(node.Args, 1, rt, locale, vars)
			res := rt.Fraction(arg0, arg1)

			return res, nil

		case "t":
			arg0 := arg(node.Args, 0, rt, locale, vars)
			arg1 := arg(node.Args, 1, rt, locale, vars)
			arg2 := arg(node.Args, 2, rt, locale, vars)
			res := rt.T(arg0, locale, arg1, arg2)

			return res, nil
		}

		return stringEmpty, nil

	default:
		return stringError, fmt.Errorf("unhandled node: %#v", node)
	}
}

func arg(nodes []Node, i int, rt Runtime, locale string, vars Vars) Value {
	if i < 0 || i >= len(nodes) {
		return stringEmpty
	}

	value, _ := Eval(nodes[i], rt, locale, vars)

	return value
}
