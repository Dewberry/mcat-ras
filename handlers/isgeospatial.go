package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/Dewberry/s3api/blobstore" // warning: replaces standard errors
	"github.com/labstack/echo/v4"
)

// IsGeospatial godoc
// @Summary Check if the RAS model has geospatial information
// @Description Check if the RAS model has geospatial information
// @Tags MCAT
// @Accept json
// @Produce json
// @Param definition_file query string true "/models/ras/CHURCH HOUSE GULLY/CHURCH HOUSE GULLY.prj"
// @Success 200 {object} bool
// @Router /isgeospatial [get]
func IsGeospatial(bh *blobstore.BlobHandler) echo.HandlerFunc {
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

		if !isAModel(s3Ctrl, bucket, definitionFile) {
			return c.JSON(http.StatusBadRequest, definitionFile+" is not a valid RAS prj file.")
		}

		return c.JSON(http.StatusOK, isGeospatial(s3Ctrl, bucket, definitionFile))
	}
}

func isGeospatial(s3Ctrl *blobstore.S3Controller, bucket, definitionFile string) bool {

	modelVersions, err := getVersions(s3Ctrl, bucket, definitionFile)
	if err != nil {
		return false
	}

	for _, version := range strings.Split(modelVersions, ",") {
		if strings.Contains(version, ".g") {
			geomVersion := strings.TrimSpace(strings.Split(version, ":")[1])
			v, err := strconv.ParseFloat(geomVersion, 64)
			if err != nil {
				fmt.Printf("could not convert the geometry version to a float. prj file: %s\n", definitionFile)
				return false
			}
			if v < 4 {
				fmt.Printf("geometry file version: %f is not geospatial\n", v)
				return false
			}
		}
	}

	return true
}
