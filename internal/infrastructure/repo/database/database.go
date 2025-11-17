package database

import (
		"context"
		"errors"
		"database/sql"

		"github.com/rs/zerolog"
		"github.com/jackc/pgx/v5"

		"github.com/go-order/shared/erro"
		"github.com/go-order/internal/domain/model"

		go_core_otel_trace "github.com/eliezerraj/go-core/otel/trace"
		go_core_db_pg "github.com/eliezerraj/go-core/database/postgre"
)

var tracerProvider go_core_otel_trace.TracerProvider

type WorkerRepository struct {
	DatabasePG *go_core_db_pg.DatabasePGServer
	logger		*zerolog.Logger
}

// Above new worker
func NewWorkerRepository(databasePG *go_core_db_pg.DatabasePGServer,
						appLogger *zerolog.Logger) *WorkerRepository{
	logger := appLogger.With().
						Str("package", "repo.database").
						Logger()
	logger.Info().
			Str("func","NewWorkerRepository").Send()

	return &WorkerRepository{
		DatabasePG: databasePG,
		logger: &logger,
	}
}

// Above get stats from database
func (w *WorkerRepository) Stat(ctx context.Context) (go_core_db_pg.PoolStats){
	w.logger.Info().
			Str("func","Stat").Send()
	
	stats := w.DatabasePG.Stat()

	resPoolStats := go_core_db_pg.PoolStats{
		AcquireCount:         stats.AcquireCount(),
		AcquiredConns:        stats.AcquiredConns(),
		CanceledAcquireCount: stats.CanceledAcquireCount(),
		ConstructingConns:    stats.ConstructingConns(),
		EmptyAcquireCount:    stats.EmptyAcquireCount(),
		IdleConns:            stats.IdleConns(),
		MaxConns:             stats.MaxConns(),
		TotalConns:           stats.TotalConns(),
	}

	return resPoolStats
}

// About create a order
func (w* WorkerRepository) AddOrder(ctx context.Context, 
									tx pgx.Tx, 
									order *model.Order) (*model.Order, error){
	// trace
	ctx, span := tracerProvider.SpanCtx(ctx, "database.AddOrder")
	defer span.End()

	w.logger.Info().
			Ctx(ctx).
			Str("func","AddOrder").Send()

	conn, err := w.DatabasePG.Acquire(ctx)
	if err != nil {
		w.logger.Error().
				Ctx(ctx).
				Err(err).Send()
		return nil, errors.New(err.Error())
	}
	defer w.DatabasePG.Release(conn)

	//Prepare
	var id int

	// Query Execute
	query := `INSERT INTO public.order (transaction_id,
										fk_cart_id,
										fk_clearance_id,
										user_id,
										status,
										currency,
										amount,
										address,
										created_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id`

	row := tx.QueryRow(	ctx, 
						query,			
						order.Transaction,
						order.Cart.ID,
						order.Payment.ID,
						order.User,
						order.Status,
						order.Currency,
						order.Amount,
						order.Address,
						order.CreatedAt)
						
	if err := row.Scan(&id); err != nil {
		w.logger.Error().
				Ctx(ctx).
				Err(err).Send()
		return nil, errors.New(err.Error())
	}

	// Set PK
	order.ID = id
	
	return order , nil
}

// About get a order
func (w *WorkerRepository) GetOrder(ctx context.Context,
									order *model.Order) (*model.Order, error){
	// trace
	ctx, span := tracerProvider.SpanCtx(ctx, "database.GetOrder")
	defer span.End()

	w.logger.Info().
			Ctx(ctx).
			Str("func","GetOrder").Send()

	// db connection
	conn, err := w.DatabasePG.Acquire(ctx)
	if err != nil {
		w.logger.Error().
				Ctx(ctx).
				Err(err).Send()
		return nil, errors.New(err.Error())
	}
	defer w.DatabasePG.Release(conn)

	// Query and Execute
	query := `select	o.id,
						o.transaction_id,
						o.status,
						o.currency,
						o.amount,
						o.address,
						o.created_at,
						o.updated_at,
						ca.id,
						ca.created_at,
						ca.updated_at,
						ca_it.id,
						ca_it.status,
						ca_it.currency,
						ca_it.quantity,
						ca_it.discount,
						ca_it.price,
						ca_it.created_at,
						ca_it.updated_at						
				from public.order o,
					cart ca,
					cart_item ca_it,
					product p
				where ca.id = o.fk_cart_id
				and ca.id = ca_it.fk_cart_id
				and p.id = ca_it.fk_product_id
				and o.id = $1`

	rows, err := conn.Query(ctx, 
							query, 
							order.ID)
	if err != nil {
		w.logger.Error().
				Ctx(ctx).
				Err(err).Send()
		return nil, errors.New(err.Error())
	}
	defer rows.Close()
	
    if err := rows.Err(); err != nil {
		w.logger.Error().
				Ctx(ctx).
				Err(err).Msg("fatal error closing rows")
        return nil, errors.New(err.Error())
    }

	resOrder := model.Order{}
	resCart := model.Cart{}
	resCartItem := model.CartItem{}
	listCartItem := []model.CartItem{}
	//resPayment := model.Payment{}

	var nullOrderUpdatedAt sql.NullTime
	var nullCartUpdatedAt sql.NullTime
	var nullCartItemUpdatedAt sql.NullTime

	for rows.Next() {
		err := rows.Scan(	
							&resOrder.ID, 
							&resOrder.Transaction, 
							&resOrder.Status, 							
							&resOrder.Currency, 
							&resOrder.Amount,
							&resOrder.Address,
							&resOrder.CreatedAt,
							&nullOrderUpdatedAt,

							&resCart.ID,
							&resCart.CreatedAt,
							&nullCartUpdatedAt,
							
							&resCartItem.ID, 
							&resCartItem.Status, 
							&resCartItem.Currency, 
							&resCartItem.Quantity, 
							&resCartItem.Discount, 
							&resCartItem.Price, 
							&resCartItem.CreatedAt,
							&nullCartItemUpdatedAt,
							
						)
		if err != nil {
			w.logger.Error().
					Ctx(ctx).
					Err(err).Send()
			return nil, errors.New(err.Error())
        }

		if nullOrderUpdatedAt.Valid {
        	resOrder.UpdatedAt = &nullOrderUpdatedAt.Time
    	} else {
			resOrder.UpdatedAt = nil
		}

		if nullCartUpdatedAt.Valid {
        	resCart.UpdatedAt = &nullCartUpdatedAt.Time
    	} else {
			resCart.UpdatedAt = nil
		}

		if nullCartItemUpdatedAt.Valid {
        	resOrder.UpdatedAt = &nullCartItemUpdatedAt.Time
    	} else {
			resOrder.UpdatedAt = nil
		}

		listCartItem = append(listCartItem, resCartItem) 
		resCart.CartItem = &listCartItem
		resOrder.Cart = resCart
	}

	if resOrder == (model.Order{}) {
		w.logger.Warn().
				Ctx(ctx).
				Err(err).Send()
		return nil, erro.ErrNotFound
	}
		
	return &resOrder, nil
}