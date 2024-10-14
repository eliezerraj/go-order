package util

import(
	"os"

	"github.com/joho/godotenv"
	"github.com/go-order/internal/core"
)

func GetBucketEnv() core.BucketConfig {
	childLogger.Debug().Msg("GetBucketEnv")

	err := godotenv.Load(".env")
	if err != nil {
		childLogger.Info().Err(err).Msg("env file not found !!!")
	}
	
	var bucketConfig	core.BucketConfig
	
	if os.Getenv("BUCKET_NAME") !=  "" {
		bucketConfig.BucketNameKey = os.Getenv("BUCKET_NAME")
	}
	if os.Getenv("FILE_PATH") !=  "" {
		bucketConfig.FilePath = os.Getenv("FILE_PATH")
	}
	if os.Getenv("AWS_REGION") !=  "" {	
		bucketConfig.AwsRegion = os.Getenv("AWS_REGION")
	}

	return bucketConfig
}