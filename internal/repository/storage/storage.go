package storage

import (
	"context"
	"time"
	"errors"

	"github.com/go-order/internal/repository/pg"
	"github.com/go-order/internal/core"
	"github.com/go-order/internal/lib"
	"github.com/go-order/internal/erro"

	"github.com/rs/zerolog/log"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var childLogger = log.With().Str("repository.pg", "storage").Logger()

type WorkerRepository struct {
	databasePG pg.DatabasePG
}

func NewWorkerRepository(databasePG pg.DatabasePG) WorkerRepository {
	childLogger.Debug().Msg("NewWorkerRepository")
	return WorkerRepository{
		databasePG: databasePG,
	}
}

func (w WorkerRepository) StartTx(ctx context.Context) (pgx.Tx, *pgxpool.Conn, error) {
	childLogger.Debug().Msg("StartTx")

	span := lib.Span(ctx, "storage.StartTx")
	defer span.End()

	span = lib.Span(ctx, "storage.Acquire")
	conn, err := w.databasePG.Acquire(ctx)
	if err != nil {
		childLogger.Error().Err(err).Msg("error acquire")
		return nil, nil, errors.New(err.Error())
	}
	span.End()
	
	tx, err := conn.Begin(ctx)
    if err != nil {
        return nil, nil ,errors.New(err.Error())
    }

	return tx, conn, nil
}

func (w WorkerRepository) ReleaseTx(connection *pgxpool.Conn) {
	childLogger.Debug().Msg("ReleaseTx")

	defer connection.Release()
}

func (w WorkerRepository) Get(ctx context.Context, order *core.Order) (*core.Order, error){
	childLogger.Debug().Msg("Get")

	span := lib.Span(ctx, "storage.Get")	
	defer span.End()

	conn, err := w.databasePG.Acquire(ctx)
	if err != nil {
		childLogger.Error().Err(err).Msg("error acquire")
		return nil, errors.New(err.Error())
	}
	defer w.databasePG.Release(conn)

	result_query := core.Order{}

	query := `SELECT id, 
					order_id, 
					person_id, 
					status, 
					currency, 
					amount, 
					create_at, 
					update_at, 
					tenant_id 
					FROM public.order 
					WHERE order_id =$1`

	rows, err := conn.Query(ctx, query, order.OrderID)
	if err != nil {
		childLogger.Error().Err(err).Msg("error query statement")
		return nil, errors.New(err.Error())
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan( &result_query.ID, 
							&result_query.OrderID, 
							&result_query.PersonID,
							&result_query.Status,
							&result_query.Currency,
							&result_query.Amount,
							&result_query.CreateAt,
							&result_query.UpdateAt,
							&result_query.TenantID)
		if err != nil {
			childLogger.Error().Err(err).Msg("error scan statement")
			return nil, errors.New(err.Error())
        }
		return &result_query, nil
	}
	
	return nil, erro.ErrNotFound
}

func (w WorkerRepository) Add(ctx context.Context, tx pgx.Tx, order *core.Order) (*core.Order, error){
	childLogger.Debug().Msg("Add")
	childLogger.Debug().Interface("Add : ", order).Msg("")

	span := lib.Span(ctx, "storage.Add")	
	defer span.End()

	conn, err := w.databasePG.Acquire(ctx)
	if err != nil {
		childLogger.Error().Err(err).Msg("error acquire")
		return nil, errors.New(err.Error())
	}
	defer w.databasePG.Release(conn)

	query := `INSERT INTO public.order (order_id, 
										person_id, 
										status, 
										currency, 
										amount, 
										create_at, 
										tenant_id) VALUES($1, $2, $3, $4, $5, $6, $7) RETURNING id`

	order.CreateAt = time.Now()
	row := tx.QueryRow(ctx, query, order.OrderID, 
									order.PersonID,
									order.Status,
									order.Currency,
									order.Amount,
									order.CreateAt,
									order.TenantID)

	var id int
	if err := row.Scan(&id); err != nil {
		childLogger.Error().Err(err).Msg("error queryRow insert")
		return nil, errors.New(err.Error())
	}

	order.ID = id
	return order , nil
}