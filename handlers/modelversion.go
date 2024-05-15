package handlers

import (
	"bufio"
	"fmt"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Dewberry/mcat-ras/tools"
	"github.com/Dewberry/s3api/blobstore"

	// warning: replaces standard errors
	"github.com/labstack/echo/v4"
)

// ModelVersion godoc
// @Summary Extract the RAS model version
// @Description Extract the RAS model version given an s3 key
// @Tags MCAT
// @Accept json
// @Produce json
// @Param definition_file query string true "/models/ras/CHURCH HOUSE GULLY/CHURCH HOUSE GULLY.prj"
// @Success 200 {string} string "4.0"
// @Failure 500 {object} SimpleResponse
// @Router /modelversion [get]
func ModelVersion(bh *blobstore.BlobHandler) echo.HandlerFunc {
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

		version, err := getVersions(s3Ctrl, bucket, definitionFile)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, err.Error())
		}

		return c.JSON(http.StatusOK, version)
	}
}

func modFiles(s3Ctrl *blobstore.S3Controller, bucket, definitionFile string) ([]string, error) {
	mFiles := make([]string, 0)
	prefix := filepath.Dir(definitionFile) + "/"

	files, err := tools.GetListWithDetail(s3Ctrl, bucket, prefix, false)
	if err != nil {
		return mFiles, err
	}

	for _, file := range *files {
		// get only files that share the same base name or .prj files for projection
		// rational behind .prj file is that there can be a shp file in the same level of Hec-RAS
		// providing potential projection
		if strings.HasPrefix(filepath.Join(file.Path, file.Name), strings.TrimSuffix(definitionFile, "prj")) || filepath.Ext(file.Name) == ".prj" {
			mFiles = append(mFiles, filepath.Join(file.Path, file.Name))
		}
	}

	return mFiles, nil
}

func pullVersion(s3Ctrl *blobstore.S3Controller, bucket, fp string) (string, error) {
	f, err := s3Ctrl.FetchObjectContent(bucket, fp)
	if err != nil {
		return "", err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	var line string
	for sc.Scan() {

		line = sc.Text()

		match, err := regexp.MatchString("Program Version=", line)
		if err != nil {
			return "", err
		}

		if match {
			return strings.Split(line, "=")[1], nil
		}
	}

	return "", fmt.Errorf("unable to find program version in file %s", fp)
}

func getVersions(s3Ctrl *blobstore.S3Controller, bucket, definitionFile string) (string, error) {
	var version string

	mFiles, err := modFiles(s3Ctrl, bucket, definitionFile)
	if err != nil {
		return version, err
	}

	for _, fp := range mFiles {

		ext := filepath.Ext(fp)

		if tools.RasRE.Plan.MatchString(ext) ||
			tools.RasRE.Geom.MatchString(ext) ||
			tools.RasRE.AllFlow.MatchString(ext) {
			ver, err := pullVersion(s3Ctrl, bucket, fp)
			if err != nil {
				fmt.Println(err)
			} else {
				version += fmt.Sprintf("%s: %s, ", ext, ver)
			}
		}
	}

	if len(version) >= 2 {
		version = version[0 : len(version)-2]
	}

	return version, nil
}
