package service

import (
	"context"
	"time"
	"github.com/rs/zerolog/log"

	"github.com/go-order/internal/lib"
	//"github.com/go-order/internal/erro"
	"github.com/go-order/internal/adapter/event"
	"github.com/go-order/internal/core"
	"github.com/go-order/internal/repository/storage"
)

var childLogger = log.With().Str("service", "service").Logger()

type WorkerService struct {
	workerRepo		*storage.WorkerRepository
	producerWorker	event.EventNotifier
}

func NewWorkerService( 	workerRepo 		*storage.WorkerRepository,
						eventNotifier	event.EventNotifier) *WorkerService{
	childLogger.Debug().Msg("NewWorkerService")

	return &WorkerService{
		workerRepo:		 	workerRepo,
		producerWorker: 	eventNotifier,
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

	res, err := s.workerRepo.Add(ctx, tx, order)
	if err != nil {
		return nil, err
	}

	order.ID = res.ID
	order.CreateAt = res.CreateAt
	eventData := core.EventData{Order: order}
	event := core.Event{
		EventDate: time.Now(),
		EventData:	&eventData,	
	}
	
	err = s.producerWorker.Producer(ctx, event)
	if err != nil {
		return nil, err
	}

	return res, nil
}