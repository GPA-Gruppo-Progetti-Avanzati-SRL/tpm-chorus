package jq_test

import (
	"embed"
	_ "embed"
	"errors"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/xforms/jq"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

//go:embed test-data/cases/input-001.json
var inputJson string

//go:embed test-data/cases/jq-001.txt
var jqText []byte

func TestJQ(t *testing.T) {
	dataOut, err := jq.ApplyJQTransformation(jqText, inputJson)
	require.NoError(t, err)

	t.Logf("dataOut: %#v", dataOut)
}

func TestXForms(t *testing.T) {
	const semLogContext = "test-operators"
	var err error

	filterPrefix := ""
	err = catalog.executeXForms(filterPrefix, true)
	require.NoError(t, err)
}

type CatalogEntry struct {
	Rule string `yaml:"rule,omitempty" mapstructure:"rule,omitempty" json:"rule,omitempty"`
	In   string `yaml:"in,omitempty" mapstructure:"in,omitempty" json:"in,omitempty"`
	Out  string `yaml:"out,omitempty" mapstructure:"out,omitempty" json:"out,omitempty"`
}

type Catalog []CatalogEntry

func (c Catalog) executeXForms(prefix string, writeOutput bool) error {
	const semLogContext = "catalog::execute-forms"
	for i, entry := range c {
		doIt := false
		if prefix != "" {
			if strings.HasPrefix(entry.Rule, prefix) {
				doIt = true
			}
		} else {
			doIt = true
		}

		if doIt {
			err := c.executeXFormByEntryNdx(i, writeOutput)
			if err != nil {
				log.Error().Err(err).Str("rule", entry.Rule).Msg(semLogContext)
				return err
			}
		}
	}

	return nil
}

func (c Catalog) executeXFormByEntryNdx(ndx int, writeOutput bool) error {

	ruleData, inData, err := c.readTestDataByNdx(ndx)
	if err != nil {
		return err
	}

	dataOut, err := jq.ApplyJQTransformationToJson(ruleData, inData)
	if err != nil {
		return err
	}

	log.Info().Str("dataOut", string(dataOut)).Msg("dataOut")

	if writeOutput {
		err = os.WriteFile(filepath.Join(outRootFolder, c[ndx].Out), dataOut, os.ModePerm)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c Catalog) findEntryIndexByRuleName(n string) (int, error) {
	for i, entry := range c {
		if entry.Rule == n {
			return i, nil
		}
	}

	return -1, nil
}

func (c Catalog) readTestDataByNdx(ndx int) ([]byte, []byte, error) {
	if ndx < 0 {
		return nil, nil, errors.New("invalid index")
	}

	ruleData, err := testCases.ReadFile(path.Join(EmbeddedRootFolder, c[ndx].Rule))
	if err != nil {
		return nil, nil, err
	}

	inData, err := testCases.ReadFile(path.Join(EmbeddedRootFolder, c[ndx].In))
	if err != nil {
		return nil, nil, err
	}

	return ruleData, inData, nil
}

//go:embed all:test-data/cases/*
var testCases embed.FS

const EmbeddedRootFolder = "test-data/cases"
const outRootFolder = "test-data/outs"

var catalog Catalog

func TestMain(m *testing.M) {
	var err error

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	catalog, err = readCatalog()
	if err != nil {
		panic(err)
	}

	if len(catalog) == 0 {
		err = errors.New("catalog is empty")
		panic(err)
	}

	exitVal := m.Run()
	os.Exit(exitVal)
}

func readCatalog() (Catalog, error) {
	catalogData, err := testCases.ReadFile(path.Join(EmbeddedRootFolder, "catalog.yml"))
	if err != nil {
		return nil, err
	}

	var catalog Catalog
	err = yaml.Unmarshal(catalogData, &catalog)
	if err != nil {
		return nil, err
	}

	return catalog, nil
}
