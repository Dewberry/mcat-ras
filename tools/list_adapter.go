package tools

import (
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Dewberry/s3api/blobstore"
)

func GetListWithDetail(s3Ctrl *blobstore.S3Controller, bucket, prefix string, recursive bool) (*[]blobstore.ListResult, error) {
	delimiter := ""
	if !recursive {
		delimiter = "/"
	}

	response, err := s3Ctrl.GetList(bucket, prefix, delimiter != "")
	if err != nil {
		return nil, err
	}

	var result []blobstore.ListResult
	count := 0

	if delimiter != "" {
		for _, cp := range response.CommonPrefixes {
			result = append(result, blobstore.ListResult{
				ID:    count,
				Name:  filepath.Base(strings.TrimSuffix(*cp.Prefix, "/")),
				Path:  *cp.Prefix,
				IsDir: true,
			})
			count++
		}
	}

	for _, object := range response.Contents {
		if *object.Key != prefix {
			result = append(result, blobstore.ListResult{
				ID:       count,
				Name:     filepath.Base(*object.Key),
				Size:     strconv.FormatInt(*object.Size, 10),
				Path:     filepath.Dir(*object.Key) + "/",
				Type:     filepath.Ext(*object.Key),
				IsDir:    false,
				Modified: *object.LastModified,
			})
			count++
		}
	}

	return &result, nil
}
