package handlers

import (
	"fmt"
	"net/http"
	"path/filepath"
	"sync"

	"github.com/Dewberry/mcat-ras/config"
	"github.com/Dewberry/mcat-ras/tools"
	"github.com/Dewberry/s3api/blobstore"

	"github.com/go-errors/errors" // warning: replaces standard errors
	"github.com/labstack/echo/v4"
)

type SimpleResponse struct {
	Status     int
	Message    string
	StackTrace string
}

// ForcingData godoc
// @Summary Extract forcing data from flow files
// @Description forcing data from a RAS model given an s3 key
// @Tags MCAT
// @Accept json
// @Produce json
// @Param definition_file query string true "/models/ras/CHURCH HOUSE GULLY/CHURCH HOUSE GULLY.prj"
// @Success 200 {object} interface{}
// @Failure 500 {object} SimpleResponse
// @Router /forcingdata [get]
func ForcingData(ac *config.APIConfig) echo.HandlerFunc {
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
			return c.JSON(http.StatusBadRequest, definitionFile+" is not a valid Ras prj file.")
		}

		data, err := forcingData(s3Ctrl, bucket, definitionFile)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, SimpleResponse{http.StatusInternalServerError, fmt.Sprintf("Go error encountered: %v", err.Error()), err.(*errors.Error).ErrorStack()})
		}

		return c.JSON(http.StatusOK, data)
	}
}

func forcingData(s3Ctrl *blobstore.S3Controller, bucket, definitionFile string) (tools.ForcingData, error) {
	fd := tools.ForcingData{
		Steady:   make(map[string]tools.SteadyData),
		Unsteady: make(map[string]tools.UnsteadyData),
	}

	mfiles, err := modFiles(s3Ctrl, bucket, definitionFile)
	if err != nil {
		return fd, errors.Wrap(err, 0)
	}
	fFiles := []string{}

	for _, fp := range mfiles {
		ext := filepath.Ext(fp)
		if tools.RasRE.AllFlow.MatchString(ext) {
			fFiles = append(fFiles, fp)
		}
	}

	c := make(chan error, len(fFiles))
	var mu sync.Mutex

	for _, fp := range fFiles {
		go tools.GetForcingData(&fd, s3Ctrl, bucket, fp, c, &mu)
	}

	for i := 0; i < len(fFiles); i++ {
		err = <-c
		if err != nil {
			return fd, err
		}
	}

	return fd, nil
}
