package addarrayitems

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/kzxform/operators"
	"strconv"

	"github.com/qntfy/jsonparser"
	"github.com/qntfy/kazaam"
	"github.com/qntfy/kazaam/transform"
	"github.com/rs/zerolog/log"
	"strings"
)

const (
	OperatorAddArrayItems = "add-array-items"
	OperatorSemLogContext = OperatorAddArrayItems
)

func AddArrayItems(kc kazaam.Config) func(spec *transform.Config, data []byte) ([]byte, error) {
	return func(spec *transform.Config, data []byte) ([]byte, error) {

		const semLogContext = OperatorSemLogContext + "::execute"

		params, err := getParamsFromSpec(spec)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return nil, err
		}

		if len(params.mapping) == 0 {
			return data, nil
		}

		// clone the data... in place process has some glitches.
		outData := make([]byte, len(data))
		copy(outData, data)

		for _, m := range params.mapping {
			outData, err = process(outData, m.dst, data, m.src)
			if err != nil {
				log.Error().Err(err).Msg(semLogContext)
				return nil, err
			}
		}

		return outData, err
	}
}

func process(outData []byte, dst operators.JsonReference, data []byte, src operators.JsonReference) ([]byte, error) {
	const semLogContext = OperatorSemLogContext + "::process"

	var rootRef operators.JsonReference
	if src.WithArrayISpecifierIndex >= 0 {
		rootRef = operators.JsonReference{
			WithArrayISpecifierIndex: -1,
			Path:                     src.Path[:strings.Index(src.Path, "[i]")],
			Keys:                     src.Keys[:src.WithArrayISpecifierIndex],
		}
	} else {
		rootRef = src
	}

	rootArray, err := operators.GetJsonArray(data, rootRef)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return nil, err
	}

	var loopErr error
	var loopIndex int
	adder := Adder{}
	_, err = jsonparser.ArrayEach(rootArray, func(value []byte, dataType jsonparser.ValueType, offset int, errParam error) {
		if loopErr != nil {
			log.Error().Err(err).Msg(semLogContext + " previous error in for-each")
			return
		}

		valueToAdd := value
		dt := dataType
		if src.WithArrayISpecifierIndex >= 0 {
			// Questo mapping crea riferimenti del tipo [0].prop1.prop2, [1].prop1.prop2.... toglie la parte a monte per eseguire il mapping
			// con la variabile rootArray
			nestedRef := operators.JsonReference{
				WithArrayISpecifierIndex: -1,
				Path:                     strings.ReplaceAll(src.Path[strings.Index(src.Path, "[i]"):], "[i]", fmt.Sprintf("[%d]", loopIndex)),
				// Keys:                     make([]string, len(sourceRef.Keys)),
			}
			nestedRef.Keys = append(nestedRef.Keys, src.Keys[src.WithArrayISpecifierIndex:]...)
			nestedRef.Keys[0] = fmt.Sprintf("[%d]", loopIndex)
			valueToAdd, dt, _, err = jsonparser.Get(rootArray, nestedRef.Keys...)
			if err != nil {
				log.Error().Err(err).Msg(semLogContext)
				loopErr = err
				return
			}
		}

		err = adder.add(valueToAdd, dt)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			loopErr = err
			return
		}

		log.Info().Str("value-to-add", string(valueToAdd)).Interface("of-data-type", dt).Msg(semLogContext)
		loopIndex++
	})

	result, err := adder.result()
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return nil, err
	}

	var nestedTargetRefKeys []string
	nestedTargetRefKeys = append(nestedTargetRefKeys, dst.Keys...)
	outData, err = jsonparser.Set(outData, result, nestedTargetRefKeys...)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return nil, err
	}

	return outData, nil
}

type Adder struct {
	numberFormat string
	jsonFormat   string
	valuesToAdd  []interface{}
}

const (
	NumberFormatInteger = "integer"
	NumberFormatFloat   = "float"
	JsonFormatString    = "string"
	JsonFormatNumber    = "number"
	JsonFormatMixed     = "mixed"
)

func (ad *Adder) result() ([]byte, error) {
	if len(ad.valuesToAdd) == 0 {
		return []byte("0"), nil
	}

	resultVal := "0"
	switch ad.numberFormat {
	case NumberFormatInteger:
		i64, err := int64Add(ad.valuesToAdd)
		if err != nil {
			return nil, err
		}
		resultVal = fmt.Sprint(i64)
	case NumberFormatFloat:
		fl64, err := float64Add(ad.valuesToAdd)
		if err != nil {
			return nil, err
		}
		resultVal = fmt.Sprintf("%f", fl64)
	default:
		// Not applicable
	}

	if ad.jsonFormat == JsonFormatString {
		var sb bytes.Buffer
		sb.WriteString("\"")
		sb.WriteString(resultVal)
		sb.WriteString("\"")
		return sb.Bytes(), nil
	}

	return []byte(resultVal), nil
}

func int64Add(values []interface{}) (int64, error) {
	summ := int64(0)
	for _, v := range values {
		summ += v.(int64)
	}

	return summ, nil
}

func float64Add(values []interface{}) (float64, error) {

	summ := float64(0.0)
	for _, v := range values {
		switch tv := v.(type) {
		case float64:
			summ += tv
		case int64:
			summ += float64(tv)
		}
	}

	return summ, nil
}

func (ad *Adder) add(value []byte, dt jsonparser.ValueType) error {
	const semLogContext = OperatorSemLogContext + "::add"
	var err error

	nf := NumberFormatInteger
	jf := JsonFormatNumber
	s := string(value)
	switch dt {
	case jsonparser.String:
		jf = JsonFormatString
		nf, err = ad.appendValue(s)

	case jsonparser.Number:
		jf = JsonFormatNumber
		nf, err = ad.appendValue(s)
	default:
	case jsonparser.Boolean:
		err = errors.New("unsupported value type in add-array-items")
		log.Error().Err(err).Str("value", string(value)).Msg(semLogContext)
	}

	if err != nil {
		return err
	}

	if ad.jsonFormat == "" {
		ad.jsonFormat = jf
	} else if ad.jsonFormat != jf {
		ad.jsonFormat = JsonFormatMixed
	}

	if ad.numberFormat == "" {
		ad.numberFormat = NumberFormatInteger
	}

	if nf == NumberFormatFloat {
		ad.numberFormat = NumberFormatFloat
	}

	return nil
}

func (ad *Adder) appendValue(s string) (string, error) {
	const semLogContext = OperatorSemLogContext + "::add-string-value"
	nf := NumberFormatInteger
	var err error

	if len(s) > 0 {
		if strings.Index(s, ",") >= 0 || strings.Index(s, ".") >= 0 {
			nf = NumberFormatFloat
			s = strings.Replace(s, ",", ".", -1)
			var fl float64
			if fl, err = strconv.ParseFloat(s, 64); err == nil {
				ad.valuesToAdd = append(ad.valuesToAdd, fl)
			}
		} else {
			var i64 int64
			if i64, err = strconv.ParseInt(s, 10, 64); err == nil {
				ad.valuesToAdd = append(ad.valuesToAdd, i64)
			}
		}
	} else {
		log.Info().Str("value", s).Msg(semLogContext + " - skipping empty value")
	}

	return nf, err
}
