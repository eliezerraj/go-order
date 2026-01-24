package database

import (
	"time"
	"fmt"
	"strings"
	"context"
	"database/sql"

	"github.com/rs/zerolog"
	"github.com/jackc/pgx/v5"

	"github.com/go-order/shared/erro"
	"github.com/go-order/internal/domain/model"
	"go.opentelemetry.io/otel/trace"

	go_core_otel_trace "github.com/eliezerraj/go-core/v2/otel/trace"
	go_core_db_pg "github.com/eliezerraj/go-core/v2/database/postgre"
)

type WorkerRepository struct {
	DatabasePG 		*go_core_db_pg.DatabasePGServer
	logger			*zerolog.Logger
	tracerProvider 	*go_core_otel_trace.TracerProvider
}

// Above new worker
func NewWorkerRepository(databasePG *go_core_db_pg.DatabasePGServer,
						appLogger *zerolog.Logger,
						tracerProvider *go_core_otel_trace.TracerProvider) *WorkerRepository{
	logger := appLogger.With().
						Str("package", "repo.database").
						Logger()
	logger.Info().
			Str("func","NewWorkerRepository").Send()

	return &WorkerRepository{
		DatabasePG: databasePG,
		logger: &logger,
		tracerProvider: tracerProvider,
	}
}

// Helper function to convert nullable time to pointer
func (w *WorkerRepository) pointerTime(nt sql.NullTime) *time.Time {
	if nt.Valid {
		return &nt.Time
	}
	return nil
}

// Above get stats from database
func (w *WorkerRepository) Stat(ctx context.Context) (go_core_db_pg.PoolStats){
	w.logger.Info().
			Ctx(ctx).
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

// About create an order
func (w* WorkerRepository) AddOrder(ctx context.Context, 
									tx pgx.Tx, 
									order *model.Order) (*model.Order, error){

	w.logger.Info().
			Ctx(ctx).
			Str("func","AddOrder").Send()

	// trace
	ctx, span := w.tracerProvider.SpanCtx(ctx, "database.AddOrder", trace.SpanKindInternal)
	defer span.End()

	//Prepare
	var id int

	// Query Execute
	query := `INSERT INTO public.order (transaction_id,
										fk_cart_id,
										user_id,
										status,
										currency,
										amount,
										address,
										created_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`

	row := tx.QueryRow(	ctx, 
						query,			
						order.Transaction,
						order.Cart.ID,
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
		return nil, fmt.Errorf("FAILED to scan order ID: %w", err)
	}

	// Set PK
	order.ID = id
	
	return order , nil
}

// About get an order
func (w *WorkerRepository) GetOrder(ctx context.Context,
									order *model.Order) (*model.Order, error){
	w.logger.Info().
			Ctx(ctx).
			Str("func","GetOrder").Send()

	// trace
	ctx, span := w.tracerProvider.SpanCtx(ctx, "database.GetOrder", trace.SpanKindInternal)
	defer span.End()

	// db connection
	conn, err := w.DatabasePG.Acquire(ctx)
	if err != nil {
		w.logger.Error().
				Ctx(ctx).
				Err(err).Send()
		return nil, fmt.Errorf("FAILED to acquire database connection: %w", err)
	}
	defer w.DatabasePG.Release(conn)

	// Query and Execute
	query := `select	o.id,
						o.transaction_id,
						o.status,
						o.currency,
						o.amount,
						o.address,
						o.user_id,
						o.fk_cart_id,
						o.created_at,
						o.updated_at
				from public.order o
				where o.id = $1`

	rows, err := conn.Query(ctx, 
							query, 
							order.ID)
	if err != nil {
		w.logger.Error().
				Ctx(ctx).
				Err(err).Send()
		return nil, fmt.Errorf("FAILED to query order: %w", err)
	}
	defer rows.Close()
	
    if err := rows.Err(); err != nil {
		w.logger.Error().
				Ctx(ctx).
				Err(err).Msg("error iterating order rows")
        return nil, fmt.Errorf("error iterating order rows: %w", err)
    }

	resOrder := model.Order{}
	resCart := model.Cart{}

	var nullOrderUpdatedAt sql.NullTime

	for rows.Next() {
		err := rows.Scan(	
							&resOrder.ID, 
							&resOrder.Transaction, 
							&resOrder.Status, 							
							&resOrder.Currency, 
							&resOrder.Amount,
							&resOrder.Address,
							&resOrder.User,
							&resCart.ID,
							&resOrder.CreatedAt,
							&nullOrderUpdatedAt,
						)
		if err != nil {
			w.logger.Error().
					Ctx(ctx).
					Err(err).Send()
			return nil, err
        }

		resOrder.UpdatedAt = w.pointerTime(nullOrderUpdatedAt)

		resOrder.Cart = resCart
	}

	if resOrder == (model.Order{}) {
		w.logger.Warn().
				Ctx(ctx).
				Err(erro.ErrNotFound).
				Interface("order.ID",order.ID).Send()

		return nil, erro.ErrNotFound
	}
		
	return &resOrder, nil
}

// About get an order, cart, cart item and products
/*func (w *WorkerRepository) GetOrderV1(	ctx context.Context,
										order *model.Order) (*model.Order, error){
	w.logger.Info().
			Ctx(ctx).
			Str("func","GetOrder").Send()

	// trace
	ctx, span := w.tracerProvider.SpanCtx(ctx, "database.GetOrder", trace.SpanKindInternal)
	defer span.End()

	// db connection
	conn, err := w.DatabasePG.Acquire(ctx)
	if err != nil {
		w.logger.Error().
				Ctx(ctx).
				Str("func","GetOrder").
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
						o.user_id,
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
						ca_it.updated_at,
						p.Sku						
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
				Str("func","GetOrder").
				Err(err).Send()
		return nil, errors.New(err.Error())
	}
	defer rows.Close()
	
    if err := rows.Err(); err != nil {
		w.logger.Error().
				Ctx(ctx).
				Str("func","GetOrder").
				Err(err).Msg("fatal error closing rows")
        return nil, errors.New(err.Error())
    }

	resOrder := model.Order{}
	resCart := model.Cart{}
	resCartItem := model.CartItem{}
	listCartItem := []model.CartItem{}
	resProduct := model.Product{}

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
							&resOrder.User,
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
							
							&resProduct.Sku,
						)
		if err != nil {
			w.logger.Error().
					Ctx(ctx).
					Str("func","GetOrder").
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

		resCartItem.Product = resProduct
		listCartItem = append(listCartItem, resCartItem)
		resCart.CartItem = &listCartItem
		resOrder.Cart = resCart
	}

	if resOrder == (model.Order{}) {
		w.logger.Warn().
				Ctx(ctx).
				Str("func","GetOrder").
				Err(erro.ErrNotFound).
				Interface("order.ID",order.ID).Send()
		return nil, erro.ErrNotFound
	}
		
	return &resOrder, nil
}*/

// About update an order
func (w* WorkerRepository) UpdateOrder(ctx context.Context, 
										tx pgx.Tx, 
										order *model.Order) (int64, error){
	w.logger.Info().
			Ctx(ctx).
			Str("func","UpdateOrder").Send()
			
	// trace
	ctx, span := w.tracerProvider.SpanCtx(ctx, "database.UpdateOrder", trace.SpanKindInternal)
	defer span.End()

	// Query Execute
	query := `UPDATE public.order
				SET status = $2,
					updated_at = $3
				WHERE id = $1`

	row, err := tx.Exec(ctx, 
						query,	
						order.ID,			
						order.Status,
						order.UpdatedAt)
	if err != nil {
		w.logger.Error().
				Ctx(ctx).
				Str("func","UpdateOrder").
				Err(err).Send()
		return 0, fmt.Errorf("FAILED to update order: %w", err)
	}
	
	return row.RowsAffected(), nil
}

// About create a outbox
func (w* WorkerRepository) OutboxOrder(	ctx context.Context, 
										tx pgx.Tx, 
										orderOutbox *model.Outbox) (*model.Outbox, error){

	w.logger.Info().
			Ctx(ctx).
			Str("func","OutboxOrder").Send()

	// trace
	ctx, span := w.tracerProvider.SpanCtx(ctx, "database.OutboxOrder", trace.SpanKindInternal)
	defer span.End()

	// Query Execute
	var event_id string
	query := `INSERT INTO public.order_outbox (	event_id,
												event_type,
												event_date,
												transaction_id,
												event_metadata,
												event_data)
				VALUES ($1, $2, $3, $4, $5, $6) RETURNING event_id`

	row := tx.QueryRow(	ctx, 
						query,			
						orderOutbox.ID,
						orderOutbox.Type,
						orderOutbox.CreatedAt,
						orderOutbox.Transaction,
						orderOutbox.Metadata,
						orderOutbox.Data)
						
	if err := row.Scan(&event_id); err != nil {
		if strings.Contains(err.Error(), "duplicate key value violates") {
    		w.logger.Warn().
					 Ctx(ctx).
					 Err(err).Send()
		} else {
			w.logger.Error().
					 Ctx(ctx).
				     Err(err).Send()
		}
		return nil, fmt.Errorf("FAILED to insert order: %w", err)
	}

	orderOutbox.ID = event_id
	
	return orderOutbox , nil
}