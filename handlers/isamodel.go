package handlers

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/Dewberry/mcat-ras/tools"
	"github.com/Dewberry/s3api/blobstore"

	"github.com/labstack/echo/v4"
)

// IsAModel godoc
// @Summary Check if the given key is a RAS model
// @Description Check if the given key is a RAS model
// @Tags MCAT
// @Accept json
// @Produce json
// @Param definition_file query string true "/models/ras/CHURCH HOUSE GULLY/CHURCH HOUSE GULLY.prj"
// @Success 200 {object} bool
// @Router /isamodel [get]
func IsAModel(bh *blobstore.BlobHandler) echo.HandlerFunc {
	return func(c echo.Context) error {

		definitionFile := c.QueryParam("definition_file")
		bucket := c.QueryParam("bucket")
		if definitionFile == "" {
			return c.JSON(http.StatusBadRequest, "Missing query parameter: `definition_file`")
		}
		s3Ctrl, err := bh.GetController(bucket)
		if err != nil {
			errMsg := fmt.Errorf("error getting S3 controller: %s", err.Error())
			return c.JSON(http.StatusInternalServerError, errMsg.Error())
		}

		return c.JSON(http.StatusOK, isAModel(s3Ctrl, bucket, definitionFile))
	}
}

func isAModel(s3Ctrl *blobstore.S3Controller, bucket, definitionFile string) bool {
	if filepath.Ext(definitionFile) != ".prj" {
		return false
	}

	firstLine, err := tools.ReadFirstLine(s3Ctrl, bucket, definitionFile)
	if err != nil {
		return false
	}

	if !strings.Contains(firstLine, "Proj Title=") {
		return false
	}

	files, err := modFiles(s3Ctrl, bucket, definitionFile)
	if err != nil {
		return false
	}

	for _, f := range files {
		if filepath.Ext(f)[0:2] == ".g" {
			return true
		}
	}

	return false
}
