package request

import (
	"encoding/json"
	"github.com/superioz/aqua/internal/config"
	"mime/multipart"
)

var (
	emptyRequestMetadata = &RequestMetadata{Expiration: config.ExpireNever}
)

// RequestFormFile is the metadata we get from the file
// which is requested to be uploaded.
type RequestFormFile struct {
	File          multipart.File
	ContentType   string
	ContentLength int64
}

type RequestMetadata struct {
	Expiration int64 `json:"expiration"`
}

func GetMetadata(form *multipart.Form) *RequestMetadata {
	metaRawList := form.Value["metadata"]
	if len(metaRawList) == 0 {
		return emptyRequestMetadata
	}
	metaRaw := metaRawList[0]

	var metadata *RequestMetadata
	err := json.Unmarshal([]byte(metaRaw), &metadata)
	if err != nil {
		return emptyRequestMetadata
	}
	return metadata
}
