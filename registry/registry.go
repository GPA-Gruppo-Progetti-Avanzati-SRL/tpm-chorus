package registry

import (
	"context"
	"fmt"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/constants"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/orchestration/config"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/registry/configBundle"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-chorus/registry/oapiextensions"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/rs/zerolog/log"
	"regexp"
	"strings"
)

type OpenApiRegistry struct {
	Entries []OpenApiRegistryEntry `yaml:"repos" mapstructure:"repos" json:"repos"`
}

var theRegistry OpenApiRegistry

func GetRegistry() *OpenApiRegistry {
	return &theRegistry
}

func LoadRegistry(cfg *Config) (*OpenApiRegistry, error) {

	const semLogContext = "open-api-registry::load-registry"

	theRegistry = OpenApiRegistry{}

	topVersion, topSha, err := Scan4TopLevelVersionAndSHA(cfg.MountPoint)
	if err != nil {
		return nil, err
	}

	orchs, err := Crawl(cfg)
	if err != nil {
		return nil, err
	}

	if len(orchs) == 0 {
		return nil, fmt.Errorf("no orchestrations found in %s", cfg.MountPoint)
	}

	for _, r := range orchs {

		oapiName, oapiContent, err := r.getOpenApiData()
		log.Info().Str(constants.SemLogPath, r.OrchestrationBundle.GetPath()).Str("name", oapiName).Msg(semLogContext + " found openapi file")
		doc, err := openapi3.NewLoader().LoadFromData(oapiContent)
		if err != nil {
			return nil, err
		}

		err = validateOpenapiDoc(doc)
		if err != nil {
			return nil, err
		}

		smpExtensions := oapiextensions.RetrieveSymphonyPathExtensions(doc)
		if len(smpExtensions) == 0 {
			return nil, fmt.Errorf("cannot load symphony extensions from file %s in %s", oapiName, r.OrchestrationBundle.GetPath())
		}

		ver, sha := r.GetOpenApiVersionAndSha()
		if ver == "" && sha == "" {
			ver = topVersion
			sha = topSha
		}

		orchestrations, err := loadOrchestrations(&r, smpExtensions, ver, sha)
		if err != nil {
			return nil, err
		}
		r.OpenapiDoc = doc
		r.Orchestrations = orchestrations
		r.Version = ver
		r.SHA = sha
		// entry := OpenApiRepo{OpenapiDoc: doc, Repo: r.Repo, Orchestrations: orchestrations /*, Version: ver, SHA: sha */}
		theRegistry.Entries = append(theRegistry.Entries, r)
	}

	return &theRegistry, nil
}

var ServersUrlPattern = regexp.MustCompile(`^(?:http|https)://[0-9a-zA-Z\.]*(?:\:[0-9]{2,4})?(.*)`)

func loadOrchestrations(oapiRepo *OpenApiRegistryEntry, oIds []oapiextensions.SymphonyOperationExtension, version, sha string) ([]config.Orchestration, error) {

	const semLogContext = "registry::load-orchestrations"

	var os []config.Orchestration
	for _, xs := range oIds {
		b, assets, err := oapiRepo.OrchestrationBundle.GetOrchestrationData(xs.Id)
		if err != nil {
			return nil, err
		}

		o, err := config.NewOrchestrationFromYAML(b)
		if err != nil {
			return nil, err
		}

		// id in file is overridden in case is not present or is just different
		o.Id = xs.Id
		if o.Description == "" {
			o.Description = xs.Comment
		}

		for _, a := range assets {
			switch a.Type {
			case configBundle.AssetTypeDictionary:
				var d config.Dictionary
				d, err = config.NewDictionary(a.Name, a.Data)
				o.Dictionaries = append(o.Dictionaries, d)
			case configBundle.AssetTypeSHA:
				o.SHA = strings.TrimSpace(string(a.Data))
			case configBundle.AssetTypeVersion:
				o.Version = strings.TrimSpace(string(a.Data))
			default:
				o.References = append(o.References, config.DataReference{Path: a.Name, Data: a.Data})
			}

			if err != nil {
				return nil, err
			}
		}

		if o.Version == "" && o.SHA == "" {
			o.Version = version
			o.SHA = sha
		}

		log.Trace().Str("orch-id", o.Id).Str("version", o.Version).Str("SHA", o.SHA).Msg(semLogContext + " orchestration version and sha")
		os = append(os, o)
	}

	return os, nil
}

func validateOpenapiDoc(doc *openapi3.T) error {
	err := doc.Validate(context.Background())
	if err != nil {
		return err
	}

	// Hack. In case the url is in the form http://localhost:8080/something... I reset to /something....
	// This because otherwise the openapi3 lib doesn't match it.
	if len(doc.Servers) > 0 {
		u := util.ExtractCapturedGroupIfMatch(ServersUrlPattern, doc.Servers[0].URL)
		doc.Servers[0].URL = u
	}

	return nil
}

func (reg *OpenApiRegistry) ShowInfo() {

	for _, e := range reg.Entries {
		e.OrchestrationBundle.ShowInfo()
	}

	for _, e := range reg.Entries {
		log.Info().Str(constants.SemLogPath, e.OrchestrationBundle.GetPath()).Str("open-api-ver", e.OpenapiDoc.OpenAPI).Msg("open api info")
	}
}
