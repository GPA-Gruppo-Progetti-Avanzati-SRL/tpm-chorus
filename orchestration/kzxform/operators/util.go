package operators

import (
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"regexp"

	"github.com/qntfy/jsonparser"
	"github.com/qntfy/kazaam/transform"
)

func GetJsonArray(data []byte, ref JsonReference) ([]byte, error) {

	targetArray, vt, _, _ := jsonparser.Get(data, ref.Keys...)
	if vt != jsonparser.Array {
		err := fmt.Errorf("target-ref '%s' is not array", ref.Path)
		return nil, err
	}

	return targetArray, nil
}

func GetJsonString(data []byte, targetRef string, required bool) (string, error) {
	targetRefKeys, withI, _, err := SplitKeySpecifier(targetRef)
	if err != nil {
		return "", err
	}

	if withI >= 0 {
		return "", errors.New("get-json-string - reference does not support [i] wildcard")
	}

	jsonValue, dataType, _, err := jsonparser.Get(data, targetRefKeys...)
	if err != nil {
		return "", err
	}

	if dataType == jsonparser.NotExist {
		if required {
			return "", fmt.Errorf("%s is not present but is required", targetRef)
		}

		return "", nil
	}

	if dataType != jsonparser.String {
		return "", fmt.Errorf("%s is not a string but %s", targetRef, jsonValue)
	}

	return string(jsonValue), nil
}

func GetJsonValue(data []byte, targetRef string) (jsonValue []byte, dataType jsonparser.ValueType, err error) {
	targetRefKeys, withI, _, err := SplitKeySpecifier(targetRef)
	if err != nil {
		return nil, jsonparser.NotExist, err
	}

	if withI >= 0 {
		return nil, jsonparser.NotExist, errors.New("get-json-value - reference does not support [i] wildcard")
	}

	jsonValueRes, dataTypeRes, _, err := jsonparser.Get(data, targetRefKeys...)
	if err != nil {
		return nil, jsonparser.NotExist, err
	}

	if dataTypeRes == jsonparser.NotExist {
		return nil, dataTypeRes, nil
	}

	return jsonValueRes, dataTypeRes, nil
}

var KeyPatternRegexpOld = regexp.MustCompile("([a-zA-Z0-9-_]+|\\.|\\[\\*])")
var KeyPatternRegexp = regexp.MustCompile("([a-zA-Z0-9-_]+|\\.|\\[[0-9*i+]\\])")

func SplitKeySpecifier(k string) ([]string, int, int, error) {
	matches := KeyPatternRegexp.FindAllSubmatch([]byte(k), -1)

	withIota := -1
	withPlus := -1
	elemNdx := 0
	var res []string
	for _, m := range matches {
		captured := string(m[1])
		if captured != "." {
			if captured == "[i]" {
				if withIota != -1 {
					return nil, -1, -1, errors.New("invalid syntax for key specifier: " + k)
				}
				withIota = elemNdx
			} else if captured == "[+]" {
				if withPlus != -1 {
					return nil, -1, -1, errors.New("invalid syntax for key specifier: " + k)
				}
				withPlus = elemNdx
			}
			elemNdx++
			res = append(res, captured)
		}
	}

	return res, withIota, withPlus, nil
}

func GetJsonReferenceParam(spec *transform.Config, n string, required bool) (JsonReference, error) {
	/*
		var err error
		param, ok := (*spec.Spec)[n]
		if !ok {
			if required {
				err = fmt.Errorf("cannot find json reference param %s in specs", n)
			}
			return JsonReference{}, err
		}

		s, ok := param.(string)
		if !ok {
			return JsonReference{}, fmt.Errorf("param %s is not a string but %T", n, param)
		}
	*/

	s, err := GetStringParam(spec, n, required, "")
	if s == "" {
		return JsonReference{}, err
	}

	return ToJsonReference(s)
}

func GetStringParam(spec *transform.Config, n string, required bool, defaultValue string) (string, error) {
	param, ok := (*spec.Spec)[n]
	if !ok {
		if required {
			return "", fmt.Errorf("cannot find string param %s in specs", n)
		}

		return defaultValue, nil
	}

	s, ok := param.(string)
	if !ok {
		return "", fmt.Errorf("param %s is not a string but %T", n, param)
	}

	return s, nil
}

func GetBoolParam(spec *transform.Config, n string, required bool) (bool, error) {

	param, ok := (*spec.Spec)[n]
	if !ok {
		if required {
			return false, fmt.Errorf("cannot find string param %s in specs", n)
		}
		return false, nil
	}

	b, ok := param.(bool)
	if !ok {
		return false, fmt.Errorf("param %s is not a bool but %T", n, param)
	}

	return b, nil
}

func GetArrayParam(spec *transform.Config, n string, required bool) ([]interface{}, error) {
	param, ok := (*spec.Spec)[n]
	if !ok {
		if required {
			return nil, fmt.Errorf("cannot find array param %s in specs", n)
		}

		return nil, nil
	}

	arr, ok := param.([]interface{})
	if !ok {
		return nil, fmt.Errorf("param %s is not an array but %T", n, param)
	}

	return arr, nil
}

func GetStringParamFromMap(spec interface{}, n string, required bool) (string, error) {

	param, err := GetParamFromMap(spec, n, required)
	if err != nil {
		return "", nil
	}

	if param == nil {
		return "", nil
	}

	s, ok := param.(string)
	if !ok {
		return "", fmt.Errorf("param %s is not a string but %T", n, param)
	}

	return s, nil
}

func GetArrayParamFromMap(spec interface{}, n string, required bool) ([]interface{}, error) {
	param, err := GetParamFromMap(spec, n, required)
	if err != nil {
		return nil, nil
	}

	arr, ok := param.([]interface{})
	if !ok {
		return nil, fmt.Errorf("param %s is not an array but %T", n, param)
	}

	return arr, nil
}

func GetJsonReferenceParamFromMap(spec interface{}, n string, required bool) (JsonReference, error) {
	var err error

	param, err := GetStringParamFromMap(spec, n, required)
	if err != nil {
		return JsonReference{}, nil
	}

	return ToJsonReference(param)
}

func GetBoolParamFromMap(spec interface{}, n string, required bool) (bool, error) {

	param, err := GetParamFromMap(spec, n, required)
	if err != nil {
		return false, nil
	}

	if param == nil {
		return false, nil
	}

	b, ok := param.(bool)
	if !ok {
		return false, fmt.Errorf("param %s is not a bool but %T", n, param)
	}

	return b, nil
}

func GetParamFromMap(spec interface{}, n string, required bool) (interface{}, error) {

	m, ok := spec.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("expected a map[string]interface{} and got %T", spec)
	}

	param, ok := m[n]
	if !ok {
		if required {
			return nil, fmt.Errorf("cannot find param %s in map", n)
		}

		return nil, nil
	}

	return param, nil
}

// LenOfArray iterate over an array and compute the length. If jsonRef is specified the actual array is a subproperty of the object provided ad data
func LenOfArray(data []byte, jsonRef JsonReference) (int, error) {
	const semLogContext = "kazaam-util::len-of-array"

	var err error
	var nestedArray []byte
	if !jsonRef.IsZero() {
		nestedArray, err = GetJsonArray(data, jsonRef)
	} else {
		nestedArray = data
	}

	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return -1, err
	}

	nestedLoopIndex := 0
	_, err = jsonparser.ArrayEach(nestedArray, func(value []byte, dataType jsonparser.ValueType, offset int, errParam error) {
		nestedLoopIndex++
	})

	return nestedLoopIndex, nil
}
