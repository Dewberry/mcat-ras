package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/Dewberry/s3api/blobstore"
	"github.com/labstack/gommon/log"
)

type APIConfig struct {
	Host           string
	Port           int
	Bh             *blobstore.BlobHandler
	DestinationCRS int
}

// Address tells the application where to run the api out of
func (app *APIConfig) Address() string {
	return fmt.Sprintf("%s:%d", app.Host, app.Port)
}

// Init initializes the API's configuration
func Init() *APIConfig {
	config := new(APIConfig)
	config.Host = "" // 0.0.0.0
	config.Port = 5600
	config.Bh = blobstoreInit()
	config.DestinationCRS = 4326
	return config
}

func blobstoreInit() *blobstore.BlobHandler {
	var authLvl int
	var err error
	authLvlString := os.Getenv("AUTH_LEVEL")
	if authLvlString == "" {
		authLvl = 0
		log.Warn("Fine Grained Access Control disabled")
	} else {
		authLvl, err = strconv.Atoi(authLvlString)
		if err != nil {
			log.Fatalf("could not convert AUTH_LEVEL env variable to integer: %v", err)
		}
	}
	bh, err := blobstore.NewBlobHandler(".env.json", authLvl)
	if err != nil {
		errMsg := fmt.Errorf("failed to initialize a blobhandler %s", err.Error())
		log.Fatal(errMsg)
	}
	return bh
}
