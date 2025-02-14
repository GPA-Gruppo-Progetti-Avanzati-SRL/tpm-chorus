package rhpcfg

import (
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/wfcase"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/wfcase/wfexpressions"
	"github.com/rs/zerolog/log"
)

type SinkStageDefinitionReference struct {
	StageId        string `json:"id,omitempty" yaml:"id,omitempty" mapstructure:"id,omitempty"`
	Typ            string `mapstructure:"type,omitempty" yaml:"type,omitempty" json:"type,omitempty"`
	OpType         string `yaml:"op-type,omitempty" mapstructure:"op-type,omitempty" json:"op-type,omitempty"`
	Description    string `mapstructure:"description,omitempty" yaml:"description,omitempty" json:"description,omitempty"`
	DefinitionFile string `mapstructure:"definition-file,omitempty" yaml:"definition-file,omitempty" json:"definition-file,omitempty"`
	Data           []byte `mapstructure:"-" yaml:"-" json:"-"`
}

func (k *SinkStageDefinitionReference) Id() string {
	return k.StageId
}

func (k *SinkStageDefinitionReference) Type() string {
	return k.Typ
}

func (k *SinkStageDefinitionReference) GetSinkStagesResolver(wfc *wfcase.WfCase) (*wfexpressions.Evaluator, error) {
	const semLogContext = "sink-stage-definition-reference::get-sink-stage-resolver"
	expressionCtx, err := wfc.ResolveHarEntryReferenceByName(wfcase.InitialRequestHarEntryId)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return nil, err
	}

	// Sinks are different from activities in that they execute after the end of the orchestration.
	// The resolution provided by ResolveExpressionContextName is wrong: if request is provided as value the context is the request part of the request and this is not the case.
	// if another reference is provided it is always the response got back. So the need to force the useResponse in any case...
	expressionCtx.UseResponse = true

	resolver, err := wfc.GetEvaluatorByHarEntryReference(expressionCtx, true, "", false)
	if err != nil {
		return nil, err
	}

	return resolver, nil
}

type SinkStageDefinitionReferences []SinkStageDefinitionReference
