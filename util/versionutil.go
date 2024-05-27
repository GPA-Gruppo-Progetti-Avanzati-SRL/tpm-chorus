package util

var version string

func SetVersion(aVersion string) {
	version = aVersion
}

func GetVersion() string {
	return version
}
