package kafkasink

import (
	"context"
	"errors"
	"fmt"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/constants"
	orchestrationConfig "github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/config"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/config/repo"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/wfcase"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/wfcase/wfexpressions"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/pipeline/plconfig/sinkconfig"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/pipeline/plexecutable"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util"
	varResolver "github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util/vars"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-http-archive/har"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-kafka-common/tprod"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
	"net/http"
	"strings"
	"time"
)

type KafkaSinkStage struct {
	sinkconfig.SinkStageDefinitionReference
	Definition sinkconfig.KafkaSinkDefinition
	AssetGroup repo.AssetGroup

	mq *KafkaSinkStageBufferedMessageQueue
	//mqBuffered *kproducer.KafkaSinkStageBufferedMessageQueue
	//mqDirect   *kproducer.KafkaSinkStageDirectMessageQueue
}

func NewKafkaSinkStage(pipelineWorkMode string, ref sinkconfig.SinkStageDefinitionReference, g repo.AssetGroup) (*KafkaSinkStage, error) {
	const semLogContext = "kafka-sink-stage::new"
	const semLogStageId = "stage-id"
	var def sinkconfig.KafkaSinkDefinition
	err := yaml.Unmarshal(ref.Data, &def)
	if err != nil {
		log.Error().Err(err).Str(semLogStageId, ref.Id()).Str("type", ref.Typ).Msg(semLogContext)
		return nil, err
	}

	var sinkStage *KafkaSinkStage
	sinkStage = &KafkaSinkStage{
		SinkStageDefinitionReference: ref,
		Definition:                   def,
		AssetGroup:                   g,
	}

	if sinkStage.Definition.Event.Body.ExternalValue != "" {
		if sinkStage.AssetGroup.AssetIndexByPath(def.Event.Body.ExternalValue) < 0 {
			return nil, fmt.Errorf(semLogContext+" cannot find external value (%s) for stage %s", sinkStage.Definition.Event.Body.ExternalValue, ref.StageId)
		}

		b, err := g.ReadRefsData(def.Event.Body.ExternalValue)
		if err != nil {
			log.Error().Err(err).Str(semLogStageId, ref.Id()).Str("type", ref.Typ).Msg(semLogContext)
			return nil, err
		}

		sinkStage.Definition.Event.Body.Data = b
	}

	err = sinkStage.newMessageQueue(pipelineWorkMode)
	if err != nil {
		return sinkStage, err
	}
	return sinkStage, nil
}

func (stage *KafkaSinkStage) newMessageQueue(pipelineWorkMode string) error {

	const semLogContext = "kafka-sink-stage::new-queue"
	const semLogStageId = "stage-id"
	kp, err := tprod.NewKafkaProducerWrapper(context.Background(), stage.Definition.BrokerName, "", kafka.TopicPartition{Partition: kafka.PartitionAny}, stage.Definition.WithSynchDelivery(), false)
	if err != nil {
		log.Error().Err(err).Str(semLogStageId, stage.SinkStageDefinitionReference.Id()).Msg(semLogContext)
		return err
	}

	bufSize := stage.Definition.MessageProducerBufferSize
	if bufSize == 0 {
		bufSize = util.IntCoalesce(bufSize, 1000)
	}

	log.Warn().Int("buf-size", bufSize).Msg(semLogContext)
	stage.mq, _ = NewKafkaSinkStageBufferedMessageQueue(pipelineWorkMode, stage.Definition.BrokerName, kp, bufSize, stage.Definition.FlushTimeout, stage.Definition.RefMetrics.GId, stage.Definition.WithRandomError)

	if !stage.Definition.WithSynchDelivery() {
		log.Info().Str(semLogStageId, stage.SinkStageDefinitionReference.Id()).Msg(semLogContext + " - synch delivery off - starting montiro producer events")
		if stage.mq != nil {
			go stage.mq.MonitorProducerEvents(stage.Definition.BrokerName)
		}
	} else {
		log.Info().Str(semLogStageId, stage.SinkStageDefinitionReference.Id()).Msg(semLogContext + " - synch delivery on")
	}

	return nil
}

func (stage *KafkaSinkStage) Reset() error {
	const semLogContext = "kafka-sink-stage::reset"
	const semLogStageId = "stage-id"
	log.Info().Msg(semLogContext)
	stage.Close()

	// Recreate the producer.
	kp, err := tprod.NewKafkaProducerWrapper(context.Background(), stage.Definition.BrokerName, "", kafka.TopicPartition{Partition: kafka.PartitionAny}, stage.Definition.WithSynchDelivery(), false)
	if err != nil {
		log.Error().Err(err).Str(semLogStageId, stage.SinkStageDefinitionReference.Id()).Msg(semLogContext)
		return err
	}

	stage.mq.SetKafkaProducerWrapper(kp)

	// Restart events monitoring
	if !stage.Definition.WithSynchDelivery() {
		log.Info().Str(semLogStageId, stage.SinkStageDefinitionReference.Id()).Msg(semLogContext + " - synch delivery off - starting montiro producer events")
		if stage.mq != nil {
			go stage.mq.MonitorProducerEvents(stage.Definition.BrokerName)
		}
	} else {
		log.Info().Str(semLogStageId, stage.SinkStageDefinitionReference.Id()).Msg(semLogContext + " - synch delivery on")
	}

	return nil
}

func (stage *KafkaSinkStage) Close() {
	const semLogContext = "kafka-sink-stage::close"
	log.Info().Msg(semLogContext)
	if stage.mq != nil {
		err := stage.mq.Close()
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
		}
	}

	// Should consider a sleep after closing the producer and the monitor to exit....
	// may be....
	time.Sleep(100 * time.Millisecond)
}

func (stage *KafkaSinkStage) Flush() (int, error) {
	const semLogContext = "kafka-sink-stage::flush"

	return stage.mq.Flush( /*stage.Definition.WaitConfigOnNotCompleted()*/ )
	/*
		var sz int
		var err error
		if stage.mqBuffered != nil {
			sz := len(stage.mqBuffered.Items)
			err := stage.mqBuffered.Flush()
			if err != nil {
				log.Error().Err(err).Msg(semLogContext)
				return sz, err
			}

			stats := stage.mqBuffered.Stats()
			waitCfg := stage.Definition.WaitConfigOnNotCompleted()
			if !waitCfg.IsZero() && !stats.IsFail() && !stats.IsComplete() {
				log.Info().Float64("wait", waitCfg.Wait.Seconds()).Float64("first-wait", waitCfg.FirstWait.Seconds()).Int("max-iterations", waitCfg.MaxWaitTimes).Msg(semLogContext + " waiting for sink completion")
				if waitCfg.FirstWait > 0 {
					time.Sleep(waitCfg.FirstWait)
				}
				i := 0
				for !stats.IsFail() && !stats.IsComplete() && i < waitCfg.MaxWaitTimes {
					log.Info().Float64("wait", waitCfg.Wait.Seconds()).Int("iteration", i).Msg(semLogContext + " waiting for sink completion")
					time.Sleep(waitCfg.Wait)
					stats = stage.mqBuffered.Stats()
					i++
				}
			}

			err = stats.IsErr(semLogContext, true)
		}

		if stage.mqDirect != nil {
			err := stage.mqDirect.Flush()
			if err != nil {
				log.Error().Err(err).Msg(semLogContext)
				return 0, err
			}

			stats := stage.mqDirect.Stats()
			waitCfg := stage.Definition.WaitConfigOnNotCompleted()
			if !waitCfg.IsZero() && !stats.IsFail() && !stats.IsComplete() {
				log.Info().Float64("wait", waitCfg.Wait.Seconds()).Float64("first-wait", waitCfg.FirstWait.Seconds()).Int("max-iterations", waitCfg.MaxWaitTimes).Msg(semLogContext + " waiting for sink completion")
				if waitCfg.FirstWait > 0 {
					time.Sleep(waitCfg.FirstWait)
				}
				i := 0
				for !stats.IsFail() && !stats.IsComplete() && i < waitCfg.MaxWaitTimes {
					log.Info().Float64("wait", waitCfg.Wait.Seconds()).Int("iteration", i).Msg(semLogContext + " waiting for sink completion")
					time.Sleep(waitCfg.Wait)
					stats = stage.mqDirect.Stats()
					i++
				}
			}

			err = stats.IsErr(semLogContext, true)
		}

		return sz, err
	*/
}

func (stage *KafkaSinkStage) Clear() int {
	const semLogContext = "kafka-sink-stage-queue::clear"
	stage.mq.Clear()
	return 0
}

func (stage *KafkaSinkStage) Sink(evt *plexecutable.PipelineEvent) error {

	const semLogContext = "kafka-sink-stage::sink"
	const semLogSinkStage = "sink-stage"
	var err error

	pos := evt.BatchPosition
	part := evt.BatchPartition
	rt := evt.ResumeToken
	wfc := evt.WfCase

	log.Info().Str(semLogSinkStage, stage.StageId).Msg(semLogContext)

	stats := stage.mq.statsInfo

	err = stats.IsErr(false)
	if err != nil {
		return err
	}

	req, msgKey, msgBody, err := stage.newRequestDefinition(wfc)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return err
	}

	_ = wfc.SetHarEntryRequest(stage.StageId, req, orchestrationConfig.PersonallyIdentifiableInformation{})

	resp, err := stage.produce(pos, part, rt, wfc, req, msgKey, msgBody)
	_ = wfc.SetHarEntryResponse(stage.StageId, resp, orchestrationConfig.PersonallyIdentifiableInformation{})
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return err
	}

	//var msgHeaders map[string]string
	//var eventKey string
	//for _, h := range req.Headers {
	//	if h.Name == MessageKeyHeaderName {
	//		eventKey = h.Value
	//	} else {
	//		if msgHeaders == nil {
	//			msgHeaders = make(map[string]string)
	//		}
	//		msgHeaders[h.Name] = h.Value
	//	}
	//}
	//
	//msg := tprod.Message{
	//	HarSpan: nil,
	//	Span:    nil,
	//	ToTopic: tprod.TargetTopic{
	//		Id: stage.Definition.TopicName,
	//	},
	//	Headers:         msgHeaders,
	//	Key:             []byte(eventKey),
	//	Body:            req.PostData.Data,
	//	MessageProducer: nil,
	//}

	return nil
}

func (stage *KafkaSinkStage) newRequestDefinition(wfc *wfcase.WfCase) (*har.Request, []byte, []byte, error) {

	const semLogContext = "kafka-activity::new-request-definition"

	/*
		expressionCtx, err := wfc.ResolveExpressionContextName(config.InitialRequestContextNameStringReference)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return nil, nil, nil, err
		}

		// Sinks are different from activities in that they execute after the end of the orchestration.
		// The resolution provided by ResolveExpressionContextName is wrong: if request is provided as value the context is the request part of the request and this is not the case.
		// if another reference is provided it is always the response got back. So the need to force the useResponse in any case...
		expressionCtx.UseResponse = true

		// note the ignoreNonApplicationJsonResponseContent has been set to false since it doesn't apply to the request processing
		resolver, err := wfc.GetResolverByContext(expressionCtx, true, "", false)
	*/
	resolver, err := stage.GetSinkStagesResolver(wfc)
	if err != nil {
		return nil, nil, nil, err
	}

	var opts []har.RequestOption

	ub := har.UrlBuilder{}
	ub.WithScheme("kafka")

	ub.WithHostname(fmt.Sprintf("%s.%s", stage.Definition.BrokerName, "tpm-kyrie"))

	s, _, err := varResolver.ResolveVariables(stage.Definition.TopicName, varResolver.SimpleVariableReference, resolver.VarResolverFunc, true)
	if err != nil {
		return nil, nil, nil, err
	}
	ub.WithPath(fmt.Sprintf("/%s/topics/%s", stage.StageId, s))

	opts = append(opts, har.WithMethod(http.MethodPost))
	opts = append(opts, har.WithUrl(ub.Url()))

	for _, h := range stage.Definition.Event.Headers {
		r, _, err := varResolver.ResolveVariables(h.Value, varResolver.SimpleVariableReference, resolver.VarResolverFunc, true)
		if err != nil {
			return nil, nil, nil, err
		}
		opts = append(opts, har.WithHeader(har.NameValuePair{Name: h.Name, Value: r}))
	}

	var msgBody, msgKey []byte
	if !stage.Definition.Event.Body.IsZero() {
		msgBody, err = stage.newRequestDefinitionMessageBody(wfc, resolver)
		if err != nil {
			return nil, nil, nil, err
		}
		// opts = append(opts, har.WithBody(msgBody))
	} else {
		err = errors.New(" body of activity cannot be empty")
		log.Error().Err(err).Msg(semLogContext)
		return nil, nil, nil, err
	}

	msgKey, err = stage.newRequestDefinitionMessageKey(wfc, resolver)
	if err != nil {
		return nil, nil, nil, err
	}
	// opts = append(opts, har.WithHeader(har.NameValuePair{Name: MessageKeyHeaderName, Value: string(msgKey)}))

	var composedBody = stage.compose(msgKey, msgBody)
	opts = append(opts, har.WithBody(composedBody))

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

	return &req, msgKey, msgBody, nil
}

func (stage *KafkaSinkStage) compose(key []byte, body []byte) []byte {

	var sb strings.Builder
	numberOfElements := 0
	sb.WriteString("{")
	if len(key) > 0 {
		numberOfElements++
		sb.WriteString(fmt.Sprintf("\"%s\": ", "key"))
		sb.WriteString(string(key))
	}
	if len(body) > 0 {
		if numberOfElements > 0 {
			sb.WriteString(",")
		}
		numberOfElements++
		sb.WriteString(fmt.Sprintf("\"%s\": ", "body"))
		sb.WriteString(string(body))
	}

	sb.WriteString("}")
	return []byte(sb.String())

}
func (stage *KafkaSinkStage) newRequestDefinitionMessageBody(wfc *wfcase.WfCase, resolver *wfexpressions.Evaluator) ([]byte, error) {
	const semLogContext = "kafka-sink-stage::message-body"
	var bodyContent []byte
	if stage.Definition.Event.Body.Data != nil {
		bodyContent = stage.Definition.Event.Body.Data
	} else {
		err := errors.New("event template has not been loaded")
		log.Warn().Err(err).Msg(semLogContext)
		bodyContent, _ = stage.AssetGroup.ReadRefsData(stage.Definition.Event.Body.ExternalValue)
	}
	s, _, err := varResolver.ResolveVariables(string(bodyContent), varResolver.SimpleVariableReference, resolver.VarResolverFunc, true)
	if err != nil {
		return nil, err
	}

	if stage.Definition.Event.Body.Typ == "simple" {
		return []byte(s), nil
	}

	b, err := wfc.ProcessTemplate(s)
	if err != nil {
		return nil, err
	}

	return b, nil
}

func (stage *KafkaSinkStage) newRequestDefinitionMessageKey(wfc *wfcase.WfCase, resolver *wfexpressions.Evaluator) ([]byte, error) {

	const semLogContext = "kafka-sink-stage::new-message-key"

	var s string
	var err error
	if stage.Definition.Event.Key != "" {
		// messageKey, _ := a.Refs.Find(ep.Definition.Key)
		messageKey := stage.Definition.Event.Key
		s, _, err = varResolver.ResolveVariables(string(messageKey), varResolver.SimpleVariableReference, resolver.VarResolverFunc, true)
		if err != nil {
			return nil, err
		}
	} else {
		s = "kafka-sink-stage"
		log.Warn().Msgf("message key not specified... defaulting to '%s'", s)
	}

	return []byte(s), nil
}

func (stage *KafkaSinkStage) produce(pos, part int, rt string, wfc *wfcase.WfCase, reqDef *har.Request, msgKey, msgBody []byte) (*har.Response, error) {

	const semLogContext = "kafka-sink-stage::produce"
	const semLogContextStatusCode = "status-code"
	/*
		var msgKey string
		var msgHeaders map[string]string
		if len(reqDef.Headers) > 0 {
			msgHeaders = make(map[string]string)
			for _, h := range reqDef.Headers {
				msgHeaders[h.Name] = h.Value
				if h.Name == MessageKeyHeaderName {
					msgKey = h.Value
				}
			}
		}
	*/

	var msgHeaders map[string]string
	for _, h := range reqDef.Headers {
		if msgHeaders == nil {
			msgHeaders = make(map[string]string)
		}
		msgHeaders[h.Name] = h.Value
	}

	msg := QueueMessage{
		BatchPosition:  pos,
		BatchPartition: part,
		Rt:             rt,
		Span:           nil,
		ToTopic:        stage.Definition.TopicName,
		Headers:        msgHeaders,
		Key:            msgKey,
		Body:           msgBody,
	}

	resp := []byte("kafka sink message accepted")
	sc := http.StatusAccepted

	var err error
	err = stage.mq.Produce(msg)

	if err != nil {
		resp = []byte(err.Error())
		sc = http.StatusInternalServerError
	}

	responseHeaders := []har.NameValuePair{{Name: "Content-Type", Value: "text/plain"}, {Name: "Content-Length", Value: fmt.Sprint(len(resp))}}
	r := &har.Response{
		Status:      sc,
		HTTPVersion: "1.1",
		StatusText:  http.StatusText(sc),
		Headers:     responseHeaders,
		HeadersSize: -1,
		BodySize:    int64(len(resp)),
		Cookies:     []har.Cookie{},
		Content: &har.Content{
			MimeType: constants.ContentTypeApplicationJson,
			Size:     int64(len(resp)),
			Data:     resp,
		},
	}

	log.Trace().Int(semLogContextStatusCode, r.Status).Int("num-headers", len(r.Headers)).Int64("content-length", r.BodySize).Msg(semLogContext + " message produced")
	return r, err
}
