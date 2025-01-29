package amt_test

import (
	"fmt"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/funcs/purefuncs/amt"
	"github.com/stretchr/testify/require"
	"testing"
)

type amountFormatInputWanted struct {
	amt        string
	sourceUnit string
	targetUnit string
	wanted     string
	negate     bool
}

func TestFormat(t *testing.T) {
	arr := []amountFormatInputWanted{
		{sourceUnit: amt.MicroCent, targetUnit: amt.Cent, amt: "10000", wanted: "1"},
		{sourceUnit: amt.Mill, targetUnit: amt.Mill, amt: "10", wanted: "10"},
		{sourceUnit: amt.Mill, targetUnit: amt.Cent, amt: "100", wanted: "10"},
		{sourceUnit: amt.Cent, targetUnit: amt.MicroCent, amt: "1", wanted: "10000"},
		{sourceUnit: amt.Cent, targetUnit: amt.Mill, amt: "1", wanted: "10"},
		{sourceUnit: amt.Cent, targetUnit: amt.Cent, amt: "1", wanted: "1"},
		{sourceUnit: amt.Cent, targetUnit: amt.Cent, amt: "00000", wanted: "0"},
		{sourceUnit: amt.Cent, targetUnit: amt.Decimal, amt: "1", wanted: "0.01"},
		{sourceUnit: amt.Decimal, targetUnit: amt.Cent, amt: "1500", wanted: "150000"},
		{sourceUnit: amt.Decimal, targetUnit: amt.Cent, amt: "1500.0000000000", wanted: "150000"},
		{sourceUnit: amt.Decimal, targetUnit: amt.Cent, amt: "1500,00", wanted: "-150000", negate: true},
		{sourceUnit: amt.Decimal, targetUnit: amt.Cent, amt: ",00", wanted: "0", negate: true},
	}

	for i, test := range arr {
		w, err := amt.Format(test.amt, test.sourceUnit, test.targetUnit, test.negate)
		require.NoError(t, err, "[%d] %s", i, test.amt)
		require.Equal(t, test.wanted, w, "[%d] %s", i, test.amt)
	}
}

type amountConversionInputWanted struct {
	OpType        string
	targetUnit    string
	sourceUnit    string
	decimalFormat bool
	amts          []interface{}
	wanted        string
}

func TestAddAndDiff(t *testing.T) {
	addDiffArr := []amountConversionInputWanted{
		{OpType: amt.AmountOpAdd, targetUnit: amt.Mill, sourceUnit: amt.Cent, decimalFormat: false, amts: []interface{}{"15000", "100"}, wanted: "151000"},
		{OpType: amt.AmountOpAdd, targetUnit: amt.Cent, sourceUnit: amt.Cent, decimalFormat: true, amts: []interface{}{"15000", "100"}, wanted: "151.00"},
		{OpType: amt.AmountOpDiff, targetUnit: amt.Mill, sourceUnit: amt.Cent, decimalFormat: false, amts: []interface{}{"1234", "100"}, wanted: "11340"},
		{OpType: amt.AmountOpDiff, targetUnit: amt.Mill, sourceUnit: amt.Cent, decimalFormat: false, amts: []interface{}{"100", "1234"}, wanted: "-11340"},
		{OpType: amt.AmountOpAdd, targetUnit: amt.Cent, sourceUnit: amt.DecimalCent, decimalFormat: false, amts: []interface{}{"1", "1"}, wanted: "200"},
	}

	for i, input := range addDiffArr {
		news, err := amt.Amt(input.OpType, input.sourceUnit, input.targetUnit, input.decimalFormat, input.amts...)
		require.NoError(t, err)
		require.Equal(t, input.wanted, news, fmt.Sprintf("error on funcs.AmtAdd [%d]", i))
	}

	convArr := []amountConversionInputWanted{
		{OpType: amt.AmountOpAdd, targetUnit: amt.Cent, sourceUnit: amt.MicroCent, decimalFormat: false, amts: []interface{}{"000000000001000000"}, wanted: "100"},
		{OpType: amt.AmountOpAdd, targetUnit: amt.MicroCent, sourceUnit: amt.Cent, decimalFormat: false, amts: []interface{}{"100"}, wanted: "1000000"},
		{OpType: amt.AmountOpAdd, targetUnit: amt.Cent, sourceUnit: amt.Cent, decimalFormat: true, amts: []interface{}{"999"}, wanted: "9.99"},
		{OpType: amt.AmountOpAdd, targetUnit: amt.Cent, sourceUnit: amt.Cent, decimalFormat: true, amts: []interface{}{"1"}, wanted: "0.01"},
		{OpType: amt.AmountOpAdd, targetUnit: amt.Mill, sourceUnit: amt.Cent, decimalFormat: true, amts: []interface{}{"123"}, wanted: "12.30"},
		{OpType: amt.AmountOpAdd, targetUnit: amt.Cent, sourceUnit: amt.DecimalCent, decimalFormat: false, amts: []interface{}{"12.34"}, wanted: "1234"},
		{OpType: amt.AmountOpAdd, targetUnit: amt.Cent, sourceUnit: amt.DecimalCent, decimalFormat: false, amts: []interface{}{"12,34"}, wanted: "1234"},
		{OpType: amt.AmountOpAdd, targetUnit: amt.Cent, sourceUnit: amt.DecimalCent, decimalFormat: false, amts: []interface{}{"0,1"}, wanted: "10"},
		{OpType: amt.AmountOpAdd, targetUnit: amt.Cent, sourceUnit: amt.Cent, decimalFormat: false, amts: []interface{}{"9.000000"}, wanted: "9"},
		{OpType: amt.AmountOpAdd, targetUnit: amt.Mill, sourceUnit: amt.DecimalMillis, decimalFormat: false, amts: []interface{}{"9.001000"}, wanted: "9001"},
		{OpType: amt.AmountOpAdd, targetUnit: amt.Cent, sourceUnit: amt.DecimalCent, decimalFormat: false, amts: []interface{}{"1"}, wanted: "100"},
		{OpType: amt.AmountOpAdd, targetUnit: amt.Cent, sourceUnit: amt.Mill, decimalFormat: true, amts: []interface{}{"355120"}, wanted: "355.12"},
		{OpType: amt.AmountOpAdd, targetUnit: amt.Cent, sourceUnit: amt.DecimalCent, decimalFormat: false, amts: []interface{}{"2400,45"}, wanted: "240045"},
	}

	for i, input := range convArr {
		news, err := amt.AmtConv(input.sourceUnit, input.targetUnit, input.decimalFormat, input.amts[0])
		require.NoError(t, err)
		require.Equal(t, input.wanted, news, fmt.Sprintf("error on funcs.AmtConv [%d]", i))
	}
}

type amountCompareInputWanted struct {
	cmpUnit     string
	amt1SrcUnit string
	amt1        string
	amt2SrcUnit string
	amt2        string
	wanted      bool
}

func TestAmtCompare(t *testing.T) {
	cmpArr := []amountCompareInputWanted{
		{
			cmpUnit:     amt.Cent,
			amt1SrcUnit: amt.Cent,
			amt1:        "72",
			amt2SrcUnit: amt.Cent,
			amt2:        "123",
			wanted:      false,
		}}

	for _, input := range cmpArr {
		r, err := amt.AmtCmp(input.cmpUnit, input.amt1, input.amt1SrcUnit, input.amt2, input.amt2SrcUnit)
		require.NoError(t, err)
		require.Equal(t, input.wanted, r)
	}
}
