package mime

var (
	// Types is a whitelist of all supported
	// mime types. Taken from https://developer.mozilla.org/
	Types = map[string]string{
		"application/pdf":             "pdf",
		"application/json":            "json",
		"application/gzip":            "gz",
		"application/vnd.rar":         "rar",
		"application/zip":             "zip",
		"application/x-7z-compressed": "7z",
		"image/png":                   "png",
		"image/jpeg":                  "jpg",
		"image/gif":                   "gif",
		"image/svg+xml":               "svg",
		"text/csv":                    "csv",
		"text/plain":                  "txt",
		"audio/mpeg":                  "mp3",
		"audio/ogg":                   "ogg",
		"audio/opus":                  "opus",
		"audio/webm":                  "weba",
		"video/mp4":                   "mp4",
		"video/mpeg":                  "mpeg",
		"video/webm":                  "webm",
	}
)

// IsValid checks if given type is inside Types map
func IsValid(t string) bool {
	for mt := range Types {
		if mt == t {
			return true
		}
	}
	return false
}

// GetExtension returns an extension for the given MIME type
func GetExtension(t string) string {
	ext, ok := Types[t]
	if !ok {
		return "application/octet-stream"
	}
	return ext
}
