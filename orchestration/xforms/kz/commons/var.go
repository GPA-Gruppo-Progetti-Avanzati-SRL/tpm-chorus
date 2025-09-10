package commons

import "github.com/qntfy/jsonparser"

var NullVar = ExpressionVariable{Dt: jsonparser.Null}
var EmptyArrayVar = ExpressionVariable{Dt: jsonparser.Array, Val: []byte(`[]`)}
var EmptyObjectVar = ExpressionVariable{Dt: jsonparser.Array, Val: []byte(`{}`)}

type ExpressionVariable struct {
	Val []byte
	Dt  jsonparser.ValueType
}

func NewExpressionVariable(val []byte, dt jsonparser.ValueType) ExpressionVariable {
	return ExpressionVariable{Val: val, Dt: dt}
}
