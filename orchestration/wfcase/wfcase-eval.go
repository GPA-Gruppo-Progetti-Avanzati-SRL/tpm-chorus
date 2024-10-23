package wfcase

import (
	"fmt"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/constants"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/config"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/wfcase/wfexpressions"
	varResolver "github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util/vars"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-http-archive/har"
	"github.com/PaesslerAG/gval"
	"github.com/rs/zerolog/log"
	"strings"
)

func (wfc *WfCase) GetEvaluatorByHarEntryReference(resolverContext HarEntryReference, withVars bool, withTransformationId string, ignoreNonApplicationJsonResponseContent bool) (*wfexpressions.Evaluator, error) {
	const semLogContext = "wfcase::get-evaluator-by-har-entry-reference"
	var resolver *wfexpressions.Evaluator
	if wfc.ExpressionEvaluator != nil && wfc.ExpressionEvaluator.Name == resolverContext.String() {
		log.Trace().Str("name", resolverContext.Name).Msg(semLogContext + " resolver cached")
		return wfc.ExpressionEvaluator, nil
	}

	var err error
	if entry, ok := wfc.Entries[resolverContext.Name]; ok {
		if resolverContext.UseResponse {
			resolver, err = wfc.getEvaluatorForHarEntryResponse(resolverContext.String(), entry, withVars, withTransformationId, ignoreNonApplicationJsonResponseContent)
		} else {
			resolver, err = wfc.getEvaluatorForHarEntryRequest(resolverContext.String(), entry, withVars, withTransformationId)
		}
	} else {
		return nil, fmt.Errorf("cannot find ctxName %s in case", resolverContext.Name)
	}

	log.Trace().Str("name", resolverContext.Name).Msg(semLogContext + " new resolver created")
	wfc.ExpressionEvaluator = resolver
	return wfc.ExpressionEvaluator, err
}

func (wfc *WfCase) getEvaluatorForHarEntryRequest(evalName string, endpointData *har.Entry, withVars bool, withTransformationId string) (*wfexpressions.Evaluator, error) {

	var err error
	var resolver *wfexpressions.Evaluator

	opts := []wfexpressions.EvaluatorOption{wfexpressions.WithHeaders(endpointData.Request.Headers)}
	if endpointData.Request.PostData != nil {
		opts = append(opts, wfexpressions.WithBody(endpointData.Request.PostData.MimeType, endpointData.Request.PostData.Data, withTransformationId), wfexpressions.WithParams(endpointData.Request.PostData.Params))
	}

	if withVars {
		opts = append(opts, wfexpressions.WithProcessVars(wfc.Vars))
	}
	resolver, err = wfexpressions.NewEvaluator(evalName, opts...)
	if err != nil {
		return nil, err
	}

	return resolver, nil
}

func (wfc *WfCase) getEvaluatorForHarEntryResponse(evalName string, endpointData *har.Entry, withVars bool, withTransformationId string, ignoreNonApplicationJsonContent bool) (*wfexpressions.Evaluator, error) {

	var err error
	var resolver *wfexpressions.Evaluator

	opts := []wfexpressions.EvaluatorOption{wfexpressions.WithHeaders(endpointData.Response.Headers)}
	if endpointData.Response.Content != nil && len(endpointData.Response.Content.Data) > 0 {
		// This condition should not consider the body if is not application json and the ignore flag has been set to true
		if strings.HasPrefix(endpointData.Response.Content.MimeType, constants.ContentTypeApplicationJson) || !ignoreNonApplicationJsonContent {
			opts = append(opts, wfexpressions.WithBody(endpointData.Response.Content.MimeType, endpointData.Response.Content.Data, withTransformationId))
		} else {
			log.Debug().Str("content-type", endpointData.Response.Content.MimeType).Msg("ignoring body")
		}
	}
	if withVars {
		opts = append(opts, wfexpressions.WithProcessVars(wfc.Vars))
	}
	resolver, err = wfexpressions.NewEvaluator(evalName, opts...)
	if err != nil {
		return nil, err
	}

	return resolver, nil
}

func (wfc *WfCase) BooleanEvalProcessVars(varExpressions []string, policy string) (int, error) {
	if policy == config.AtLeastOne {
		return wfc.Vars.IndexOfFirstTrueExpression(varExpressions)
	}

	return wfc.Vars.IndexOfTheOnlyOneTrueExpression(varExpressions)
}

func (wfc *WfCase) EvalExpression(varExpression string) bool {
	_, err := wfc.Vars.IndexOfTheOnlyOneTrueExpression([]string{varExpression})
	return err == nil
}

func (wfc *WfCase) ResolveStrings(resolverContext HarEntryReference, expr []string, transformationId string, ignoreNonApplicationJsonResponseContent bool) ([]string, error) {

	resolver, err := wfc.GetEvaluatorByHarEntryReference(resolverContext, true, transformationId, ignoreNonApplicationJsonResponseContent)
	if err != nil {
		return nil, err
	}

	var resolved []string
	for _, s := range expr {
		val, _, err := varResolver.ResolveVariables(s, varResolver.SimpleVariableReference, resolver.VarResolverFunc, true)
		if err != nil {
			return nil, err
		}

		resolved = append(resolved, val)
	}

	return resolved, nil
}

func (wfc *WfCase) SetVars(resolverContext HarEntryReference, vars []config.ProcessVar, transformationId string, ignoreNonApplicationJsonResponseContent bool) error {
	return wfc.SetVarsFromCase(nil, resolverContext, vars, transformationId, ignoreNonApplicationJsonResponseContent)
}

func (wfc *WfCase) SetVarsFromCase(sourceWfc *WfCase, resolverContext HarEntryReference, vars []config.ProcessVar, transformationId string, ignoreNonApplicationJsonResponseContent bool) error {
	const semLogContext = "wf-case::set-vars-from-case"
	if len(vars) == 0 {
		return nil
	}

	if sourceWfc == nil {
		sourceWfc = wfc
	}

	resolver, err := sourceWfc.GetEvaluatorByHarEntryReference(resolverContext, true, transformationId, ignoreNonApplicationJsonResponseContent)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return err
	}

	for _, v := range vars {
		boolGuard := true
		if v.Guard != "" {
			boolGuard, err = sourceWfc.Vars.EvalToBool(v.Guard)
		}

		if boolGuard && err == nil {

			resolvedName := v.Name
			if v.GlobalScope {
				resolvedName, err = resolver.InterpolateAndEvalToString(v.Name)
				if err != nil {
					log.Error().Err(err).Msg(semLogContext)
					return err
				}
			}

			val, _, err := varResolver.ResolveVariables(v.Value, varResolver.SimpleVariableReference, resolver.VarResolverFunc, true)
			if err != nil {
				log.Error().Err(err).Msg(semLogContext)
				return err
			}

			val, isExpr := IsExpression(val)

			// Was isExpression(val) but in doing this I use the evaluated value and I depend on the value of the variables  with potentially weird values.
			var varValue interface{} = val
			if isExpr && val != "" {
				varValue, err = gval.Evaluate(val, sourceWfc.Vars)
				if err != nil {
					log.Error().Err(err).Msg(semLogContext)
					return err
				}
			}

			err = wfc.Vars.Set(resolvedName, varValue, v.GlobalScope, v.Ttl)
			if err != nil {
				log.Error().Err(err).Msg(semLogContext)
				return err
			}
		}

		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return err
		}
	}

	resolver.ClearTempVariables()
	return nil
}
