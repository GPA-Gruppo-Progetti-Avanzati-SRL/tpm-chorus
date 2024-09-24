package constants

const (
	ContentTypeApplicationJson = "application/json"
	ContentTypeTextPlain       = "text/plain"

	ContentTypeHeader = "Content-Type"

	DebugMode = true

	SemLogOpenApi          = "open-api"
	SemLogOrchestrationSid = "sid"
	SemLogPath             = "path"
	SemLogFile             = "file"
	SemLogType             = "type"
	SemLogName             = "name"
	SemLogMethod           = "method"

	SemLogActivity           = "act"
	SemLogNextActivity       = "next-act"
	SemLogActivityName       = "name"
	SemLogActivityNumOutputs = "len-outputs"
	SemLogActivityNumInputs  = "len-inputs"
	SemLogCacheKey           = "cache-key"

	VERSION string = "0.0.1-SNAPSHOT"
)
