package config

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
	"strings"
)

type Dictionaries []Dictionary

type Dictionary struct {
	Name string
	Dict map[string]string
}

func (d Dictionary) Map(elems ...string) string {
	k := strings.Join(elems, "_")
	if v, ok := d.Dict[k]; ok {
		return fmt.Sprintf("%v", v)
	}

	log.Warn().Str("key", k).Msg("unresolved dictionary lookup")
	return ""
}

func NewDictionary(n string, data []byte) (Dictionary, error) {

	d := Dictionary{Name: n, Dict: make(map[string]string)}
	err := yaml.Unmarshal(data, d.Dict)
	return d, err
}

func (dicts Dictionaries) Map(dictName string, elems ...string) (string, error) {
	if len(dicts) == 0 {
		log.Error().Str("dict-name", dictName).Msg("unresolved dictionary name")
		return "", fmt.Errorf("no dicts loaded")
	}

	dictNdx := dicts.findDictByName(dictName)
	if dictNdx < 0 {
		log.Error().Str("dict-name", dictName).Msg("unresolved dictionary name")
		return "", fmt.Errorf("dictionary not resolved (name: %s)", dictName)
	}

	return dicts[dictNdx].Map(elems...), nil
}

func (dicts Dictionaries) findDictByName(n string) int {
	for i, d := range dicts {
		if d.Name == n {
			return i
		}
	}

	return -1
}
