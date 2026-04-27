package service

import (
	"fmt"
	"time"
	"context"

	"github.com/google/uuid"
	"github.com/go-order/internal/domain/model"
	
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/codes"

	go_core_http "github.com/eliezerraj/go-core/v2/http"
)

// About create a order
func (s *WorkerService) AddOrder(ctx context.Context, 
								 order *model.Order) (*model.Order, error){
	s.logger.Info().
		Ctx(ctx).
		Str("func","AddOrder").Send()

	// trace and log
	ctx, span := s.tracerProvider.SpanCtx(ctx, "service.GetOrder", trace.SpanKindServer)

	// prepare database
	tx, conn, err := s.workerRepository.DatabasePG.StartTx(ctx)
	if err != nil {
		span.RecordError(err) 
        span.SetStatus(codes.Error, err.Error())
		s.logger.Error().
			Ctx(ctx).
			Err(err).Send()
		return nil, err
	}

	// handle connection
	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		} else {
			tx.Commit(ctx)
		}
		s.workerRepository.DatabasePG.ReleaseTx(conn)
		span.End()
	}()

	// create saga orchestration process
	listStepProcess := []model.StepProcess{}

	// prepare headers http for calling services
	headers := s.buildHeaders(ctx)

	// ---------------------- STEP 1 (create a cart) --------------------------
	endpoint, err := s.getServiceEndpoint(0)
	if err != nil {
		span.RecordError(err) 
        span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	// prepare data
	order.Cart.Status = "CART:PENDING"

	httpClientParameter := go_core_http.HttpClientParameter {
		Url:	endpoint.Url + "/cart",
		Method:	"POST",
		Timeout: endpoint.HttpTimeout,
		Headers: &headers,
		Body: order.Cart,
	}

	// call a service via http
	resPayload, err := s.doHttpCall(ctx, httpClientParameter)
	if err != nil {	
		span.RecordError(err) 
        span.SetStatus(codes.Error, err.Error())	
		return nil, err
	}

	cart, err := s.parseCartFromPayload(ctx, resPayload)
	if err != nil {
		span.RecordError(err) 
        span.SetStatus(codes.Error, err.Error())
		s.logger.Error().
			Ctx(ctx).
			Err(err).Send()
		return nil, err
	}

	registerOrchestrationProcess("CART:PENDING:SUCESSFULL", &listStepProcess)
	
	// -------------------------- UPDATE INVENTORY (PENDING) ---------------
	endpoint, err = s.getServiceEndpoint(1)
	if err != nil {
		span.RecordError(err) 
        span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	for i := range *cart.CartItem {

		cartItem := &(*cart.CartItem)[i]

		inventory := model.Inventory { 
			Pending: cartItem.Quantity,
		}

		httpClientParameter = go_core_http.HttpClientParameter {
			Url:	fmt.Sprintf("%s%s%v", endpoint.Url , "/inventory/product/", cartItem.Product.Sku),
			Method:	"PUT",
			Timeout: endpoint.HttpTimeout,
			Headers: &headers,
			Body: inventory,
		}

		_, err = s.doHttpCall(ctx, httpClientParameter)
		if err != nil {
			span.RecordError(err) 
			span.SetStatus(codes.Error, err.Error())
			s.logger.Error().
				Ctx(ctx).
				Err(err).Send()
			return nil, err
		}

		registerOrchestrationProcess(fmt.Sprintf("%s%v","INVENTORY:PENDING:SUCESSFULL:",cartItem.Product.Sku), &listStepProcess)
	}

	// -------------------------- CREATE A ORDER -----------------------------
	// prepare data
	order.CreatedAt = time.Now()
	order.Cart = *cart
	order.Transaction = "order:" + uuid.New().String()
	order.Status = "ORDER:PENDING"

	for i := range *order.Cart.CartItem { 
		cartItem := &(*cart.CartItem)[i]
		order.Amount = order.Amount + (cartItem.Price * float64(cartItem.Quantity)) - cartItem.Discount // calc order summary
	}

	// Create order
	resOrder, err := s.workerRepository.AddOrder(ctx, tx, order)
	if err != nil {
		return nil, err
	}
	order.ID = resOrder.ID

	registerOrchestrationProcess("ORDER:PENDING:SUCESSFULL", &listStepProcess)

	// -------------------------- CREATE A OUTBOX -----------------------------
	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)

	metaData := map[string]any{
		"otel": carrier,
		"request-id": fmt.Sprintf("%v",ctx.Value("request-id")),
	}
	
	orderOutbox := model.Outbox{
		ID:	uuid.New().String(),
		Type: "order_created",
		Transaction: order.Transaction,
		CreatedAt: order.CreatedAt,
		Metadata: metaData,
		Data: order,
	}

	// Create orderOutbox
	_, err = s.workerRepository.OutboxOrder(ctx, tx, &orderOutbox)
	if err != nil {
		return nil, err
	}

	registerOrchestrationProcess("ORDER:OUTBOX:SUCESSFULL", &listStepProcess)

	// --------------------
	order.StepProcess = &listStepProcess
	return order, nil
}