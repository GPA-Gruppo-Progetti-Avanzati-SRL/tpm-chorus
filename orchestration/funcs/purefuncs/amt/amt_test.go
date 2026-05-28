package amt_test

import (
	"fmt"
	"math"
	"testing"

	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/funcs/purefuncs/amt"
	"github.com/stretchr/testify/require"
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

type amtFmtConvCase struct {
	value     interface{}
	srcFmt    string
	dstFmt    string
	decimals  int
	decSep    string
	wanted    string
	wantError bool
}

func TestAmtFmtConv(t *testing.T) {
	cases := []amtFmtConvCase{
		// ── intN → intN ──────────────────────────────────────────────────────
		{"710", amt.NumFmtInt2, amt.NumFmtInt3, 0, "", "7100", false},
		{"7100", amt.NumFmtInt3, amt.NumFmtInt2, 0, "", "710", false},
		{"7155", amt.NumFmtInt3, amt.NumFmtInt2, 0, "", "715", false},  // troncamento
		{"1", amt.NumFmtInt2, amt.NumFmtInt6, 0, "", "10000", false},
		{"10000", amt.NumFmtInt6, amt.NumFmtInt2, 0, "", "1", false},
		{"100", amt.NumFmtInt3, amt.NumFmtInt2, 0, "", "10", false},
		{"10", amt.NumFmtInt3, amt.NumFmtInt3, 0, "", "10", false},

		// ── decimal → intN ───────────────────────────────────────────────────
		{"7.10", amt.NumFmtDecimal, amt.NumFmtInt3, 0, "", "7100", false},
		{"7,10", amt.NumFmtDecimal, amt.NumFmtInt3, 0, "", "7100", false},  // virgola in input
		{"7", amt.NumFmtDecimal, amt.NumFmtInt6, 0, "", "7000000", false},
		{"1500", amt.NumFmtDecimal, amt.NumFmtInt2, 0, "", "150000", false},
		{"1500.00", amt.NumFmtDecimal, amt.NumFmtInt2, 0, "", "150000", false},
		{"-0.50", amt.NumFmtDecimal, amt.NumFmtInt2, 0, "", "-50", false},
		{"-7.10", amt.NumFmtDecimal, amt.NumFmtInt3, 0, "", "-7100", false},
		{"00000000015", amt.NumFmtDecimal, amt.NumFmtInt3, 0, "", "15000", false},  // leading zeros

		// ── intN → decimal (numero fisso di decimali) ─────────────────────────
		{"710", amt.NumFmtInt2, amt.NumFmtDecimal, 2, ".", "7.10", false},
		{"710", amt.NumFmtInt2, amt.NumFmtDecimal, 2, ",", "7,10", false},
		{"-150000", amt.NumFmtInt2, amt.NumFmtDecimal, 2, ".", "-1500.00", false},
		{"1", amt.NumFmtInt2, amt.NumFmtDecimal, 2, ".", "0.01", false},

		// ── decimal → decimal ────────────────────────────────────────────────
		{"7,10", amt.NumFmtDecimal, amt.NumFmtDecimal, 4, ".", "7.1000", false},
		{"7.10", amt.NumFmtDecimal, amt.NumFmtDecimal, 2, ".", "7.10", false},
		{"7.10", amt.NumFmtDecimal, amt.NumFmtDecimal, 2, ",", "7,10", false},
		{"7.1", amt.NumFmtDecimal, amt.NumFmtDecimal, 4, ".", "7.1000", false},
		{"0", amt.NumFmtDecimal, amt.NumFmtDecimal, 2, ".", "0.00", false},
		{"1500", amt.NumFmtDecimal, amt.NumFmtDecimal, 2, ".", "1500.00", false},
		{"00000000015", amt.NumFmtDecimal, amt.NumFmtDecimal, 0, ".", "15", false},  // leading zeros

		// ── tipi numerici nativi come input ───────────────────────────────────
		{710, amt.NumFmtInt2, amt.NumFmtInt3, 0, "", "7100", false},
		{int64(710), amt.NumFmtInt2, amt.NumFmtInt3, 0, "", "7100", false},
		{float64(7.10), amt.NumFmtDecimal, amt.NumFmtInt3, 0, "", "7100", false},
		{float64(-0.5), amt.NumFmtDecimal, amt.NumFmtInt2, 0, "", "-50", false},

		// ── decimals=-1 → precisione naturale senza zeri finali ───────────────
		{"12350", amt.NumFmtInt3, amt.NumFmtDecimal, -1, ",", "12,35", false},
		{"12300", amt.NumFmtInt3, amt.NumFmtDecimal, -1, ",", "12,3", false},
		{"12000", amt.NumFmtInt3, amt.NumFmtDecimal, -1, ",", "12", false},
		{"7,10", amt.NumFmtDecimal, amt.NumFmtDecimal, -1, ".", "7.1", false},
		{"1500.00", amt.NumFmtDecimal, amt.NumFmtDecimal, -1, ".", "1500", false},
		{"-12350", amt.NumFmtInt3, amt.NumFmtDecimal, -1, ".", "-12.35", false},
		{"500", amt.NumFmtInt3, amt.NumFmtDecimal, -1, ".", "0.5", false},
	}

	for i, c := range cases {
		got, err := amt.AmtFmtConv(c.value, c.srcFmt, c.dstFmt, c.decimals, c.decSep)
		require.NoError(t, err, "[%d] value=%v src=%s dst=%s", i, c.value, c.srcFmt, c.dstFmt)
		require.Equal(t, c.wanted, got, "[%d] value=%v src=%s dst=%s", i, c.value, c.srcFmt, c.dstFmt)
	}
}

func TestAmtFmtConvErrors(t *testing.T) {
	// formato sorgente sconosciuto
	_, err := amt.AmtFmtConv("10", "unknown", amt.NumFmtInt2, 0, "")
	require.Error(t, err)

	// formato destinazione sconosciuto
	_, err = amt.AmtFmtConv("10", amt.NumFmtInt2, "unknown", 0, "")
	require.Error(t, err)

	// valore non numerico
	_, err = amt.AmtFmtConv("abc", amt.NumFmtInt2, amt.NumFmtInt3, 0, "")
	require.Error(t, err)

	// separatore decimale in un formato integrale → errore
	_, err = amt.AmtFmtConv("7.10", amt.NumFmtInt2, amt.NumFmtInt3, 0, "")
	require.Error(t, err, "atteso errore per valore decimale passato come integrale")

	_, err = amt.AmtFmtConv("7,10", amt.NumFmtInt2, amt.NumFmtInt3, 0, "")
	require.Error(t, err, "atteso errore per valore decimale (virgola) passato come integrale")

	// separatore output non valido
	_, err = amt.AmtFmtConv("710", amt.NumFmtInt2, amt.NumFmtDecimal, 2, ";")
	require.Error(t, err, "atteso errore per decSep non valido")

	// parte decimale non numerica
	_, err = amt.AmtFmtConv("7.abc", amt.NumFmtDecimal, amt.NumFmtInt2, 0, "")
	require.Error(t, err, "atteso errore per parte decimale non numerica")

	// stringa vuota
	_, err = amt.AmtFmtConv("", amt.NumFmtInt2, amt.NumFmtInt3, 0, "")
	require.Error(t, err, "atteso errore per stringa vuota")

	// valori molto grandi: ok con implementazione string-based
	got, err := amt.AmtFmtConv("999999999999999999", amt.NumFmtInt2, amt.NumFmtInt3, 0, "")
	require.NoError(t, err)
	require.Equal(t, "9999999999999999990", got)

	got, err = amt.AmtFmtConv(float64(1e13), amt.NumFmtDecimal, amt.NumFmtInt2, 0, "")
	require.NoError(t, err)
	require.Equal(t, "1000000000000000", got)
}

func TestAmtFmtConvNegativeZero(t *testing.T) {
	cases := []struct {
		value  interface{}
		src    string
		dst    string
		dec    int
		sep    string
		wanted string
	}{
		{"-0", amt.NumFmtDecimal, amt.NumFmtInt2, 0, "", "0"},
		{"-0.00", amt.NumFmtDecimal, amt.NumFmtInt2, 0, "", "0"},
		{"-0.000", amt.NumFmtDecimal, amt.NumFmtDecimal, 2, ".", "0.00"},
		{"-000", amt.NumFmtInt3, amt.NumFmtDecimal, 2, ".", "0.00"},
		{"-0", amt.NumFmtInt2, amt.NumFmtInt3, 0, "", "0"},
		{int(-0), amt.NumFmtDecimal, amt.NumFmtDecimal, 2, ".", "0.00"},
		{int64(-0), amt.NumFmtDecimal, amt.NumFmtDecimal, 2, ".", "0.00"},
		{float64(-0.0), amt.NumFmtDecimal, amt.NumFmtDecimal, 2, ".", "0.00"},
	}
	for i, c := range cases {
		got, err := amt.AmtFmtConv(c.value, c.src, c.dst, c.dec, c.sep)
		require.NoError(t, err, "[%d]", i)
		require.Equal(t, c.wanted, got, "[%d] value=%v src=%s dst=%s", i, c.value, c.src, c.dst)
	}
}

func TestAmtFmtConvMinInt64(t *testing.T) {
	got, err := amt.AmtFmtConv(int64(math.MinInt64), amt.NumFmtDecimal, amt.NumFmtDecimal, 0, ".")
	require.NoError(t, err)
	require.Equal(t, "-9223372036854775808", got)

	got, err = amt.AmtFmtConv(int(math.MinInt64), amt.NumFmtDecimal, amt.NumFmtDecimal, 0, ".")
	require.NoError(t, err)
	require.Equal(t, "-9223372036854775808", got)
}

func TestAmtFmtConvRounding(t *testing.T) {
	// troncamento verso zero (non arrotondamento)
	got, err := amt.AmtFmtConv("7155", amt.NumFmtInt3, amt.NumFmtInt2, 0, "")
	require.NoError(t, err)
	require.Equal(t, "715", got, "7.155 troncato a int2 deve essere 715")

	got, err = amt.AmtFmtConv("7159", amt.NumFmtInt3, amt.NumFmtInt2, 0, "")
	require.NoError(t, err)
	require.Equal(t, "715", got)

	got, err = amt.AmtFmtConv("-7155", amt.NumFmtInt3, amt.NumFmtInt2, 0, "")
	require.NoError(t, err)
	require.Equal(t, "-715", got)
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
