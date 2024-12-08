package operators

type JsonReference struct {
	WithArrayISpecifierIndex int
	Path                     string
	Keys                     []string
}

func MustToJsonReference(s string) JsonReference {
	j, err := ToJsonReference(s)
	if err != nil {
		panic(err)
	}

	return j
}

func ToJsonReference(s string) (JsonReference, error) {
	keys, iSpecifierNdx, err := SplitKeySpecifier(s)
	if err != nil {
		return JsonReference{}, err
	}

	jr := JsonReference{
		Path:                     s,
		Keys:                     keys,
		WithArrayISpecifierIndex: iSpecifierNdx,
	}

	return jr, nil
}

func (jr JsonReference) IsZero() bool {
	return jr.Path == ""
}
