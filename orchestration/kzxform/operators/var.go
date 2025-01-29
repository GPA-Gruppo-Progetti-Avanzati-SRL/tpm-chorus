package operators

import "github.com/qntfy/jsonparser"

type ExpressionVariable struct {
	Val []byte
	Dt  jsonparser.ValueType
}

func NewExpressionVariable(val []byte, dt jsonparser.ValueType) ExpressionVariable {
	return ExpressionVariable{Val: val, Dt: dt}
}
