package handlers

import (
	"fmt"
	"net/http"

	ras "github.com/Dewberry/mcat-ras/tools"
	"github.com/Dewberry/s3api/blobstore"

	"github.com/go-errors/errors" // warning: replaces standard errors
	"github.com/labstack/echo/v4"
)

// Index godoc
// @Summary Index a RAS model
// @Description Extract metadata from a RAS model given an s3 key
// @Tags MCAT
// @Accept json
// @Produce json
// @Param definition_file query string true "/models/ras/CHURCH HOUSE GULLY/CHURCH HOUSE GULLY.prj"
// @Success 200 {object} ras.Model
// @Failure 500 {object} SimpleResponse
// @Router /index [get]
func Index(bh *blobstore.BlobHandler) echo.HandlerFunc {
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

		rm, err := ras.NewRasModel(s3Ctrl, bucket, definitionFile)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, SimpleResponse{http.StatusInternalServerError, fmt.Sprintf("Go error encountered: %v", err.Error()), err.(*errors.Error).ErrorStack()})
		}
		mod := rm.Index()

		return c.JSON(http.StatusOK, mod)
	}
}
