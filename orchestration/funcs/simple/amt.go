package simple

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
)

// type AmountUnit string

const (
	Dime          string = "dime"
	Cent                 = "cent"
	DecimalCent          = "decimal-2"
	DecimalMillis        = "decimal-3"
	Mill                 = "mill"
	DeciMill             = "deci-mill"
	MicroCent            = "micro"

	ConversionMapKetFormat = "%s_to_%s"

	AmountOpAdd  = "add"
	AmountOpDiff = "diff"
)

var amountConversionsMap = map[string]func(string) string{
	fmt.Sprintf(ConversionMapKetFormat, Cent, Mill):      func(a string) string { return a + "0" },
	fmt.Sprintf(ConversionMapKetFormat, Cent, MicroCent): func(a string) string { return a + "0000" },
	fmt.Sprintf(ConversionMapKetFormat, Cent, Cent):      func(a string) string { return a },
	fmt.Sprintf(ConversionMapKetFormat, Mill, Mill):      func(a string) string { return a },
	fmt.Sprintf(ConversionMapKetFormat, Mill, Cent): func(a string) string {
		if len(a) > 1 {
			return a[0 : len(a)-1]
		}

		return "0"
	},

	fmt.Sprintf(ConversionMapKetFormat, MicroCent, Cent): func(a string) string {
		if len(a) > 4 {
			return a[0 : len(a)-4]
		}

		return "0"
	},
}

func toDecimalFormat(s string) string {
	if len(s) < 3 {
		s = "000" + s
		s = s[len(s)-3:]
	}
	return s[:len(s)-2] + "." + s[len(s)-2:]
}

func AmtAdd(sourceUnit, targetUnit string, decimalFormat bool, amts ...interface{}) (string, error) {
	return Amt(AmountOpAdd, sourceUnit, targetUnit, decimalFormat, amts...)
}

func AmtDiff(sourceUnit, targetUnit string, decimalFormat bool, amts ...interface{}) (string, error) {
	return Amt(AmountOpDiff, sourceUnit, targetUnit, decimalFormat, amts...)
}

func AmtConv(sourceUnit, targetUnit string, decimalFormat bool, amt interface{}) (string, error) {
	const semLogContext = "funcs::amt-conv"

	var exprSourceUnit string

	iAmt, amtSourceUnit, err := evaluate(amt, sourceUnit)
	if amtSourceUnit != exprSourceUnit && exprSourceUnit != "" {
		log.Warn().Str("amt-source-unit", amtSourceUnit).Str("expr-source-unit", exprSourceUnit).Msg(semLogContext + " source unit changed over computation")
	}
	exprSourceUnit = amtSourceUnit

	if err != nil {
		val := "0"
		if decimalFormat {
			val = toDecimalFormat(val)
		}
		log.Error().Err(err).Str("result", val).Interface("amount", amt).Msg("conversion error")
		return val, err
	}

	if iAmt < 0 {
		log.Warn().Int64("total", iAmt).Msg("Op gives a negative result")
	}

	convName := fmt.Sprintf(ConversionMapKetFormat, exprSourceUnit, targetUnit)
	if f, ok := amountConversionsMap[convName]; ok {
		val := f(fmt.Sprintf("%d", iAmt))
		if decimalFormat {
			val = toDecimalFormat(val)
		}
		return val, nil
	}

	val := "0"
	if decimalFormat {
		val = toDecimalFormat(val)
	}
	err = fmt.Errorf("conversion not supported from %s to %s", string(exprSourceUnit), string(targetUnit))
	log.Error().Err(err).Str("result", val).Int64("total", iAmt).Msg(semLogContext)
	return val, err
}

func AmtCmp(cmpUnit string, amt1, amt1Unit string, amt2, amt2Unit string) (bool, error) {
	var err error
	if amt1Unit != cmpUnit {
		amt1, err = Amt(AmountOpAdd, amt1Unit, cmpUnit, false, amt1)
		if err != nil {
			return false, err
		}
	}

	if amt2Unit != cmpUnit {
		amt2, err = Amt(AmountOpAdd, amt2Unit, cmpUnit, false, amt2)
		if err != nil {
			return false, err
		}
	}

	if len(amt1) > len(amt2) {
		amt2 = fmt.Sprintf("%s%s", strings.Repeat("0", len(amt1)-len(amt2)), amt2)
	}

	if len(amt2) > len(amt1) {
		amt1 = fmt.Sprintf("%s%s", strings.Repeat("0", len(amt2)-len(amt1)), amt1)
	}

	return strings.Compare(amt1, amt2) > 0, nil
}

func Amt(opType string, sourceUnit, targetUnit string, decimalFormat bool, amts ...interface{}) (string, error) {

	const semLogContext = "funcs::amt"

	var total int64
	var exprSourceUnit string
	for aNdx, a := range amts {

		i, amtSourceUnit, err := evaluate(a, sourceUnit)
		if amtSourceUnit != exprSourceUnit && exprSourceUnit != "" {
			log.Warn().Str("amt-source-unit", amtSourceUnit).Str("expr-source-unit", exprSourceUnit).Msg(semLogContext + " source unit changed over computation")
		}
		exprSourceUnit = amtSourceUnit

		if err != nil {
			val := "0"
			if decimalFormat {
				val = toDecimalFormat(val)
			}
			log.Error().Err(err).Str("result", val).Interface("amount", a).Msg("conversion error")
			return val, err
		}

		total = amtOp(opType, total, aNdx, i)
	}

	if total < 0 {
		log.Warn().Int64("total", total).Msg("Op gives a negative result")
	}

	convName := fmt.Sprintf(ConversionMapKetFormat, exprSourceUnit, targetUnit)
	if f, ok := amountConversionsMap[convName]; ok {
		val := f(fmt.Sprintf("%d", total))
		if decimalFormat {
			val = toDecimalFormat(val)
		}
		return val, nil
	}

	val := "0"
	if decimalFormat {
		val = toDecimalFormat(val)
	}
	err := fmt.Errorf("conversion not supported from %s to %s", string(exprSourceUnit), string(targetUnit))
	log.Error().Err(err).Str("result", val).Int64("total", total).Msg(semLogContext)
	return val, err
}

func evaluate(a interface{}, sourceUnit string) (int64, string, error) {
	var f float64
	var i int64
	var err error
	switch ta := a.(type) {
	case string:
		if strings.Index(ta, ",") >= 0 {
			ta = strings.ReplaceAll(ta, ",", ".")
		}

		if strings.Index(ta, ".") >= 0 {
			f, err = strconv.ParseFloat(ta, 64)
			if err == nil {
				switch sourceUnit {
				case Cent:
					i = int64(f)
				case DecimalCent:
					f100 := math.Round(f * 100) // used a temp variable to better analyze in debug whats is going on in this case.
					i = int64(f100)
					sourceUnit = Cent
				case DecimalMillis:
					i = int64(math.Round(f * 1000))
					sourceUnit = Mill
				default:
					i = int64(f)
				}
			}
		} else {
			i, err = strconv.ParseInt(ta, 10, 64)
			if err == nil {
				switch sourceUnit {
				case DecimalCent:
					i = i * 100
					sourceUnit = Cent
				case DecimalMillis:
					i = i * 1000
					sourceUnit = Mill
				}
			}
		}
	case int64:
		i = ta
	case int32:
		i = int64(ta)
	case int:
		i = int64(ta)
	}
	return i, sourceUnit, err
}

func amtOp(opType string, total int64, ndx int, i int64) int64 {

	switch opType {
	case AmountOpAdd:
		total = total + i
	case AmountOpDiff:
		if ndx == 0 {
			total = i
		} else {
			total = total - i
		}
	}

	return total
}
