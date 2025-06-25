package funcs_test

import (
	"fmt"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/funcs/purefuncs"
	"os"
	"strconv"
	"testing"
	"time"

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
			news = purefuncs.Printf("%12s", s.input)
		case "PadLeft":
			news = purefuncs.PadLeft(s.input, 12, "0")
		case "Left":
			news = purefuncs.Left(s.input, 5)
		case "Right":
			news = purefuncs.Right(s.input, 5)
		default:
			t.Fatalf("func %s not present", s.funcName)
		}

		require.Equal(t, s.wanted, news)
	}

	/*
		news, err = simple.AmtConv(convArr[12].sourceUnit, convArr[12].targetUnit, convArr[12].decimalFormat, convArr[12].amts[0])
		require.NoError(t, err)
		require.Equal(t, convArr[12].wanted, news, fmt.Sprintf("error on funcs.Amt [%d]", 12))
	*/

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
		{
			funcName:     "ParseAndFmtDate",
			input:        "2024-07-11 16:55:43.747975000000",
			fmtLayout:    "02/01/2006 15:04:05",
			layouts:      []string{"2006-01-02 15:04:05.000000000000"},
			wantedString: "11/07/2024 16:55:43",
			location:     "",
		},
		{
			funcName:  "Age",
			input:     "2023-12-09",
			layouts:   []string{"2006-01-02"},
			wantedInt: 0,
		},
		{
			funcName:   "DateAdd",
			input:      time.Now(),
			layouts:    nil,
			wantedBool: true,
		},
		{
			funcName:   "DateAdd",
			input:      "2025-06-19T10:29:48.043+02:00",
			layouts:    []string{time.RFC3339},
			wantedBool: true,
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
			b := purefuncs.IsDate(dinput.input, dinput.layouts...)
			require.Equal(t, b, dinput.wantedBool, fmt.Sprintf("error on simple.IsDate [%d]", ndx))
		case "ParseDate":
			tm := purefuncs.ParseDate(dinput.input, dinput.location, dinput.layouts...)
			require.Equal(t, tm != nil, dinput.wantedBool, fmt.Sprintf("error on simple.ParseDate [%d]", ndx))
		case "ParseAndFmtDate":
			s := purefuncs.ParseAndFmtDate(dinput.input, dinput.location, dinput.fmtLayout, dinput.layouts...)
			require.Equal(t, dinput.wantedString, s, fmt.Sprintf("error on simple.ParseAndFmtDate [%d]", ndx))
		case "DateDiff":
			i := purefuncs.DateDiff(dinput.input, dinput.value2, dinput.fmtLayout, dinput.layouts...)
			require.Equal(t, dinput.wantedInt, i, fmt.Sprintf("error on simple.DateDiff [%d]", ndx))
		//case "Age":
		//	i := purefuncs.Age(dinput.input, "include", dinput.layouts...)
		//	require.Equal(t, dinput.wantedInt, i, fmt.Sprintf("error on age [%d]", ndx))
		case "DateAdd":
			tm := purefuncs.DateAdd(dinput.input, -1, 2, 3, 4, 0, 30, dinput.layouts...)
			require.Equal(t, tm != nil, dinput.wantedBool, fmt.Sprintf("error on DateAdd [%d]", ndx))

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

func TestParseDate(t *testing.T) {
	// var locale = "Europe/Rome"
	// var location, _ = time.LoadLocation(locale)

	var layout = "2006-01-02 15:04:05.999999"
	var dateString = "2024-07-11 16:55:43.747975000000"

	date, err := time.Parse(layout, dateString)
	require.NoError(t, err)
	fmt.Println(date)

}

func TestRegex(t *testing.T) {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	var iws = []struct {
		input   string
		pattern []string
		wanted  string
	}{
		{
			pattern: []string{"¢(?:\\s)?([0-9]+(?:,|\\.)?[0-9]*)", "zero"},
			input:   "¢ zero per 12 mesi",
			wanted:  "0",
		},
		{
			pattern: []string{"¢(?:\\s)?([0-9]+(?:,|\\.)?[0-9]*)", "zero"},
			input:   "¢ zorro per 12 mesi",
			wanted:  "",
		},
		{
			pattern: []string{"¢(?:\\s)?([0-9]+(?:,|\\.)?[0-9]*)"},
			input:   "zero per 12 mesi ¢ 2,34 ",
			wanted:  "2,34",
		},
		{
			pattern: []string{"¢(?:\\s)?([0-9]+(?:,|\\.)?[0-9]*)"},
			input:   "¢ 2.3.4",
			wanted:  "2.3",
		},
		{
			pattern: []string{"¢(?:\\s)?([0-9]+(?:,|\\.)?[0-9]*)"},
			input:   "¢23.4",
			wanted:  "23.4",
		},
		{
			pattern: []string{"([0-9]+)(?:\\s)?mesi"},
			input:   "12 mesi",
			wanted:  "12",
		},
	}

	for ndx, iw := range iws {
		s := purefuncs.RegexSetMatchAndExtract(iw.input, "0", "", iw.pattern...)
		require.Equal(t, iw.wanted, s, fmt.Sprintf("error on regex [%d]", ndx))
		t.Log(ndx, s)
	}
}
