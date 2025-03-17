package mongosink

import (
	"errors"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/config/repo"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util/fileutil"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-mongo-common/jsonops"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"strings"
)

const (
	SinkTypeMongo = "mongo-db"
)

type MongoSinkDefinition struct {
	OpType         jsonops.MongoJsonOperationType                     `yaml:"-" json:"-" mapstructure:"-"`
	LksName        string                                             `yaml:"lks-name,omitempty" json:"lks-name,omitempty" mapstructure:"lks-name,omitempty"`
	CollectionId   string                                             `yaml:"collection-id,omitempty" json:"collection-id,omitempty" mapstructure:"collection-id,omitempty"`
	Statement      map[jsonops.MongoJsonOperationStatementPart]string `yaml:"statement,omitempty" json:"statement,omitempty" mapstructure:"statement,omitempty"`
	StatementParts map[jsonops.MongoJsonOperationStatementPart][]byte
}

func (def *MongoSinkDefinition) WriteToFile(folderName string, fileName string, writeOpts ...fileutil.WriteOption) error {
	const semLogContext = "mongo-sink-definition::write-to-file"
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

func (def *MongoSinkDefinition) LoadStatement(refs repo.AssetGroup) (map[jsonops.MongoJsonOperationStatementPart][]byte, error) {
	const semLogContext = "mongo-sink-definition::load-statement-config"
	var err error

	var statementData = map[jsonops.MongoJsonOperationStatementPart][]byte{}
	for n, p := range def.Statement {
		sdata := []byte(p)
		if !(strings.HasPrefix(p, "{") || strings.HasPrefix(p, "[")) {
			sdata, err = refs.ReadRefsData(p)
			if err != nil {
				err = errors.New("cannot find mongo statement definition")
				log.Error().Err(err).Msg(semLogContext)
				return nil, err
			}
		}
		statementData[n] = sdata
	}

	return statementData, err
}
