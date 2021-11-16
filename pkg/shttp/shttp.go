package shttp

import (
	"bytes"
	"fmt"
	"github.com/h2non/filetype"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"strings"
)

// Upload takes an url and multiple reader, that are used to fill in a multipart form.
//
// Heavily inspired by: https://stackoverflow.com/a/20397167/11155150 but with the standard http package,
// there are not many other ways to do that, so it doesn't really matter.
func Upload(client *http.Client, url string, values map[string]io.Reader, header map[string]string) (*http.Response, error) {
	// Prepare a form that you will submit to that URL.
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	for key, r := range values {
		var fw io.Writer
		var err error

		if x, ok := r.(io.Closer); ok {
			defer x.Close()
		}

		// Add an image file
		if file, ok := r.(*os.File); ok {
			mime, err := GetFileType(file)
			if err != nil {
				return nil, err
			}

			fw, err = createFormFile(w, key, file.Name(), mime)
			if err != nil {
				return nil, err
			}
		} else {
			// Add other fields
			fw, err = w.CreateFormField(key)
			if err != nil {
				return nil, err
			}
		}

		// Write to form field
		_, err = io.Copy(fw, r)
		if err != nil {
			return nil, err
		}
	}
	w.Close()

	req, err := http.NewRequest("POST", url, &b)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	for key, val := range header {
		req.Header.Set(key, val)
	}

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// createFormFile is copied from multipart.CreateFormFile, because it
// always sets the content type to `application/octet-stream`.
func createFormFile(w *multipart.Writer, fieldname, filename, contentType string) (io.Writer, error) {
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition",
		fmt.Sprintf(`form-data; name="%s"; filename="%s"`,
			escapeQuotes(fieldname), escapeQuotes(filename)))
	h.Set("Content-Type", contentType)
	return w.CreatePart(h)
}

var quoteEscaper = strings.NewReplacer("\\", "\\\\", `"`, "\\\"")

func escapeQuotes(s string) string {
	return quoteEscaper.Replace(s)
}

// GetFileType returns the MIME type of given file by reading
// the file header of the file that is located on disk.
func GetFileType(file *os.File) (string, error) {
	f, err := os.Open(file.Name())
	if err != nil {
		return "", err
	}

	head := make([]byte, 261)
	_, err = f.Read(head)
	if err != nil {
		return "", err
	}

	kind, err := filetype.Match(head)
	if err != nil {
		return "", err
	}

	if kind == filetype.Unknown {
		return "", fmt.Errorf("unknown file type for %s", file.Name())
	}
	return kind.MIME.Value, nil
}
