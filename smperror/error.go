package smperror

import (
	"encoding/json"
	"errors"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util/templateutil"
	"github.com/rs/zerolog/log"
	"net/http"
	"text/template"
	"time"
)

const (
	ErrorDefaultMessage       = ""
	ErrorDefaultAmbit         = ""
	ServerErrorDefaultAmbit   = "system"
	ServerErrorDefaultMessage = "Server error"
	BadRequestDefaultAmbit    = "validation"
	BadRequestDefaultMessage  = "Validation error"
)

type Option func(executableError *SymphonyError)

const (
	OnErrorLevelFatal  = "fatal"
	OnErrorLevelSystem = "system"
	OnErrorLevelError  = "error"
)

type SymphonyError struct {
	StatusCode  int    `yaml:"-" mapstructure:"-" json:"-"`
	Ambit       string `yaml:"ambit,omitempty" mapstructure:"ambit,omitempty" json:"ambit,omitempty"`
	Step        string `yaml:"step,omitempty" mapstructure:"step,omitempty" json:"step,omitempty"`
	ErrCode     string `yaml:"code,omitempty" mapstructure:"code,omitempty" json:"code,omitempty"`
	Message     string `yaml:"message,omitempty" mapstructure:"message,omitempty" json:"message,omitempty"`
	Description string `yaml:"description,omitempty" mapstructure:"description,omitempty" json:"description,omitempty"`
	Ts          string `yaml:"timestamp,omitempty" mapstructure:"timestamp,omitempty" json:"timestamp,omitempty"`
	Level       string `yaml:"-" mapstructure:"-" json:"-"`
}

func WithErrorStatusCode(c int) Option {
	return func(e *SymphonyError) {
		e.StatusCode = c
	}
}

func WithErrorAmbit(a string) Option {
	return func(e *SymphonyError) {
		e.Ambit = a
	}
}

func WithCode(c string) Option {
	return func(e *SymphonyError) {
		e.ErrCode = c
	}
}

func WithStep(s string) Option {
	return func(e *SymphonyError) {
		e.Step = s
	}
}

func WithErrorMessage(m string) Option {
	return func(e *SymphonyError) {
		e.Message = m
	}
}

func WithError(err error) Option {
	return func(e *SymphonyError) {
		var smpErr *SymphonyError
		if errors.As(err, &smpErr) {
			e.StatusCode = smpErr.StatusCode
			e.ErrCode = smpErr.ErrCode
			e.Ambit = smpErr.Ambit
			e.Step = smpErr.Step
			e.Message = smpErr.Message
			e.Description = smpErr.Description
		}
	}
}

func WithDescription(m string) Option {
	return func(e *SymphonyError) {
		e.Description = m
	}
}

func WithLevel(m string) Option {
	return func(e *SymphonyError) {
		if m == "" || (m != OnErrorLevelFatal && m != OnErrorLevelSystem && m != OnErrorLevelError) {
			m = OnErrorLevelFatal
		}

		e.Level = m
	}
}

func NewExecutableError(opts ...Option) *SymphonyError {
	err := &SymphonyError{StatusCode: 0, Ambit: ErrorDefaultAmbit, Message: ErrorDefaultMessage}
	for _, o := range opts {
		o(err)
	}
	return err
}

func NewExecutableServerError(opts ...Option) *SymphonyError {
	err := &SymphonyError{StatusCode: http.StatusInternalServerError, Ambit: ServerErrorDefaultAmbit, Message: ServerErrorDefaultMessage}
	for _, o := range opts {
		o(err)
	}
	return err
}

func NewBadRequestError(opts ...Option) *SymphonyError {
	err := &SymphonyError{StatusCode: http.StatusBadRequest, Ambit: BadRequestDefaultAmbit, Message: BadRequestDefaultMessage}
	for _, o := range opts {
		o(err)
	}
	return err
}

func (exe *SymphonyError) Error() string {
	b, err := json.Marshal(exe)
	if err != nil {
		log.Err(err).Str("message", exe.Message).Int("status-code", exe.StatusCode).Msg("error in marshalling executable error")
		return err.Error()
	}

	return string(b)
}

func (exe SymphonyError) ToJSON(withTemplate []byte) ([]byte, error) {

	exe.Ts = time.Now().Format(time.RFC3339)
	var jsn []byte
	var err error
	if len(withTemplate) > 0 {
		m := map[string]interface{}{
			"ambit":       util.JSONEscape(exe.Ambit, false),
			"message":     util.JSONEscape(exe.Message, false),
			"code":        util.JSONEscape(exe.ErrCode, false),
			"description": util.JSONEscape(exe.Description, false),
			"step":        util.JSONEscape(exe.Step, false),
			"ts":          exe.Ts,
		}

		var t *template.Template
		t, err = templateutil.Parse([]templateutil.Info{{Name: "example-resp", Content: string(withTemplate)}}, nil)
		if err == nil {
			jsn, err = templateutil.Process(t, m, false)
			if err == nil {
				return jsn, nil
			}
		}
	}

	if len(jsn) == 0 {
		jsn, err = json.Marshal(exe)
		if err != nil {
			return nil, err
		}
	}

	return jsn, nil
}
