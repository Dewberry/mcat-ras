package handlers

import (
	"fmt"
	"net/http"

	// warning: replaces standard errors
	"github.com/Dewberry/s3api/blobstore"
	"github.com/labstack/echo/v4"
)

// ModelType godoc
// @Summary Extract the model type
// @Description Extract the model type given an s3 key
// @Tags MCAT
// @Accept json
// @Produce json
// @Param definition_file query string true "/models/ras/CHURCH HOUSE GULLY/CHURCH HOUSE GULLY.prj"
// @Success 200 {string} string "RAS"
// @Failure 500 {object} SimpleResponse
// @Router /modeltype [get]
func ModelType(bh *blobstore.BlobHandler) echo.HandlerFunc {
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

		return c.JSON(http.StatusOK, "RAS")
	}
}
