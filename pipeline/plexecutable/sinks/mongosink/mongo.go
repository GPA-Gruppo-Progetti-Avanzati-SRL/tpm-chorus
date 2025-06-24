package mongosink

import (
	"context"
	"errors"
	"fmt"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/constants"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/config"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/config/repo"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/wfcase"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/pipeline/plconfig/sinkconfig"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/pipeline/plexecutable"
	varResolver "github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util/vars"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-http-archive/har"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-mongo-common/jsonops"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-mongo-common/mongolks"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gopkg.in/yaml.v3"
	"net/http"
	"time"
)

var opTypes = map[jsonops.MongoJsonOperationType]struct{}{
	jsonops.ReplaceOneOperationType: struct{}{},
	jsonops.UpdateOneOperationType:  struct{}{},
	jsonops.DeleteOneOperationType:  struct{}{},
	jsonops.InsertOneOperationType:  struct{}{},
}

func UnmarshalMongoSinkStageDefinition(ref sinkconfig.SinkStageDefinitionReference, refs repo.AssetGroup) (sinkconfig.MongoSinkDefinition, error) {
	const semLogContext = "mongo-sink-definition::unmarshal"

	var err error

	maDef := sinkconfig.MongoSinkDefinition{OpType: jsonops.MongoJsonOperationType(ref.OpType)}
	if _, ok := opTypes[jsonops.MongoJsonOperationType(ref.OpType)]; !ok {
		err = errors.New("unsupported op-type")
		log.Error().Err(err).Str("op-type", ref.OpType).Msg(semLogContext)
		return maDef, err
	}

	err = yaml.Unmarshal(ref.Data, &maDef)
	if err != nil {
		return maDef, err
	}

	maDef.StatementParts, err = maDef.LoadStatement(refs)
	if err != nil {
		return maDef, err
	}

	return maDef, nil
}

type MongoSinkStage struct {
	sinkconfig.SinkStageDefinitionReference
	Definition sinkconfig.MongoSinkDefinition `yaml:"-" json:"-" mapstructure:"-"`
	AssetGroup repo.AssetGroup                `yaml:"-" json:"-" mapstructure:"-"`
	batch      []mongo.WriteModel
	statsInfo  *StatsInfo
}

func NewMongoSinkStage(pipelineId string, ref sinkconfig.SinkStageDefinitionReference, g repo.AssetGroup) (*MongoSinkStage, error) {
	const semLogContext = "mongo-sink-stage::new"
	const semLogStageId = "stage-id"
	def, err := UnmarshalMongoSinkStageDefinition(ref, g)
	if err != nil {
		log.Error().Err(err).Str("op-type", ref.OpType).Str(semLogStageId, ref.Id()).Str("type", ref.Typ).Msg(semLogContext)
		return nil, err
	}

	if !def.BulkWriteOrdered() {
		log.Warn().Str("ordered-write", def.OrderedBlkWrite).Msg(semLogContext + " - ordered options set to false")
	}

	var sinkStage *MongoSinkStage
	sinkStage = &MongoSinkStage{
		SinkStageDefinitionReference: ref,
		Definition:                   def,
		AssetGroup:                   g,
		statsInfo:                    NewStatsInfo(pipelineId, ref.StageId, "", def.MetricsConfigGroupId()),
	}

	return sinkStage, nil
}

func (s *MongoSinkStage) IncDiscardedMessage(n int64) {
	s.statsInfo.IncDiscarded(n)
}

func (a *MongoSinkStage) resolveStatementParts(wfc *wfcase.WfCase, m map[jsonops.MongoJsonOperationStatementPart][]byte) (map[jsonops.MongoJsonOperationStatementPart][]byte, error) {
	const semLogContext = "mongo-sink-definition::resolve-statement-parts"
	/*
		expressionCtx, err := wfc.ResolveExpressionContextName(config.InitialRequestContextNameStringReference)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return nil, err
		}

		// Sinks are different from activities in that they execute after the end of the orchestration.
		// The resolution provided by ResolveExpressionContextName is wrong: if request is provided as value the context is the request part of the request and this is not the case.
		// if another reference is provided it is always the response got back. So the need to force the useResponse in any case...
		expressionCtx.UseResponse = true

		resolver, err := wfc.GetResolverByContext(expressionCtx, true, "", false)
	*/
	resolver, err := a.GetSinkStagesResolver(wfc)
	if err != nil {
		return nil, err
	}

	newMap := map[jsonops.MongoJsonOperationStatementPart][]byte{}
	for n, b := range m {
		s, _, err := varResolver.ResolveVariables(string(b), varResolver.SimpleVariableReference, resolver.VarResolverFunc, true)
		if err != nil {
			return nil, err
		}

		b1, err := wfc.ProcessTemplate(s)
		if err != nil {
			return nil, err
		}

		newMap[n] = b1
	}

	return newMap, nil
}

func (stage *MongoSinkStage) Flush() (int, error) {
	const semLogContext = "mongo-sink-stage::flush"
	log.Info().Int("num-statements", len(stage.batch)).Msg(semLogContext)
	defer func() { stage.batch = nil }()

	ns := len(stage.batch)
	if ns > 0 {
		beginOf := time.Now()

		stage.statsInfo.SetBulkSize(ns)
		lks, err := mongolks.GetLinkedService(context.Background(), stage.Definition.LksName)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return 0, err
		}

		c := lks.GetCollection(stage.Definition.CollectionId, "")
		if c == nil {
			err = errors.New("cannot find requested collection")
			log.Error().Err(err).Str("collection", stage.Definition.CollectionId).Msg(semLogContext)
			return 0, err
		}

		blkOpts := options.BulkWriteOptions{}
		blkOpts.SetOrdered(stage.Definition.BulkWriteOrdered())
		resp, err := c.BulkWrite(context.Background(), stage.batch)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			stage.statsInfo.IncErrors(1)
			return 0, err
		}

		stage.statsInfo.Update(resp, time.Since(beginOf))
	}

	return ns, nil
}

func (stage *MongoSinkStage) Reset() error {
	const semLogContext = "mongo-sink-stage::reset"
	return nil
}

func (stage *MongoSinkStage) Close() {
	const semLogContext = "mongo-sink-stage::close"
}

func (stage *MongoSinkStage) Clear() int {
	const semLogContext = "mongo-sink-stage::clear"

	ns := len(stage.batch)
	log.Info().Int("num-statements", ns).Msg(semLogContext)
	stage.batch = nil
	return ns
}

func (stage *MongoSinkStage) Sink(evt *plexecutable.PipelineEvent) error {
	const semLogContext = "mongo-sink-stage::sink"
	var err error

	wfc := evt.WfCase

	statementConfig, err := stage.resolveStatementParts(wfc, stage.Definition.StatementParts)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return err
	}

	op, err := jsonops.NewOperation(jsonops.MongoJsonOperationType(stage.OpType), statementConfig)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return err
	}

	req, err := stage.newRequestDefinition(wfc, op)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return err
	}

	_ = wfc.SetHarEntryRequest(stage.StageId, req, config.PersonallyIdentifiableInformation{})

	resp, err := stage.produce(wfc, req, op)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return err
	}

	_ = wfc.SetHarEntryResponse(stage.StageId, resp, config.PersonallyIdentifiableInformation{})

	return nil
}

func (a *MongoSinkStage) newRequestDefinition(wfc *wfcase.WfCase, op jsonops.Operation) (*har.Request, error) {

	var opts []har.RequestOption

	ub := har.UrlBuilder{}
	ub.WithPort(27017)
	ub.WithScheme("mongodb")

	ub.WithHostname("localhost")
	ub.WithPath(fmt.Sprintf("/%s/%s/%s", a.StageId, a.Definition.CollectionId, string(a.Definition.OpType)))

	opts = append(opts, har.WithMethod("POST"))
	opts = append(opts, har.WithUrl(ub.Url()))
	opts = append(opts, har.WithBody([]byte(op.ToString())))

	req := har.Request{
		HTTPVersion: "1.1",
		Cookies:     []har.Cookie{},
		QueryString: []har.NameValuePair{},
		HeadersSize: -1,
		BodySize:    -1,
	}
	for _, o := range opts {
		o(&req)
	}

	return &req, nil
}

func (stage *MongoSinkStage) produce(wfc *wfcase.WfCase, reqDef *har.Request, op jsonops.Operation) (*har.Response, error) {

	const semLogContext = "mongo-sink-stage::produce"
	const semLogContextStatusCode = "status-code"
	wrm, err := op.NewWriteModel()
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return nil, err
	}
	stage.batch = append(stage.batch, wrm)

	b := []byte("mongo sink operation accepted")

	sc := http.StatusAccepted
	responseHeaders := []har.NameValuePair{{Name: "Content-Type", Value: "text/plain"}, {Name: "Content-Length", Value: fmt.Sprint(len(b))}}
	r := &har.Response{
		Status:      sc,
		HTTPVersion: "1.1",
		StatusText:  http.StatusText(sc),
		Headers:     responseHeaders,
		HeadersSize: -1,
		BodySize:    int64(len(b)),
		Cookies:     []har.Cookie{},
		Content: &har.Content{
			MimeType: constants.ContentTypeApplicationJson,
			Size:     int64(len(b)),
			Data:     b,
		},
	}

	log.Trace().Int(semLogContextStatusCode, r.Status).Int("num-headers", len(r.Headers)).Int64("content-length", r.BodySize).Msg(semLogContext + " message produced")

	return r, nil
}
