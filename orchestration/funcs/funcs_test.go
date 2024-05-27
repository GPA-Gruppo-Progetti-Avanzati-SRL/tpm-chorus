package funcs_test

import (
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"
	"tpm-chorus/orchestration/funcs/simple"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/stretchr/testify/require"
)

type InputWanted struct {
	funcName string
	input    string
	wanted   string
}

type AmountConversionInputWanted struct {
	OpType        string
	targetUnit    string
	sourceUnit    string
	decimalFormat bool
	amts          []interface{}
	wanted        string
}

type DatesInputWanted struct {
	funcName     string
	input        interface{}
	value2       interface{}
	fmtLayout    string
	layouts      []string
	location     string
	wantedBool   bool
	wantedString string
	wantedInt    int
}

type AmountCompareInputWanted struct {
	cmpUnit     string
	amt1SrcUnit string
	amt1        string
	amt2SrcUnit string
	amt2        string
	wanted      bool
}

func TestFuncs(t *testing.T) {

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	f, err := strconv.ParseFloat("2.100", 32)
	require.NoError(t, err)
	t.Log(f)

	f, err = strconv.ParseFloat("2.100", 64)
	require.NoError(t, err)
	t.Log(f)

	sarr := []InputWanted{
		{funcName: "Printf", input: "PPYEVOBUS", wanted: "   PPYEVOBUS"},
		{funcName: "PadLeft", input: "PPYEVOBUS", wanted: "000PPYEVOBUS"},
		{funcName: "Left", input: "PPYEVOBUS", wanted: "PPYEV"},
		{funcName: "Right", input: "PPYEVOBUS", wanted: "VOBUS"},
	}

	var news string
	for _, s := range sarr {
		switch s.funcName {
		case "Printf":
			news = simple.Printf("%12s", s.input)
		case "PadLeft":
			news = simple.PadLeft(s.input, 12, "0")
		case "Left":
			news = simple.Left(s.input, 5)
		case "Right":
			news = simple.Right(s.input, 5)
		default:
			t.Fatalf("func %s not present", s.funcName)
		}

		require.Equal(t, s.wanted, news)
	}

	addDiffArr := []AmountConversionInputWanted{
		{OpType: simple.AmountOpAdd, targetUnit: simple.Mill, sourceUnit: simple.Cent, decimalFormat: false, amts: []interface{}{"15000", "100"}, wanted: "151000"},
		{OpType: simple.AmountOpAdd, targetUnit: simple.Cent, sourceUnit: simple.Cent, decimalFormat: true, amts: []interface{}{"15000", "100"}, wanted: "151.00"},
		{OpType: simple.AmountOpDiff, targetUnit: simple.Mill, sourceUnit: simple.Cent, decimalFormat: false, amts: []interface{}{"1234", "100"}, wanted: "11340"},
		{OpType: simple.AmountOpDiff, targetUnit: simple.Mill, sourceUnit: simple.Cent, decimalFormat: false, amts: []interface{}{"100", "1234"}, wanted: "-11340"},
		{OpType: simple.AmountOpAdd, targetUnit: simple.Cent, sourceUnit: simple.DecimalCent, decimalFormat: false, amts: []interface{}{"1", "1"}, wanted: "200"},
	}

	for i, input := range addDiffArr {
		news, err := simple.Amt(input.OpType, input.sourceUnit, input.targetUnit, input.decimalFormat, input.amts...)
		require.NoError(t, err)
		require.Equal(t, input.wanted, news, fmt.Sprintf("error on funcs.AmtAdd [%d]", i))
	}

	convArr := []AmountConversionInputWanted{
		{OpType: simple.AmountOpAdd, targetUnit: simple.Cent, sourceUnit: simple.MicroCent, decimalFormat: false, amts: []interface{}{"000000000001000000"}, wanted: "100"},
		{OpType: simple.AmountOpAdd, targetUnit: simple.MicroCent, sourceUnit: simple.Cent, decimalFormat: false, amts: []interface{}{"100"}, wanted: "1000000"},
		{OpType: simple.AmountOpAdd, targetUnit: simple.Cent, sourceUnit: simple.Cent, decimalFormat: true, amts: []interface{}{"999"}, wanted: "9.99"},
		{OpType: simple.AmountOpAdd, targetUnit: simple.Cent, sourceUnit: simple.Cent, decimalFormat: true, amts: []interface{}{"1"}, wanted: "0.01"},
		{OpType: simple.AmountOpAdd, targetUnit: simple.Mill, sourceUnit: simple.Cent, decimalFormat: true, amts: []interface{}{"123"}, wanted: "12.30"},
		{OpType: simple.AmountOpAdd, targetUnit: simple.Cent, sourceUnit: simple.DecimalCent, decimalFormat: false, amts: []interface{}{"12.34"}, wanted: "1234"},
		{OpType: simple.AmountOpAdd, targetUnit: simple.Cent, sourceUnit: simple.DecimalCent, decimalFormat: false, amts: []interface{}{"12,34"}, wanted: "1234"},
		{OpType: simple.AmountOpAdd, targetUnit: simple.Cent, sourceUnit: simple.DecimalCent, decimalFormat: false, amts: []interface{}{"0,1"}, wanted: "10"},
		{OpType: simple.AmountOpAdd, targetUnit: simple.Cent, sourceUnit: simple.Cent, decimalFormat: false, amts: []interface{}{"9.000000"}, wanted: "9"},
		{OpType: simple.AmountOpAdd, targetUnit: simple.Mill, sourceUnit: simple.DecimalMillis, decimalFormat: false, amts: []interface{}{"9.001000"}, wanted: "9001"},
		{OpType: simple.AmountOpAdd, targetUnit: simple.Cent, sourceUnit: simple.DecimalCent, decimalFormat: false, amts: []interface{}{"1"}, wanted: "100"},
		{OpType: simple.AmountOpAdd, targetUnit: simple.Cent, sourceUnit: simple.Mill, decimalFormat: true, amts: []interface{}{"355120"}, wanted: "355.12"},
		{OpType: simple.AmountOpAdd, targetUnit: simple.Cent, sourceUnit: simple.DecimalCent, decimalFormat: false, amts: []interface{}{"2400,45"}, wanted: "240045"},
	}

	/*
		news, err = simple.AmtConv(convArr[12].sourceUnit, convArr[12].targetUnit, convArr[12].decimalFormat, convArr[12].amts[0])
		require.NoError(t, err)
		require.Equal(t, convArr[12].wanted, news, fmt.Sprintf("error on funcs.Amt [%d]", 12))
	*/

	for i, input := range convArr {
		news, err := simple.AmtConv(input.sourceUnit, input.targetUnit, input.decimalFormat, input.amts[0])
		require.NoError(t, err)
		require.Equal(t, input.wanted, news, fmt.Sprintf("error on funcs.AmtConv [%d]", i))
	}

	cmpArr := []AmountCompareInputWanted{
		{
			cmpUnit:     simple.Cent,
			amt1SrcUnit: simple.Cent,
			amt1:        "72",
			amt2SrcUnit: simple.Cent,
			amt2:        "123",
			wanted:      false,
		}}

	for _, input := range cmpArr {
		r, err := simple.AmtCmp(input.cmpUnit, input.amt1, input.amt1SrcUnit, input.amt2, input.amt2SrcUnit)
		require.NoError(t, err)
		require.Equal(t, input.wanted, r)
	}

	now := time.Now()
	const Dashed20230301 = "2023-03-01"
	const DashedDateLayout = "2006-01-02"
	const YYYYMMDDdDateLayout = "20060102"
	const YYYYMMDDMarchFirst = "20230301"
	const YYYYMMDDMarchThird = "20230303"
	datesArr := []DatesInputWanted{
		{
			funcName:   "IsDate",
			input:      now,
			layouts:    nil,
			wantedBool: true,
		},
		{
			funcName:   "IsDate",
			input:      YYYYMMDDMarchThird,
			layouts:    nil,
			wantedBool: false,
		},
		{
			funcName:   "IsDate",
			input:      YYYYMMDDMarchFirst,
			layouts:    []string{YYYYMMDDdDateLayout},
			wantedBool: true,
		},
		{
			funcName:   "IsDate",
			input:      Dashed20230301,
			layouts:    []string{YYYYMMDDdDateLayout},
			wantedBool: false,
		},
		{
			funcName:   "IsDate",
			input:      Dashed20230301,
			layouts:    []string{YYYYMMDDdDateLayout, DashedDateLayout},
			wantedBool: true,
		},

		{
			funcName:   "ParseDate",
			input:      now,
			layouts:    nil,
			wantedBool: true,
		},
		{
			funcName:   "ParseDate",
			input:      YYYYMMDDMarchThird,
			layouts:    nil,
			wantedBool: false,
		},
		{
			funcName:   "ParseDate",
			input:      YYYYMMDDMarchFirst,
			layouts:    []string{YYYYMMDDdDateLayout},
			wantedBool: true,
		},
		{
			funcName:   "ParseDate",
			input:      Dashed20230301,
			layouts:    []string{YYYYMMDDdDateLayout},
			wantedBool: false,
		},
		{
			funcName:   "ParseDate",
			input:      Dashed20230301,
			layouts:    []string{YYYYMMDDdDateLayout, DashedDateLayout},
			wantedBool: true,
		},

		{
			funcName:     "ParseAndFmtDate",
			input:        now,
			fmtLayout:    "",
			layouts:      nil,
			wantedString: "",
		},
		{
			funcName:     "ParseAndFmtDate",
			input:        now,
			fmtLayout:    YYYYMMDDdDateLayout,
			location:     "UTC",
			layouts:      nil,
			wantedString: now.In(time.UTC).Format(YYYYMMDDdDateLayout),
		},
		{
			funcName:     "ParseAndFmtDate",
			input:        YYYYMMDDMarchFirst,
			fmtLayout:    DashedDateLayout,
			layouts:      []string{YYYYMMDDdDateLayout},
			wantedString: Dashed20230301,
		},
		{
			funcName:     "ParseAndFmtDate",
			input:        YYYYMMDDMarchFirst,
			fmtLayout:    DashedDateLayout,
			layouts:      []string{DashedDateLayout, YYYYMMDDdDateLayout},
			wantedString: Dashed20230301,
		},
		{
			funcName:     "ParseAndFmtDate",
			input:        Dashed20230301,
			fmtLayout:    YYYYMMDDdDateLayout,
			layouts:      []string{YYYYMMDDdDateLayout},
			wantedString: "",
		},
		{
			funcName:     "ParseAndFmtDate",
			input:        "2023-04-12T23:15:56.290Z",
			fmtLayout:    "02/01/2006 15:04:05",
			layouts:      []string{"2006-01-02T15:04:05Z07:00"},
			wantedString: "13/04/2023 01:15:56",
			location:     "Local",
		},
		{
			funcName:     "ParseAndFmtDate",
			input:        "2023-04-12T23:15:56.290Z",
			fmtLayout:    "02/01/2006 15:04:05",
			layouts:      []string{"2006-01-02T15:04:05Z07:00"},
			wantedString: "12/04/2023 23:15:56",
			location:     "",
		},
		{
			funcName:  "DateDiff",
			input:     now,
			value2:    now,
			fmtLayout: "seconds",
			layouts:   nil,
			wantedInt: 0,
		},
		{
			funcName:  "DateDiff",
			input:     now,
			value2:    YYYYMMDDMarchFirst,
			fmtLayout: "days",
			layouts:   nil,
			wantedInt: 0,
		},
		// {
		// 	funcName:  "DateDiff",
		// 	input:     now,
		// 	value2:    YYYYMMDDMarchFirst,
		// 	fmtLayout: "days",
		// 	layouts:   []string{YYYYMMDDdDateLayout},
		// 	wantedInt: 42,
		// },
	}

	for ndx, dinput := range datesArr {
		switch dinput.funcName {
		case "IsDate":
			b := simple.IsDate(dinput.input, dinput.layouts...)
			require.Equal(t, b, dinput.wantedBool, fmt.Sprintf("error on simple.IsDate [%d]", ndx))
		case "ParseDate":
			tm := simple.ParseDate(dinput.input, dinput.location, dinput.layouts...)
			require.Equal(t, tm != nil, dinput.wantedBool, fmt.Sprintf("error on simple.ParseDate [%d]", ndx))
		case "ParseAndFmtDate":
			s := simple.ParseAndFmtDate(dinput.input, dinput.location, dinput.fmtLayout, dinput.layouts...)
			require.Equal(t, dinput.wantedString, s, fmt.Sprintf("error on simple.ParseAndFmtDate [%d]", ndx))
		case "DateDiff":
			i := simple.DateDiff(dinput.input, dinput.value2, dinput.fmtLayout, dinput.layouts...)
			require.Equal(t, dinput.wantedInt, i, fmt.Sprintf("error on simple.DateDiff [%d]", ndx))
		}
	}
}

func TestFloat64(t *testing.T) {
	f := 12000000.0
	sf := fmt.Sprintf("%v", f)
	t.Log(sf)

	f1, err := strconv.ParseFloat(sf, 32)
	require.NoError(t, err)

	t.Log(fmt.Sprintf("%d", int64(f1)))
}

func TestLocation(t *testing.T) {

	var locale = "Europe/Rome"

	var location, _ = time.LoadLocation(locale)

	//var location = time.Local

	//Data in input da leggere ISO8601 senza timezone
	var layout = "2006-01-02T15:04:05.999"
	var dateString = "2023-04-20T18:06:24.652"

	//Verifica con date con timezone
	//var layout = "2006-01-02T15:04:05Z07:00"
	//var dateString = "2023-04-20T18:06:24.652Z"

	//var layout = "2006-01-02T15:04:05Z07:00"
	//var dateString = "2023-04-20T18:06:24.652+01:00"

	var date, _ = time.Parse(layout, dateString)
	var dateLocal, _ = time.ParseInLocation(layout, dateString, location)

	fmt.Println(date)
	fmt.Println(dateLocal)
	fmt.Println(dateLocal.In(location))
}
