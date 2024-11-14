package config

import (
	"errors"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util/fileutil"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-mongo-common/jsonops"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"strings"
)

type MongoActivityOpType string

const (
	MongoActivityFindOne = "find-one"

	MongoActivityFindOneStatementProperty  = "statement"
	MongoActivityFindOneQueryProperty      = "query"
	MongoActivityFindOneSortProperty       = "sort"
	MongoActivityFindOneProjectionProperty = "projection"
	MongoActivityFindOneOptsProperty       = "opts"

	MongoOperationResultMatchedCountPropertyVarName   = "matched-count"
	MongoOperationResultModifiedCountPropertyVarName  = "modified-count"
	MongoOperationResultUpsertedCountPropertyVarNName = "upserted-count"
	MongoOperationResultDeletedCountPropertyVarName   = "deleted-count"
	MongoOperationResultObjectIDPropertyVarName       = "object-id"
)

var opTypes = map[jsonops.MongoJsonOperationType]struct{}{
	jsonops.FindOneOperationType:      struct{}{},
	jsonops.ReplaceOneOperationType:   struct{}{},
	jsonops.AggregateOneOperationType: struct{}{},
	jsonops.UpdateOneOperationType:    struct{}{},
	jsonops.UpdateManyOperationType:   struct{}{},
	jsonops.DeleteManyOperationType:   struct{}{},
}

type MongoActivityDefinition struct {
	OpType            jsonops.MongoJsonOperationType                     `yaml:"op-type,omitempty" json:"op-type,omitempty" mapstructure:"op-type,omitempty"`
	LksName           string                                             `yaml:"lks-name,omitempty" json:"lks-name,omitempty" mapstructure:"lks-name,omitempty"`
	CollectionId      string                                             `yaml:"collection-id,omitempty" json:"collection-id,omitempty" mapstructure:"collection-id,omitempty"`
	StatementData     map[jsonops.MongoJsonOperationStatementPart]string `yaml:"statement,omitempty" json:"statement,omitempty" mapstructure:"statement,omitempty"`
	OnResponseActions OnResponseActions                                  `yaml:"on-response,omitempty" json:"on-response,omitempty" mapstructure:"on-response,omitempty"`
	CacheConfig       CacheConfig                                        `yaml:"with-cache,omitempty" json:"with-cache,omitempty" mapstructure:"with-cache,omitempty"`
	Statement         interface{}                                        `yaml:"-" json:"-" mapstructure:"-"`
}

func UnmarshalMongoActivityDefinition(opType jsonops.MongoJsonOperationType, def string, refs DataReferences) (MongoActivityDefinition, error) {
	const semLogContext = "mongo-activity-definition::unmarshal"

	var err error
	maDef := MongoActivityDefinition{OpType: opType}
	data, ok := refs.Find(def)
	if len(data) == 0 || !ok {
		err = errors.New("cannot find mongo activity definition")
		log.Error().Err(err).Msg(semLogContext)
		return maDef, err
	}

	err = yaml.Unmarshal(data, &maDef)
	if err != nil {
		return maDef, err
	}

	if _, ok := opTypes[opType]; !ok {
		err = errors.New("unsupported op-type")
		log.Error().Err(err).Str("op-type", string(opType)).Msg(semLogContext)
		return maDef, err
	}

	// Clear the cache config if not a retrieve type of operation
	if opType != jsonops.AggregateOneOperationType && opType != jsonops.FindOneOperationType {
		maDef.CacheConfig = CacheConfig{}
	}

	return maDef, nil
}

func (def *MongoActivityDefinition) WriteToFile(folderName string, fileName string, writeOpts ...fileutil.WriteOption) error {
	const semLogContext = "mongo-activity-definition::write-to-file"
	fn := filepath.Join(folderName, fileName)
	log.Info().Str("file-name", fn).Msg(semLogContext)
	b, err := yaml.Marshal(def)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return err
	}

	err = fileutil.WriteFile(fn, b, os.ModePerm, writeOpts...)
	//outFileName, _ := fileutil.ResolvePath(fn)
	//err = os.WriteFile(outFileName, b, fs.ModePerm)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return err
	}

	return nil
}

func (def *MongoActivityDefinition) LoadStatementConfig(refs DataReferences) (map[jsonops.MongoJsonOperationStatementPart][]byte, error) {
	const semLogContext = "mongo-activity-definition::load-statement-config"
	var err error

	var statementData = map[jsonops.MongoJsonOperationStatementPart][]byte{}
	for n, p := range def.StatementData {
		sdata := []byte(p)
		if !(strings.HasPrefix(p, "{") || strings.HasPrefix(p, "[")) {
			var ok bool
			sdata, ok = refs.Find(p)
			if !ok {
				err = errors.New("cannot find mongo activity definition")
				log.Error().Err(err).Msg(semLogContext)
				return nil, err
			}
		}
		statementData[n] = sdata
	}

	return statementData, err
}
