package utils

import (
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"time"
)

func UploadImageToB2(r *http.Request, field string) (string, error) {
	ctx := r.Context()

	file, fileHeader, err := r.FormFile(field)
	if err != nil {
		return "", err
	}
	defer file.Close()

	ext := filepath.Ext(fileHeader.Filename)
	objectName := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)

	bucket, err := B2Client.Bucket(ctx, "GoEatspartner")
	if err != nil {
		return "", err
	}

	obj := bucket.Object(objectName)
	writer := obj.NewWriter(ctx)

	if _, err := io.Copy(writer, file); err != nil {
		writer.Close()
		return "", err
	}

	if err := writer.Close(); err != nil {
		return "", err
	}

	url := obj.URL()

	return url, nil
}
