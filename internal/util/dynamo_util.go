package util

import(
	"os"

	"github.com/joho/godotenv"
	"github.com/go-order/internal/core"
)

func GetDynamoEnv() core.DatabaseDynamo {
	childLogger.Debug().Msg("GetDynamoEnv")

	err := godotenv.Load(".env")
	if err != nil {
		childLogger.Info().Err(err).Msg("env file not found !!!")
	}
	
	var databaseDynamo	core.DatabaseDynamo

	if os.Getenv("ORDER_TABLE_NAME") !=  "" {
		databaseDynamo.OrderTableName = os.Getenv("ORDER_TABLE_NAME")
	}
	if os.Getenv("AWS_REGION") !=  "" {
		databaseDynamo.AwsRegion = os.Getenv("AWS_REGION")
	}

	return databaseDynamo
}