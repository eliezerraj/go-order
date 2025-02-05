package service

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/go-order/internal/adapter/bucket"
	"github.com/go-order/internal/adapter/event"
	"github.com/go-order/internal/core"
	"github.com/go-order/internal/lib"
	"github.com/go-order/internal/repository/dynamo"
	"github.com/go-order/internal/repository/storage"
)

var childLogger = log.With().Str("service", "service").Logger()

type WorkerService struct {
	workerRepo		*storage.WorkerRepository
	producerWorker	event.EventNotifier
	workerDynamo	*dynamo.DynamoRepository
	bucketWorker	*bucket.BucketWorker
}

func NewWorkerService( 	workerRepo 		*storage.WorkerRepository,
						eventNotifier	event.EventNotifier,
						workerDynamo	*dynamo.DynamoRepository,
						bucketWorker	*bucket.BucketWorker) *WorkerService{
	childLogger.Debug().Msg("NewWorkerService")

	return &WorkerService{
		workerRepo:		workerRepo,
		producerWorker: eventNotifier,
		workerDynamo: 	workerDynamo,
		bucketWorker: bucketWorker,
	}
}

func (s WorkerService) Get(ctx context.Context, order *core.Order) (*core.Order, error){
	childLogger.Debug().Msg("Get")

	span := lib.Span(ctx, "service.Get")
	defer span.End()

	res, err := s.workerRepo.Get(ctx, order)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (s WorkerService) Add(ctx context.Context, order *core.Order) (*core.Order, error){
	childLogger.Debug().Msg("Add")

	span := lib.Span(ctx, "service.Add")
	tx, conn, err := s.workerRepo.StartTx(ctx)
	if err != nil {
		return nil, err
	}
	
	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		} else {
			tx.Commit(ctx)
		}
		s.workerRepo.ReleaseTx(conn)
		span.End()
	}()

	order.Status = "PENDING"
	res, err := s.workerRepo.Add(ctx, tx, order)
	if err != nil {
		return nil, err
	}

	order.ID = res.ID
	order.CreateAt = res.CreateAt
	eventData := core.EventData{Order: order}
	event := core.Event{
		EventDate: time.Now(),
		EventType: "NO_CRQS",
		EventData:	&eventData,	
	}
	
	res, err = s.workerDynamo.Add(ctx, *order)
	if err != nil {
		return nil, err
	}
	
	err = s.bucketWorker.PutObject(ctx, order.OrderID, *order)
	if err != nil {
		return nil, err
	}

	err = s.producerWorker.Producer(ctx, event)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (s WorkerService) AddAsync(ctx context.Context, order *core.Order) (*core.Order, error){
	childLogger.Debug().Msg("AddAsync")

	span := lib.Span(ctx, "service.AddAsync")
	defer span.End()

	order.Status = "PENDING"
	eventData := core.EventData{Order: order}
	event := core.Event{
		EventDate: time.Now(),
		EventType: "CQRS",
		EventData:	&eventData,	
	}
	
	err := s.producerWorker.Producer(ctx, event)
	if err != nil {
		return nil, err
	}

	return order, nil
}

func (s WorkerService) UploadImage(ctx context.Context, oderFile *core.OrderFile) (bool, error){
	childLogger.Debug().Msg("UploadImage")

	span := lib.Span(ctx, "service.UploadImage")
	defer span.End()

	err := s.bucketWorker.PutImageObject(ctx, oderFile.Name, &oderFile.File)
	if err != nil {
		return false, err
	}

	return true, nil
}