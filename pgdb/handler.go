package pgdb

import (
	"fmt"
	"net/http"

	"github.com/Dewberry/mcat-ras/config"
	"github.com/Dewberry/mcat-ras/handlers"

	"github.com/go-errors/errors" // warning: replaces standard errors
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
)

// UpsertRasModel ...
func UpsertRasModel(ac *config.APIConfig, db *sqlx.DB) echo.HandlerFunc {
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

		err = upsertModelInfo(s3Ctrl, definitionFile, bucket, db)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, handlers.SimpleResponse{Status: http.StatusInternalServerError, Message: fmt.Sprintf("Go error encountered: %v", err.Error()), StackTrace: err.(*errors.Error).ErrorStack()})
		}

		return c.JSON(http.StatusOK, "Successfully uploaded model information for "+definitionFile)
	}
}

// UpsertRasGeometry ...
func UpsertRasGeometry(ac *config.APIConfig, db *sqlx.DB) echo.HandlerFunc {
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

		err = upsertModelGeometry(db, s3Ctrl, bucket, definitionFile, ac.DestinationCRS)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, handlers.SimpleResponse{Status: http.StatusInternalServerError, Message: fmt.Sprintf("Go error encountered: %v", err.Error()), StackTrace: err.(*errors.Error).ErrorStack()})
		}

		return c.JSON(http.StatusOK, "Successfully uploaded model geometry for "+definitionFile)
	}
}

// VacuumRasViews ...
func VacuumRasViews(db *sqlx.DB) echo.HandlerFunc {
	return func(c echo.Context) error {

		for _, query := range vacuumQuery {
			_, err := db.Exec(query)
			if err != nil {
				return c.JSON(http.StatusInternalServerError, handlers.SimpleResponse{Status: http.StatusInternalServerError, Message: fmt.Sprintf("Go error encountered: %v", err.Error()), StackTrace: err.(*errors.Error).ErrorStack()})
			}
		}

		return c.JSON(http.StatusOK, "Ras tables vacuumed successfully.")
	}
}

// RefreshRasViews ...
func RefreshRasViews(db *sqlx.DB) echo.HandlerFunc {
	return func(c echo.Context) error {

		for _, query := range refreshViewsQuery {
			_, err := db.Exec(query)
			if err != nil {
				return c.JSON(http.StatusInternalServerError, handlers.SimpleResponse{Status: http.StatusInternalServerError, Message: fmt.Sprintf("Go error encountered: %v", err.Error()), StackTrace: err.(*errors.Error).ErrorStack()})
			}
		}

		return c.JSON(http.StatusOK, "Ras materialized views refreshed successfully.")
	}
}
