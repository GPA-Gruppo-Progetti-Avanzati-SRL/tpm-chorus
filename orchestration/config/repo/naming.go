package repo

import (
	"path/filepath"
	"regexp"
	"strings"
)

const (
	SHAFileName                  = "SHA"
	VERSIONFileName              = "VERSION"
	DictionaryFileNamePattern    = "^dict-([a-zA-Z_-]+)\\.(?:yaml|yml)$"
	OrchestrationFileNamePattern = "^tpm-symphony-orchestration\\.(yml|yaml)$"
)

const (
	AssetTypeOrchestration = "asset-orchestration"
	AssetTypeExternalValue = "asset-external-value"
	AssetTypeDictionary    = "asset-dictionary"
	AssetTypeVersion       = "asset-version"
	AssetTypeSHA           = "asset-sha"
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

func NameIsSHA(n string) (string, bool) {
	baseName := strings.ToUpper(filepath.Base(n))
	if baseName == SHAFileName {
		return n, true
	}

	return "", false
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
