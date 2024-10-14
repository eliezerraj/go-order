package webserver

import (
	"context"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/go-order/internal/adapter/bucket"
	"github.com/go-order/internal/adapter/event"
	"github.com/go-order/internal/adapter/event/sqs"
	"github.com/go-order/internal/core"
	"github.com/go-order/internal/handler"
	"github.com/go-order/internal/handler/controller"
	"github.com/go-order/internal/repository/dynamo"
	"github.com/go-order/internal/repository/pg"
	"github.com/go-order/internal/repository/storage"
	"github.com/go-order/internal/service"
	"github.com/go-order/internal/util"
)

var(
	logLevel 	= 	zerolog.DebugLevel
	appServer	core.AppServer
	producerWorker	event.EventNotifier
	bucketWorker	*bucket.BucketWorker
)

func init(){
	log.Debug().Msg("init")
	zerolog.SetGlobalLevel(logLevel)

	infoPod, server := util.GetInfoPod()
	database := util.GetDatabaseEnv()
	configOTEL := util.GetOtelEnv()
	queueConfig := util.GetQueueEnv()
	dynamo := util.GetDynamoEnv()
	bucket := util.GetBucketEnv()

	appServer.InfoPod = &infoPod
	appServer.Database = &database
	appServer.Server = &server
	appServer.ConfigOTEL = &configOTEL
	appServer.QueueConfig = &queueConfig
	appServer.DynamoConfig = &dynamo
	appServer.BucketConfig = &bucket
}

func Server(){
	log.Debug().Msg("----------------------------------------------------")
	log.Debug().Msg("main")
	log.Debug().Msg("----------------------------------------------------")
	log.Debug().Interface("appServer :",appServer).Msg("")
	log.Debug().Msg("----------------------------------------------------")

	ctx, cancel := context.WithTimeout(	context.Background(), 
										time.Duration( appServer.Server.ReadTimeout ) * time.Second)
	defer cancel()

	// Open Database
	count := 1
	var databasePG	pg.DatabasePG
	var err error
	for {
		databasePG, err = pg.NewDatabasePGServer(ctx, appServer.Database)
		if err != nil {
			if count < 3 {
				log.Error().Err(err).Msg("error open Database... trying again !!")
			} else {
				log.Error().Err(err).Msg("fatal error open Database aborting")
				panic(err)
			}
			time.Sleep(3 * time.Second)
			count = count + 1
			continue
		}
		break
	}

	// Create a dynamo repository
	dynamoRepository, err := dynamo.NewDynamoRepository(ctx, *appServer.DynamoConfig)
	if err != nil {
		log.Error().Err(err).Msg("erro connect to dynamo")
	}

	// Setup sqs queue type
	producerWorker, err = sqs.NewNotifierSQS(ctx, appServer.QueueConfig)
	if err != nil {
		log.Error().Err(err).Msg("erro connect to queue")
	}
	// Setup s3
	bucketWorker, err = bucket.NewBucketWorker(ctx, *appServer.BucketConfig)
	if err != nil {
		log.Error().Err(err).Msg("erro connect to s3")
	}
	_ = bucketWorker
	repoDatabase := storage.NewWorkerRepository(databasePG)
	workerService := service.NewWorkerService(&repoDatabase, producerWorker, dynamoRepository, bucketWorker)
	httpWorkerAdapter 	:= controller.NewHttpWorkerAdapter(workerService)
	httpServer 			:= handler.NewHttpAppServer(appServer.Server)
	httpServer.StartHttpAppServer(ctx, &httpWorkerAdapter, &appServer)
}