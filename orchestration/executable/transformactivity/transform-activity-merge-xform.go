package transformactivity

import (
	"encoding/json"
	"fmt"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/wfcase"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
	"strings"
)

type MergeXFormSource struct {
	ActivityName string `yaml:"activity,omitempty"  json:"activity,omitempty" mapstructure:"activity,omitempty"`
	Dest         string `yaml:"dest,omitempty" json:"dest,omitempty" mapstructure:"dest,omitempty"`
	Guard        string `yaml:"guard,omitempty" mapstructure:"guard,omitempty" json:"guard,omitempty"`
}

type MergeXForm struct {
	Sources []MergeXFormSource `yaml:"sources,omitempty"  json:"sources,omitempty" mapstructure:"sources,omitempty"`
}

func NewTransformActivityMergeXForm(definition []byte) (MergeXForm, error) {
	xform := MergeXForm{}
	err := yaml.Unmarshal(definition, &xform)
	if err != nil {
		return xform, err
	}

	return xform, nil
}

func (xform MergeXForm) Execute(wfc *wfcase.WfCase, data []byte) ([]byte, error) {
	const semLogContext = "transform-activity-merge-xform::execute"
	var m map[string]interface{}

	err := json.Unmarshal(data, &m)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return nil, err
	}

	for _, src := range xform.Sources {

		// Skip a merge transformation if there is a guard condition.
		if !wfc.EvalBoolExpression(src.Guard) {
			continue
		}

		var expressionCtx wfcase.HarEntryReference
		expressionCtx, err = wfc.ResolveHarEntryReferenceByName(src.ActivityName)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return nil, err
		}
		log.Trace().Str("expr-scope", expressionCtx.Name).Msg(semLogContext)

		var b []byte
		b, _, err = wfc.GetBodyInHarEntry(expressionCtx, true)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return nil, err
		}

		switch jsonStructure(string(b)) {
		case jsonStructureMap:
			if src.Dest == "" {
				err = json.Unmarshal(b, &m)
			} else {
				var temp map[string]interface{}
				err = json.Unmarshal(b, &temp)
				if err == nil {
					m, err = SetMapProperty(m, src.Dest, temp)
					if err != nil {
						log.Error().Err(err).Msg(semLogContext)
					}
				}
			}
		case jsonStructureArray:
			var temp []interface{}
			destProperty := src.Dest
			if src.Dest == "" {
				destProperty = uuid.New().String()
			}
			err = json.Unmarshal(b, &temp)
			if err == nil {
				m, err = SetMapProperty(m, destProperty, temp)
				if err != nil {
					log.Error().Err(err).Msg(semLogContext)
				}
			}
		}

		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return nil, err
		}
	}

	return json.Marshal(m)
}

// TODO: switch to tpm-common util.JSONStructure
const (
	jsonStructureMap    = "map"
	jsonStructureArray  = "array"
	jsonStructureString = "string"
	jsonStructureEmpty  = "empty"
)

func jsonStructure(json string) string {
	json = strings.TrimSpace(json)
	if len(json) == 0 {
		return jsonStructureEmpty
	}

	var res string
	switch json[0] {
	case '{':
		res = jsonStructureMap
	case '[':
		res = jsonStructureArray
	default:
		res = jsonStructureString
	}

	return res
}

//

func SetMapProperty(targetMap map[string]interface{}, key string, source interface{}) (map[string]interface{}, error) {
	destPath := strings.Split(key, ".")
	if len(destPath) == 1 {
		targetMap[key] = source
		return targetMap, nil
	}

	runningMap := targetMap
	for i := 0; i < len(destPath)-1; i++ {
		if elem, ok := runningMap[destPath[i]]; ok {
			if mapElem, ok := elem.(map[string]interface{}); ok {
				runningMap = mapElem
			} else {
				return targetMap, fmt.Errorf("key refereneces an existing property that is not a map")
			}
		} else {
			newMap := make(map[string]interface{})
			runningMap[destPath[i]] = newMap
			runningMap = newMap
		}
	}

	runningMap[destPath[len(destPath)-1]] = source
	return targetMap, nil
}
