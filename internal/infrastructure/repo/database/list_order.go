package database

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/go-order/internal/domain/model"
	"github.com/go-order/shared/erro"

	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/codes"
)

// About list orders
func (w *WorkerRepository) ListOrder(ctx context.Context,
									order *model.Order) (*[]model.Order, error){
	w.logger.Info().
			Ctx(ctx).
			Str("func","ListOrder").Send()

	// trace
	ctx, span := w.tracerProvider.SpanCtx(ctx, "database.ListOrder", trace.SpanKindInternal)
	defer span.End()

	// db connection
	conn, err := w.DatabasePG.Acquire(ctx)
	if err != nil {
		span.RecordError(err) 
        span.SetStatus(codes.Error, err.Error())
		w.logger.Error().
				Ctx(ctx).
				Err(err).Send()
		return nil, fmt.Errorf("FAILED to acquire database connection: %w", err)
	}
	defer w.DatabasePG.Release(conn)

	// Query and Execute
	query := `select	o.id,
						o.transaction_id,
						o.order_date,
						o.status,
						o.currency,
						o.amount,
						o.address,
						o.user_id,
						o.fk_cart_id,
						o.created_at,
						o.updated_at
				from public.order o
				where o.user_id = $1`

	rows, err := conn.Query(ctx, 
							query, 
							order.User)
	if err != nil {
		span.RecordError(err) 
        span.SetStatus(codes.Error, err.Error())
		w.logger.Error().
				Ctx(ctx).
				Err(err).Send()
		return nil, fmt.Errorf("FAILED to query order: %w", err)
	}
	defer rows.Close()
	
    if err := rows.Err(); err != nil {
		span.RecordError(err) 
        span.SetStatus(codes.Error, err.Error())
		w.logger.Error().
				Ctx(ctx).
				Err(err).Msg("error iterating order rows")
        return nil, fmt.Errorf("error iterating order rows: %w", err)
    }

	var orders []model.Order
	resOrder := model.Order{}
	resCart := model.Cart{}

	var nullOrderUpdatedAt sql.NullTime

	for rows.Next() {
		err := rows.Scan(	
							&resOrder.ID, 
							&resOrder.Transaction, 
							&resOrder.Date,
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
			span.RecordError(err) 
			span.SetStatus(codes.Error, err.Error())
			w.logger.Error().
				Ctx(ctx).
				Err(err).Send()
			return nil, fmt.Errorf("FAILED to scan row: %w", err)
        }

		resOrder.UpdatedAt = w.pointerTime(nullOrderUpdatedAt)

		resOrder.Cart = resCart
		orders = append(orders, resOrder)
	}

	if len(orders) == 0 {
		w.logger.Warn().
			Ctx(ctx).
			Err(erro.ErrNotFound).
			Interface("order.User",order.User).Send()

		return nil, erro.ErrNotFound
	}
		
	return &orders, nil
}