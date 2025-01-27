package operators_test

import (
	"embed"
	"errors"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/kzxform"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util/fileutil"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
)

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
			_, err := c.executeXFormByEntryNdx(i, writeOutput)
			if err != nil {
				log.Error().Err(err).Str("rule", entry.Rule).Msg(semLogContext)
				return err
			}
		}
	}

	return nil
}

func (c Catalog) executeXFormByRuleName(n string, writeOutput bool) ([]byte, error) {
	ndx, err := c.findEntryIndexByRuleName(n)
	if err != nil {
		return nil, err
	}

	if ndx < 0 {
		return nil, errors.New("invalid rule name")
	}

	return c.executeXFormByEntryNdx(ndx, writeOutput)
}

func (c Catalog) executeXFormByEntryNdx(ndx int, writeOutput bool) ([]byte, error) {

	ruleData, inData, err := c.readTestDataByNdx(ndx)
	if err != nil {
		return nil, err
	}

	registry := kzxform.GetRegistry()

	xform := kzxform.Config{}
	err = yaml.Unmarshal(ruleData, &xform)
	if err != nil {
		return nil, err
	}

	_, err = registry.Get(xform.Id)
	if err != nil {
		if errors.Is(err, kzxform.XFormNotFound) {
			err = registry.Add3(xform)
		}
	}

	dataOut, err := registry.Transform(xform.Id, inData)
	if err != nil {
		return nil, err
	}

	if writeOutput {
		err = os.WriteFile(filepath.Join(outRootFolder, c[ndx].Out), dataOut, os.ModePerm)
		if err != nil {
			return nil, err
		}
	}

	return dataOut, nil
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

func TestXForms(t *testing.T) {
	const semLogContext = "test-operators"
	var err error

	filterPrefix := "" // "xform-shift-array"
	err = catalog.executeXForms(filterPrefix, true)
	require.NoError(t, err)
}

func TestXFormByName(t *testing.T) {
	const semLogContext = "test-operators"
	var err error

	ruleName := "xform-set-property-ex-03-rule.yml"
	_, err = catalog.executeXFormByRuleName(ruleName, true)
	require.NoError(t, err)
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

func readSourceTemplates(templates embed.FS, rootFolder string) (map[string][]byte, error) {

	entries, err := fileutil.FindEmbeddedFiles(
		templates, rootFolder,
		fileutil.WithFindOptionNavigateSubDirs() /*, fileutil.WithFindOptionExcludeRootFolderInNames() */, fileutil.WithFindOptionPreloadContent(),
	)

	if err != nil {
		return nil, err
	}

	if len(entries) == 0 {
		return nil, err
	}

	treeNodes := map[string][]byte{}
	for _, e := range entries {
		if e.Info.IsDir() {
			continue
		}

		fulln := e.Info.Name()
		if e.Path != "" {
			p := strings.TrimPrefix(e.Path, rootFolder)
			if p != "" {
				fulln = path.Join(p, fulln)
			}
		}

		treeNodes[fulln] = e.Content
	}

	return treeNodes, nil
}
