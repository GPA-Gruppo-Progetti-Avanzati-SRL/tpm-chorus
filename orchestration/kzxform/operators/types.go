package operators

import (
	"fmt"
	"strings"
)

type JsonReference struct {
	WithArrayISpecifierIndex    int
	WithArrayPushSpecifierIndex int
	Path                        string
	Keys                        []string
}

func MustToJsonReference(s string) JsonReference {
	j, err := ToJsonReference(s)
	if err != nil {
		panic(err)
	}

	return j
}

func ToJsonReference(s string) (JsonReference, error) {
	keys, iSpecifierNdx, plusNdx, err := SplitKeySpecifier(s)
	if err != nil {
		return JsonReference{}, err
	}

	jr := JsonReference{
		Path:                        s,
		Keys:                        keys,
		WithArrayISpecifierIndex:    iSpecifierNdx,
		WithArrayPushSpecifierIndex: plusNdx,
	}

	return jr, nil
}

func (jr JsonReference) IsZero() bool {
	return jr.Path == ""
}

func (jr JsonReference) JsonReferenceToArrayWithIotaSpecifier() JsonReference {
	rootRef := JsonReference{
		WithArrayISpecifierIndex: -1,
		Path:                     jr.Path[:strings.Index(jr.Path, "[i]")],
		Keys:                     jr.Keys[:jr.WithArrayISpecifierIndex],
	}

	return rootRef
}

func (jr JsonReference) JsonReferenceToArrayItemWithIotaSpecifier(i int) JsonReference {
	rootRef := JsonReference{
		WithArrayISpecifierIndex: -1,
		Path:                     strings.ReplaceAll(jr.Path, "[i]", fmt.Sprintf("[%d]", i)),
	}
	rootRef.Keys = append(rootRef.Keys, jr.Keys...)
	rootRef.Keys[jr.WithArrayISpecifierIndex] = fmt.Sprintf("[%d]", i)
	return rootRef
}

func (jr JsonReference) JsonReferenceToArrayNestedItemWithIotaSpecifierBoh(i int) JsonReference {
	nestedRef := JsonReference{
		WithArrayISpecifierIndex: -1,
		Path:                     strings.ReplaceAll(jr.Path[strings.Index(jr.Path, "[i]"):], "[i]", fmt.Sprintf("[%d]", i)),
	}
	nestedRef.Keys = append(nestedRef.Keys, jr.Keys[jr.WithArrayISpecifierIndex:]...)
	nestedRef.Keys[0] = fmt.Sprintf("[%d]", i)
	return nestedRef
}

func (jr JsonReference) JsonReferenceToArrayNestedItemWithIotaSpecifier(i int) JsonReference {
	nestedRef := JsonReference{
		WithArrayISpecifierIndex: -1,
		Path:                     jr.Path[strings.Index(jr.Path, "[i]")+len("[i]")+1:],
	}
	nestedRef.Keys = append(nestedRef.Keys, jr.Keys[jr.WithArrayISpecifierIndex+1:]...)
	return nestedRef
}
