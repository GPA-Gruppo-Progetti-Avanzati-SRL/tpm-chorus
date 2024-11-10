package repo

import (
	"fmt"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util/fileutil"
	"github.com/rs/zerolog/log"
	"path/filepath"
)

func NewOrchestrationBundleFromFolder(dir string) (OrchestrationBundle, error) {
	const semLogContext = "new-orchestration-bundle-from-folder"

	bundle := OrchestrationBundle{
		Path: dir,
	}

	files, err := fileutil.FindFiles(dir, fileutil.WithFindFileType(fileutil.FileTypeFile), fileutil.WithFindOptionIgnoreList(DefaultIgnoreList))
	if err != nil {
		return bundle, err
	}

	for _, a := range files {
		pa := filepath.Base(a) // filepath.Join(filepath.Base(dir), filepath.Base(a))
		na := filepath.Base(a)

		fileType, fileQualifier := GetFileTypeByName(na)
		switch fileType {
		case AssetTypeOrchestration:
			bundle.AssetGroup.MountPoint = dir
			bundle.AssetGroup.Asset = Asset{Name: na, Path: pa, Type: AssetTypeOrchestration}
		case AssetTypeDictionary:
			bundle.AssetGroup.Refs = append(bundle.AssetGroup.Refs, Asset{Type: AssetTypeDictionary, Name: fileQualifier, Path: pa})
		case AssetTypeVersion:
			bundle.Version = ReadVersionFile(a)
			// bundle.AssetGroup.Refs = append(bundle.AssetGroup.Refs, Asset{Type: AssetTypeVersion, Name: na, Path: pa})
		case AssetTypeSHA:
			bundle.SHA = ReadSHAFile(a)
			// bundle.AssetGroup.Refs = append(bundle.AssetGroup.Refs, Asset{Type: AssetTypeSHA, Name: na, Path: pa})
		default:
			bundle.AssetGroup.Refs = append(bundle.AssetGroup.Refs, Asset{Type: AssetTypeExternalValue, Name: na, Path: pa})
		}
	}

	// Need to skip sub-folders without orchestration files.
	if bundle.AssetGroup.Asset.IsZero() {
		log.Info().Str("folder", dir).Msg("not an orchestration folder")
		return bundle, nil
	}

	// Load dicts also from dicts subfolder....
	dictsSubFolder := filepath.Join(dir, "dicts")
	if fileutil.FileExists(dictsSubFolder) {
		files, err = fileutil.FindFiles(filepath.Join(dir, "dicts"), fileutil.WithFindFileType(fileutil.FileTypeFile), fileutil.WithFindOptionIgnoreList(DefaultIgnoreList))
		if err != nil {
			return bundle, err
		}

		for _, a := range files {
			pa := filepath.Join("dicts", filepath.Base(a))
			na := filepath.Base(a)

			fileType, fileQualifier := GetFileTypeByName(na)
			switch fileType {
			case AssetTypeDictionary:
				bundle.AssetGroup.Refs = append(bundle.AssetGroup.Refs, Asset{Type: AssetTypeDictionary, Name: fileQualifier, Path: pa})
			}
		}
	}

	nestedOrchestrations, err := FindNestedOrchestrations(dir)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return bundle, err
	}

	var nestedBundles []OrchestrationBundle
	for _, nested := range nestedOrchestrations {
		nestedBundle, err := NewOrchestrationBundleFromFolder(nested)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return bundle, err
		}

		log.Info().Str("nested-bundle-path", nestedBundle.Path).Msg(semLogContext)
		nestedBundles = append(nestedBundles, nestedBundle)
	}
	bundle.NestedBundles = nestedBundles

	return bundle, nil
}

func (r *OrchestrationBundle) LoadOrchestrationData() ([]byte, []Asset, error) {

	const semLogContext = "orchestration-bundle::load-orchestration-data"

	orchestrationFile := filepath.Join(r.AssetGroup.MountPoint, r.AssetGroup.Asset.Path)
	orchestrationData, err := util.ReadFileAndResolveEnvVars(orchestrationFile) // ioutil.ReadFile(orchestrationFile)
	if err != nil {
		return nil, nil, err
	}

	r.AssetGroup.Asset.Data = orchestrationData

	var assets []Asset
	for i, ref := range r.AssetGroup.Refs {
		resolvedPath := filepath.Join(r.AssetGroup.MountPoint, ref.Path)
		b, err := util.ReadFileAndResolveEnvVars(resolvedPath) // ioutil.ReadFile(resolvedPath)
		if err != nil {
			return nil, nil, err
		}

		r.AssetGroup.Refs[i].Data = b
		assets = append(assets, r.AssetGroup.Refs[i])
	}

	for i := range r.NestedBundles {
		_, _, err = r.NestedBundles[i].LoadOrchestrationData()
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return nil, nil, err
		}
	}
	return orchestrationData, assets, err
}

func (r *OrchestrationBundle) GetRefAssetData(fn string) ([]byte, error) {

	if len(r.AssetGroup.Refs) == 0 {
		return nil, fmt.Errorf("no assets references found in repo %s for %s", r.Path, fn)
	}

	ndx := findAssetIndexByPath(r.AssetGroup.Refs, fn)
	if ndx < 0 {
		return nil, fmt.Errorf("no assets references found in repo %s for %s", r.Path, fn)
	}

	if r.AssetGroup.Refs[ndx].Data != nil {
		return r.AssetGroup.Refs[ndx].Data, nil
	}

	resolvedPath := filepath.Join(r.AssetGroup.MountPoint, r.AssetGroup.Refs[ndx].Path)
	b, err := util.ReadFileAndResolveEnvVars(resolvedPath)
	if err != nil {
		return nil, err
	}

	r.AssetGroup.Refs[ndx].Data = b
	return b, nil
}

func findAssetIndexByPath(assets []Asset, p string) int {
	for i, a := range assets {
		if a.Path == p {
			return i
		}
	}

	return -1
}

func FindNestedOrchestrations(dir string) ([]string, error) {
	const semLogContext = "find-nested-orchestrations"

	folders, err := fileutil.FindFiles(dir, fileutil.WithFindFileType(fileutil.FileTypeDir), fileutil.WithFindOptionFoldersIgnoreList([]string{"dicts"}))
	if err != nil {
		return nil, err
	}

	var resp []string
	for _, f := range folders {
		n := filepath.Join(f, "tpm-orchestration.yml")
		if fileutil.FileExists(n) {
			resp = append(resp, f)
		}
	}

	return resp, nil
}

/*
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

func LoadOrchestrationData(workPath string, aGroup AssetGroup) ([]byte, []Asset, error) {

	orchestrationFile := filepath.Join(workPath, aGroup.Asset.Path)
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
*/
