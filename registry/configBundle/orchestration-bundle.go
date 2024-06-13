package configBundle

import (
	"fmt"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util"
	"github.com/rs/zerolog/log"
	"io/ioutil"
	"path/filepath"
)

func NewOrchestrationRepoFromFolder(mountPoint string) (OrchestrationBundle, error) {

	const semLogContext = "repo::new-repo-from-folder"
	log.Trace().Str("mount-point", mountPoint).Msg(semLogContext)

	repo := OrchestrationBundle{
		Path: mountPoint,
	}

	subFolders, err := util.FindFiles(mountPoint, util.WithFindFileType(util.FileTypeDir), util.WithFindOptionIgnoreList(DefaultIgnoreList))
	if err != nil {
		return repo, err
	}

	m := make(map[string]AssetGroup)
	for _, fld := range subFolders {
		fldName, assetGroup, err := LoadOrchestrationRepo(fld)
		if err != nil {
			return repo, err
		}

		if !assetGroup.Root.IsZero() {
			m[fldName] = assetGroup
		}
	}

	repo.AssetGroups = m
	return repo, nil
}

func (r *OrchestrationBundle) GetPath() string {
	return r.Path
}

func (r *OrchestrationBundle) GetOrchestrationData(sid string) ([]byte, []Asset, error) {

	aGroup, ok := r.AssetGroups[sid]
	if !ok {
		return nil, nil, fmt.Errorf("no orchestration found for %s", sid)
	}

	orchestrationFile := filepath.Join(r.Path, aGroup.Root.Path)
	orchestrationData, err := util.ReadFileAndResolveEnvVars(orchestrationFile) // ioutil.ReadFile(orchestrationFile)
	if err != nil {
		return nil, nil, err
	}

	var assets []Asset
	for i, ref := range aGroup.Refs {
		resolvedPath := filepath.Join(r.Path, ref.Path)
		b, err := util.ReadFileAndResolveEnvVars(resolvedPath) // ioutil.ReadFile(resolvedPath)
		if err != nil {
			return nil, nil, err
		}

		aGroup.Refs[i].Data = b
		assets = append(assets, aGroup.Refs[i])
	}

	return orchestrationData, assets, err
}

func (r *OrchestrationBundle) GetRefAssetData(sid string, fn string) ([]byte, error) {

	ctx := sid
	if sid == "" {
		ctx = "open-api"
	}

	var refs []Asset // := r.apiDefinition.Refs
	if sid != "" {
		aGroup, ok := r.AssetGroups[sid]
		if !ok {
			return nil, fmt.Errorf("no orchestration found for %s", sid)
		}

		refs = aGroup.Refs
	}

	if len(refs) == 0 {
		return nil, fmt.Errorf("no assets references found in repo %s in %s", r.Path, ctx)
	}

	ndx := FindAssetIndexByPath(refs, fn)
	if ndx < 0 {
		return nil, fmt.Errorf("no assets references found in repo %s in %s for %s", r.Path, ctx, fn)
	}

	if refs[ndx].Data != nil {
		return refs[ndx].Data, nil
	}

	resolvedPath := filepath.Join(r.Path, refs[ndx].Path)
	b, err := ioutil.ReadFile(resolvedPath)
	if err != nil {
		return nil, err
	}

	refs[ndx].Data = b
	return b, nil
}

func Scan4AssetFiles(dir string) ([]Asset, error) {
	files, err := util.FindFiles(dir, util.WithFindFileType(util.FileTypeFile), util.WithFindOptionIgnoreList(DefaultIgnoreList))
	if err != nil {
		return nil, err
	}

	assets := make([]Asset, 0, len(files))
	for _, a := range files {

		fileType, _ := GetFileTypeByName(a)
		switch fileType {
		case AssetTypeVersion:
			assets = append(assets, Asset{Name: filepath.Base(a), Type: AssetTypeVersion, Path: filepath.Base(a)})
		case AssetTypeSHA:
			assets = append(assets, Asset{Name: filepath.Base(a), Type: AssetTypeSHA, Path: filepath.Base(a)})
		default:
			assets = append(assets, Asset{Name: filepath.Base(a), Type: AssetTypeExternalValue, Path: filepath.Base(a)})
		}
	}

	return assets, nil
}

func LoadOrchestrationRepo(dir string) (string, AssetGroup, error) {

	g := AssetGroup{}
	files, err := util.FindFiles(dir, util.WithFindFileType(util.FileTypeFile), util.WithFindOptionIgnoreList(DefaultIgnoreList))
	if err != nil {
		return "", AssetGroup{}, err
	}

	for _, a := range files {
		pa := filepath.Join(filepath.Base(dir), filepath.Base(a))
		na := filepath.Base(a)

		fileType, fileQualifier := GetFileTypeByName(na)
		switch fileType {
		case AssetTypeOrchestration:
			g.Root = Asset{Name: na, Path: pa, Type: AssetTypeOrchestration}
		case AssetTypeDictionary:
			g.Refs = append(g.Refs, Asset{Type: AssetTypeDictionary, Name: fileQualifier, Path: pa})
		case AssetTypeVersion:
			g.Refs = append(g.Refs, Asset{Type: AssetTypeVersion, Name: na, Path: pa})
		case AssetTypeSHA:
			g.Refs = append(g.Refs, Asset{Type: AssetTypeSHA, Name: na, Path: pa})
		default:
			g.Refs = append(g.Refs, Asset{Type: AssetTypeExternalValue, Name: na, Path: pa})
		}
	}

	// Need to skip sub-folders without orchestration files.
	if g.Root.IsZero() {
		log.Info().Str("folder", dir).Msg("not an orchestration folder")
		return "", AssetGroup{}, nil
	}

	// Load dicts also from dicts subfolder....
	dictsSubFolder := filepath.Join(dir, "dicts")
	if util.FileExists(dictsSubFolder) {
		files, err = util.FindFiles(filepath.Join(dir, "dicts"), util.WithFindFileType(util.FileTypeFile), util.WithFindOptionIgnoreList(DefaultIgnoreList))
		if err != nil {
			return "", AssetGroup{}, err
		}

		for _, a := range files {
			pa := filepath.Join(filepath.Base(dir), "dicts", filepath.Base(a))
			na := filepath.Base(a)

			fileType, fileQualifier := GetFileTypeByName(na)
			switch fileType {
			case AssetTypeDictionary:
				g.Refs = append(g.Refs, Asset{Type: AssetTypeDictionary, Name: fileQualifier, Path: pa})
			}
		}
	}
	return filepath.Base(dir), g, nil
}

func LoadOrchestrationData(workPath string, aGroup AssetGroup) ([]byte, []Asset, error) {

	orchestrationFile := filepath.Join(workPath, aGroup.Root.Path)
	orchestrationData, err := util.ReadFileAndResolveEnvVars(orchestrationFile) // ioutil.ReadFile(orchestrationFile)
	if err != nil {
		return nil, nil, err
	}

	var assets []Asset
	for i, ref := range aGroup.Refs {
		resolvedPath := filepath.Join(workPath, ref.Path)
		b, err := util.ReadFileAndResolveEnvVars(resolvedPath) // ioutil.ReadFile(resolvedPath)
		if err != nil {
			return nil, nil, err
		}

		aGroup.Refs[i].Data = b
		assets = append(assets, aGroup.Refs[i])
	}

	return orchestrationData, assets, err
}

func FindAssetIndexByPath(assets []Asset, p string) int {
	for i, a := range assets {
		if a.Path == p {
			return i
		}
	}

	return -1
}
