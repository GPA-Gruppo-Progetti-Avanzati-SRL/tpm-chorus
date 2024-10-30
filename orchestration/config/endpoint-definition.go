package config

import (
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type NameValuePair struct {
	Name  string `yaml:"name,omitempty" mapstructure:"name,omitempty" json:"name,omitempty"`
	Value string `yaml:"value,omitempty" mapstructure:"value,omitempty" json:"value,omitempty"`
	Guard string `yaml:"guard,omitempty" mapstructure:"guard,omitempty" json:"guard,omitempty"`
}

type PostData struct {
	Name          string `yaml:"name,omitempty" json:"name,omitempty" mapstructure:"name,omitempty"`
	Type          string `yaml:"type,omitempty" json:"type,omitempty" mapstructure:"type,omitempty"`
	ExternalValue string `yaml:"external-value,omitempty" json:"external-value,omitempty" mapstructure:"external-value,omitempty"`
	Value         string `yaml:"value,omitempty" json:"value,omitempty" mapstructure:"value,omitempty"`
	//Data          []byte `yaml:"data,omitempty" json:"data,omitempty" mapstructure:"data,omitempty"`
}

func (pd PostData) IsZero() bool {
	return /* len(pd.Data) == 0 && */ pd.ExternalValue == "" && pd.Value == ""
}

type HttpClientOptions struct {
	RestTimeout      time.Duration `yaml:"timeout,omitempty" json:"timeout,omitempty" mapstructure:"timeout,omitempty"`
	RetryCount       int           `mapstructure:"retry-count,omitempty" json:"retry-count,omitempty" yaml:"retry-count,omitempty"`
	RetryWaitTime    time.Duration `mapstructure:"retry-wait-time,omitempty" json:"retry-wait-time,omitempty" yaml:"retry-wait-time,omitempty"`
	RetryMaxWaitTime time.Duration `mapstructure:"retry-max-wait-time,omitempty" json:"retry-max-wait-time,omitempty" yaml:"retry-max-wait-time,omitempty"`
	RetryOnHttpError []int         `mapstructure:"retry-on-errors,omitempty" json:"retry-on-errors,omitempty" yaml:"retry-on-errors,omitempty"`
}

type EndpointDefinition struct {
	// Description       string             `yaml:"description,omitempty" mapstructure:"description,omitempty" json:"description,omitempty"`
	Method                                  string             `yaml:"method,omitempty" json:"method,omitempty" mapstructure:"method,omitempty"`
	Scheme                                  string             `yaml:"scheme,omitempty" json:"scheme,omitempty" mapstructure:"scheme,omitempty"`
	HostName                                string             `yaml:"hostname,omitempty" json:"hostname,omitempty" mapstructure:"hostname,omitempty"`
	Port                                    string             `yaml:"port,omitempty" json:"port,omitempty" mapstructure:"port,omitempty"`
	Path                                    string             `yaml:"Path,omitempty" json:"Path,omitempty" mapstructure:"Path,omitempty"`
	Headers                                 []NameValuePair    `yaml:"headers,omitempty" json:"headers,omitempty" mapstructure:"headers,omitempty"`
	QueryString                             []NameValuePair    `yaml:"query-string,omitempty" json:"query-string,omitempty" mapstructure:"query-string,omitempty"`
	Body                                    PostData           `yaml:"body,omitempty" json:"body,omitempty" mapstructure:"body,omitempty"`
	OnResponseActions                       []OnResponseAction `yaml:"on-response,omitempty" json:"on-response,omitempty" mapstructure:"on-response,omitempty"`
	IgnoreNonApplicationJsonResponseContent bool               `yaml:"ignore-non-json-response-body,omitempty" json:"ignore-non-json-response-body,omitempty" mapstructure:"ignore-non-json-response-body,omitempty"`
	HttpClientOptions                       *HttpClientOptions `yaml:"http-client-opts,omitempty" json:"http-client-opts,omitempty" mapstructure:"http-client-opts,omitempty"`
	CacheConfig                             CacheConfig        `yaml:"with-cache,omitempty" json:"with-cache,omitempty" mapstructure:"with-cache,omitempty"`
}

func (epd *EndpointDefinition) WriteToFile(folderName string, fileName string) error {
	const semLogContext = "endpoint-definition::write-to-file"
	fn := filepath.Join(folderName, fileName)
	log.Info().Str("file-name", fn).Msg(semLogContext)
	b, err := yaml.Marshal(epd)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return err
	}

	outFileName, _ := util.ResolvePath(fn)
	err = os.WriteFile(outFileName, b, fs.ModePerm)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return err
	}

	return nil
}

func (epd *EndpointDefinition) PortAsInt() int {

	if epd.Port == "" {
		return 0
	}

	p, err := strconv.Atoi(epd.Port)
	if err != nil {
		log.Error().Err(err).Str("port", epd.Port).Str("path", epd.Path).Msg("port conversion error in endpoint definition")
		return 0
	}

	return p
}
