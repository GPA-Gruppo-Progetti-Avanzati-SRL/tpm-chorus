package repo

import (
	"github.com/rs/zerolog/log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	SHAFileName                  = "SHA"
	VERSIONFileName              = "VERSION"
	DictionaryFileNamePattern    = "^dict-([a-zA-Z_-]+)\\.(?:yaml|yml)$"
	OrchestrationFileNamePattern = "^(tpm-symphony-orchestration|tpm-orchestration)\\.(yml|yaml)$"
	OrchestrationFileName        = "tpm-orchestration.yml"
)

const (
	AssetTypeOrchestration    = "asset-orchestration"
	AssetTypeValue            = "asset-value"
	AssetTypeExternalValue    = "asset-external-value"
	AssetTypeDictionary       = "asset-dictionary"
	AssetTypeVersion          = "asset-version"
	AssetTypeSHA              = "asset-sha"
	AssetTypeExternalTemplate = "asset-external-template"
	AssetTypeInlineTemplate   = "asset-inline-template"
)

var OrchestrationFileNameRegexp = regexp.MustCompile(OrchestrationFileNamePattern)
var DictionaryFileNameRegexp = regexp.MustCompile(DictionaryFileNamePattern)

var DefaultIgnoreList = []string{
	"^\\.",
}

var VersionSHAFileFindIncludeList = []string{
	SHAFileName,
	VERSIONFileName,
}

func NameIsOrchestrationFile(n string) (string, bool) {
	matches := OrchestrationFileNameRegexp.FindAllSubmatch([]byte(n), -1)
	if len(matches) > 0 {
		return string(matches[0][1]), true
	}

	return "", false
}

func NameIsDictionary(n string) (string, bool) {
	matches := DictionaryFileNameRegexp.FindAllSubmatch([]byte(n), -1)
	if len(matches) > 0 {
		return string(matches[0][1]), true
	}

	return "", false
}

func NameIsVersion(n string) (string, bool) {
	baseName := strings.ToUpper(filepath.Base(n))
	if baseName == VERSIONFileName {
		return n, true
	}

	return "", false
}

func ReadVersionFile(fn string) string {
	const semLogContext = "registry::Read-version-file"
	version, err := readStringFromFile(fn)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
	}

	return version
}

func NameIsSHA(n string) (string, bool) {
	baseName := strings.ToUpper(filepath.Base(n))
	if baseName == SHAFileName {
		return n, true
	}

	return "", false
}

func ReadSHAFile(fn string) string {
	const semLogContext = "registry::Read-sha-file"
	sha, err := readStringFromFile(fn)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
	}

	return sha
}

func readStringFromFile(fn string) (string, error) {
	b, err := os.ReadFile(fn)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func GetFileTypeByName(fn string) (string, string) {
	if _, ok := NameIsOrchestrationFile(fn); ok {
		return AssetTypeOrchestration, fn
	}

	if dictName, ok := NameIsDictionary(fn); ok {
		return AssetTypeDictionary, dictName
	}

	if _, ok := NameIsVersion(fn); ok {
		return AssetTypeVersion, fn
	}

	if _, ok := NameIsSHA(fn); ok {
		return AssetTypeSHA, fn
	}

	return AssetTypeExternalValue, fn
}
