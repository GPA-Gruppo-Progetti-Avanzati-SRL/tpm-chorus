package amt

import (
	"errors"
	"fmt"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util"
	"github.com/rs/zerolog/log"
	"strconv"
	"strings"
)

var formatConversionMap = map[string]func(i int64, d string) string{
	fmt.Sprintf(ConversionMapKetFormat, MicroCent, Cent): fmtConvMicroCent2Cent,
	fmt.Sprintf(ConversionMapKetFormat, Mill, Mill):      fmtConvMill2Mill,
	fmt.Sprintf(ConversionMapKetFormat, Mill, Cent):      fmtConvMillToCent,
	fmt.Sprintf(ConversionMapKetFormat, Cent, MicroCent): fmtConvCent2MicroCent,
	fmt.Sprintf(ConversionMapKetFormat, Cent, Mill):      fmtConvCent2Mill,
	fmt.Sprintf(ConversionMapKetFormat, Cent, Cent):      fmtConvCent2Cent,
	fmt.Sprintf(ConversionMapKetFormat, Cent, Decimal):   fmtConvCent2Decimal,
	fmt.Sprintf(ConversionMapKetFormat, Decimal, Cent):   fmtConvDecimal2Cent,
}

func Format(a interface{}, sourceFormat string, targetFormat string, negate bool) (string, error) {
	const semLogContext = "chorus-func-amt::format"
	var intPart int64
	var decimalPart string
	var err error
	var isNegative bool

	intPart, decimalPart, isNegative, err = split(a, sourceFormat)
	if err != nil {
		return "", err
	}

	if _, ok := IntegralAmtFormats[sourceFormat]; ok {
		if decimalPart != "" {
			err = errors.New("specified source format invalid")
			log.Error().Err(err).Str("source-format", sourceFormat).Interface("amt", a).Msg(semLogContext)
			return "", err
		}
	}

	res := ""
	if fn, ok := formatConversionMap[fmt.Sprintf(ConversionMapKetFormat, sourceFormat, targetFormat)]; ok {
		res = fn(intPart, decimalPart)
		if intPart > 0 || decimalPart != "" {
			if negate != isNegative {
				res = "-" + res
			}
		}
	} else {
		err = errors.New("source/target format pair unsupported")
		log.Error().Err(err).Interface("amt", a).Str("sourc-format", sourceFormat).Str("target-format", targetFormat).Msg(semLogContext)
	}

	return res, err
}

func split(a interface{}, sourceFormat string) (int64, string, bool, error) {
	const semLogContext = "chorus-func-amt::split"

	var err error
	var isNegative bool

	if a == nil {
		return 0, "", false, nil
	}

	var intPart int64
	var decimalPart string
	switch ta := a.(type) {
	case string:
		intPart, decimalPart, isNegative, err = splitString(ta, sourceFormat)
	case int64:
		intPart = ta
		decimalPart = ""
		isNegative = ta < 0
	case int:
		intPart = int64(ta)
		decimalPart = ""
		isNegative = ta < 0
	case int32:
		intPart = int64(ta)
		decimalPart = ""
		isNegative = ta < 0
	default:
		intPart = -1
		decimalPart = ""
		isNegative = false
		err = fmt.Errorf("unsupported type: %T", a)
	}

	if err != nil {
		log.Error().Err(err).Interface("amt", a).Msg(semLogContext)
	} else {
		log.Trace().Interface("amt", a).Int64("int-part", intPart).Str("decimal-part", decimalPart).Bool("negative", isNegative).Msg(semLogContext)
	}

	return intPart, decimalPart, isNegative, err
}

func splitString(ta string, sourceFormat string) (int64, string, bool, error) {
	const semLogContext = "chorus-func-amt::split-string"

	var err error
	var isNegative bool

	var intPart int64
	if strings.HasPrefix(ta, "-") {
		isNegative = true
		ta = ta[1:]
	}

	ta = strings.ReplaceAll(ta, ",", ".")
	parts := strings.Split(ta, ".")
	if len(parts) > 2 {
		err = errors.New("amt-fmt: too many parts")
		return 0, "", false, err
	}

	iPart := util.TrimPrefixCharacters(parts[0], "0")
	dPart := ""
	if len(parts) == 2 {
		dPart = util.TrimSuffixCharacters(parts[1], "0")
	}

	intPart = 0
	if iPart != "" {
		intPart, err = strconv.ParseInt(iPart, 10, 64)
		if err != nil {
			return 0, "", false, err
		}
	}

	if dPart != "" {
		_, err = strconv.ParseInt(dPart, 10, 64)
		if err != nil {
			return 0, "", false, err
		}
	}

	return intPart, dPart, isNegative, err
}
