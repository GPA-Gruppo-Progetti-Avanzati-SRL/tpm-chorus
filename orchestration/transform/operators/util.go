package operators

import (
	"fmt"
	"github.com/qntfy/jsonparser"
	"github.com/qntfy/kazaam/transform"
	"regexp"
)

func getJsonArray(data []byte, ref JsonReference) ([]byte, error) {

	targetArray, vt, _, _ := jsonparser.Get(data, ref.Keys...)
	if vt != jsonparser.Array {
		err := fmt.Errorf("target-ref '%s' is not array", ref.Path)
		return nil, err
	}

	return targetArray, nil
}

func getJsonString(data []byte, targetRef string, required bool) (string, error) {
	targetRefKeys, err := SplitKeySpecifier(targetRef)
	if err != nil {
		return "", err
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

func getJsonValue(data []byte, targetRef string) ([]byte, error) {
	targetRefKeys, err := SplitKeySpecifier(targetRef)
	if err != nil {
		return nil, err
	}

	jsonValue, dataType, _, err := jsonparser.Get(data, targetRefKeys...)
	if err != nil {
		return nil, err
	}

	if dataType == jsonparser.NotExist {
		return nil, nil
	}

	return jsonValue, nil
}

var KeyPatternRegexp = regexp.MustCompile("([a-zA-Z0-9-_]+|\\.|\\[\\*])")

func SplitKeySpecifier(k string) ([]string, error) {
	matches := KeyPatternRegexp.FindAllSubmatch([]byte(k), -1)

	var res []string
	for _, m := range matches {
		captured := string(m[1])
		if captured != "." {
			res = append(res, captured)
		}
	}

	return res, nil
}

func getJsonReferenceParam(spec *transform.Config, n string, required bool) (JsonReference, error) {
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

	s, err := getStringParam(spec, n, required, "")
	if s == "" {
		return JsonReference{}, err
	}

	return ToJsonReference(s)
}

func getStringParam(spec *transform.Config, n string, required bool, defaultValue string) (string, error) {
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

func getArrayParam(spec *transform.Config, n string, required bool) ([]interface{}, error) {
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

func getStringParamFromMap(spec interface{}, n string, required bool) (string, error) {

	param, err := getParamFromMap(spec, n, required)
	if err != nil {
		return "", nil
	}

	s, ok := param.(string)
	if !ok {
		return "", fmt.Errorf("param %s is not a string but %T", n, param)
	}

	return s, nil
}

func getJsonReferenceParamFromMap(spec interface{}, n string, required bool) (JsonReference, error) {
	var err error

	param, err := getStringParamFromMap(spec, n, required)
	if err != nil {
		return JsonReference{}, nil
	}

	return ToJsonReference(param)
}

func getBoolParamFromMap(spec interface{}, n string, required bool) (bool, error) {

	param, err := getParamFromMap(spec, n, required)
	if err != nil {
		return false, nil
	}

	b, ok := param.(bool)
	if !ok {
		return false, fmt.Errorf("param %s is not a bool but %T", n, param)
	}

	return b, nil
}

func getParamFromMap(spec interface{}, n string, required bool) (interface{}, error) {

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
