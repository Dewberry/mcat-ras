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

	"github.com/USACE/filestore" // warning: replaces standard errors
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
func ModelVersion(fs *filestore.FileStore) echo.HandlerFunc {
	return func(c echo.Context) error {

		definitionFile := c.QueryParam("definition_file")
		if definitionFile == "" {
			return c.JSON(http.StatusBadRequest, "Missing query parameter: `definition_file`")
		}

		if !isAModel(fs, definitionFile) {
			return c.JSON(http.StatusBadRequest, definitionFile+" is not a valid RAS prj file.")
		}

		version, err := getVersions(definitionFile, *fs)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, err.Error())
		}

		return c.JSON(http.StatusOK, version)
	}
}
func modFiles(s3ctrl *blobstore.S3Controller, bucket, definitionFile string) ([]string, error) {
	mFiles := make([]string, 0)
	prefix := filepath.Dir(definitionFile) + "/"

	result, err := s3ctrl.GetList(bucket, prefix, false)
	if err != nil {
		return mFiles, err
	}

	// Process directories
	for _, cp := range result.CommonPrefixes {
		dirPath := *cp.Prefix
		fileName := filepath.Base(dirPath)
		if strings.HasPrefix(filepath.Join(dirPath, fileName), strings.TrimSuffix(definitionFile, "prj")) {
			mFiles = append(mFiles, filepath.Join(dirPath, fileName))
		}
	}

	// Process files
	for _, object := range result.Contents {
		filePath := *object.Key
		fileName := filepath.Base(filePath)
		if strings.HasPrefix(filePath, strings.TrimSuffix(definitionFile, "prj")) || filepath.Ext(fileName) == ".prj" {
			mFiles = append(mFiles, filePath)
		}
	}

	return mFiles, nil
}

func pullVersion(fp string, fs filestore.FileStore) (string, error) {
	f, err := fs.GetObject(fp)
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

func getVersions(definitionFile string, fs filestore.FileStore) (string, error) {
	var version string

	mFiles, err := modFiles(definitionFile, fs)
	if err != nil {
		return version, err
	}

	for _, fp := range mFiles {

		ext := filepath.Ext(fp)

		if tools.RasRE.Plan.MatchString(ext) ||
			tools.RasRE.Geom.MatchString(ext) ||
			tools.RasRE.AllFlow.MatchString(ext) {
			ver, err := pullVersion(fp, fs)
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
