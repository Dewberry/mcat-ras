package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/Dewberry/s3api/blobstore"
	"github.com/USACE/filestore"
	"github.com/labstack/gommon/log"
)

type APIConfig struct {
	Host           string
	Port           int
	FileStore      *filestore.FileStore
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
	config.FileStore = FileStoreInit(os.Getenv("STORE_TYPE"))
	config.Bh = blobstoreInit()
	config.DestinationCRS = 4326
	return config
}

// FileStoreInit initializes the filestore object
func FileStoreInit(store string) *filestore.FileStore {

	var fs filestore.FileStore
	var err error
	switch store {
	case "LOCAL":
		fs, err = filestore.NewFileStore(filestore.BlockFSConfig{})
		if err != nil {
			panic(err)
		}
	case "S3":
		config := filestore.S3FSConfig{
			S3Id:     os.Getenv("AWS_ACCESS_KEY_ID"),
			S3Key:    os.Getenv("AWS_SECRET_ACCESS_KEY"),
			S3Region: os.Getenv("AWS_DEFAULT_REGION"),
			S3Bucket: os.Getenv("S3_BUCKET"),
		}

		fs, err = filestore.NewFileStore(config)
		if err != nil {
			panic(err)
		}
	}
	return &fs
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
