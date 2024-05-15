package handlers

import (
	"bufio"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/Dewberry/mcat-ras/config"
	"github.com/Dewberry/mcat-ras/tools"
	"github.com/Dewberry/s3api/blobstore"

	"github.com/dewberry/gdal"
	"github.com/go-errors/errors" // warning: replaces standard errors
	"github.com/labstack/echo/v4"
)

// GeospatialData godoc
// @Summary Extract geospatial data
// @Description Extract geospatial data from a RAS model given an s3 key
// @Tags MCAT
// @Accept json
// @Produce json
// @Param definition_file query string true "/models/ras/CHURCH HOUSE GULLY/CHURCH HOUSE GULLY.prj"
// @Success 200 {object} interface{}
// @Failure 500 {object} SimpleResponse
// @Router /geospatialdata [get]
func GeospatialData(ac *config.APIConfig) echo.HandlerFunc {
	return func(c echo.Context) error {

		definitionFile := c.QueryParam("definition_file")
		bucket := c.QueryParam("bucket")
		if definitionFile == "" {
			return c.JSON(http.StatusBadRequest, "Missing query parameter: `definition_file`")
		}
		s3Ctrl, err := ac.Bh.GetController(bucket)
		if err != nil {
			errMsg := fmt.Errorf("error getting S3 controller: %s", err.Error())
			return c.JSON(http.StatusInternalServerError, errMsg.Error())
		}

		if !isAModel(s3Ctrl, bucket, definitionFile) {
			return c.JSON(http.StatusBadRequest, definitionFile+" is not a valid RAS prj file.")
		}

		if !isGeospatial(s3Ctrl, bucket, definitionFile) {
			return c.JSON(http.StatusBadRequest, definitionFile+" is not geospatial.")
		}

		data, err := geospatialData(s3Ctrl, bucket, definitionFile, ac.DestinationCRS)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, SimpleResponse{http.StatusInternalServerError, fmt.Sprintf("Go error encountered: %v", err.Error()), err.(*errors.Error).ErrorStack()})
		}

		return c.JSON(http.StatusOK, data)
	}
}

func geospatialData(s3Ctrl *blobstore.S3Controller, bucket, definitionFile string, destinationCRS int) (tools.GeoData, error) {
	gd := tools.GeoData{Features: make(map[string]tools.Features), Georeference: destinationCRS}

	mfiles, err := modFiles(s3Ctrl, bucket, definitionFile)
	if err != nil {
		return gd, errors.Wrap(err, 0)
	}

	projecFile := strings.TrimSuffix(definitionFile, ".prj") + ".projection"
	proj, err := getProjection(s3Ctrl, bucket, projecFile)
	if err != nil {
		return gd, errors.Wrap(err, 0)
	}

	for _, fp := range mfiles {

		ext := filepath.Ext(fp)

		switch {

		case tools.RasRE.Geom.MatchString(ext):

			if err := tools.GetGeospatialData(&gd, s3Ctrl, bucket, fp, proj, destinationCRS); err != nil {
				return gd, errors.Wrap(err, 0)
			}

		}
	}

	return gd, nil
}

func getProjection(s3Ctrl *blobstore.S3Controller, bucket, fn string) (string, error) {

	f, err := s3Ctrl.FetchObjectContent(bucket, fn)
	if err != nil {
		return "", err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	sc.Scan()
	line := sc.Text()

	sourceSpRef := gdal.CreateSpatialReference(line)

	return line, sourceSpRef.Validate()
}
