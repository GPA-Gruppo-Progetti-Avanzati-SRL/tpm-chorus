package purefuncs

import "encoding/base64"

func Base64(text string) string {
	if text == "" {
		return ""
	}

	return base64.StdEncoding.EncodeToString([]byte(text))
}
