package service

import (
	"fmt"
	"time"
	"context"

	"github.com/go-order/shared/erro"
	"github.com/go-order/internal/domain/model"

	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/codes"

	go_core_http "github.com/eliezerraj/go-core/v2/http"
)

// checkout order
func (s *WorkerService) Checkout(ctx context.Context, 
								 order *model.Order) (*model.Order, error){
	s.logger.Info().
		Ctx(ctx).
		Str("func","Checkout").Send()

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

	// get the order data	
	resOrder, err := s.workerRepository.GetOrder(ctx, order)
	if err != nil {
		return nil, err
	}

	// create saga orchestration process
	listStepProcess := []model.StepProcess{}

	// prepare headers http for calling services
	headers := s.buildHeaders(ctx)

	// ---------------------- STEP 1 (get cart) --------------------------
	endpoint, err := s.getServiceEndpoint(0)
	if err != nil {
		span.RecordError(err) 
        span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	httpClientParameter := go_core_http.HttpClientParameter {
		Url:	fmt.Sprintf("%s%s%v", endpoint.Url , "/cart/" , resOrder.Cart.ID ),
		Method:	"GET",
		Timeout: endpoint.HttpTimeout,
		Headers: &headers,
	}

	resPayload, err := s.doHttpCall(ctx, httpClientParameter)
	if err != nil {
		span.RecordError(err) 
        span.SetStatus(codes.Error, err.Error())
		s.logger.Error().
			Ctx(ctx).
			Err(err).Send()
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

	resOrder.Cart = *cart

	// ---------------------- STEP 2 (update inventory - FROM PENDING TO SOLD and CartItem) --------------------------
	endpoint, err = s.getServiceEndpoint(1)
	if err != nil {
		span.RecordError(err) 
        span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	endpoint2, err := s.getServiceEndpoint(0)
	if err != nil {
		span.RecordError(err) 
        span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	for i := range *cart.CartItem {
		cartItem := &(*cart.CartItem)[i]

		// If cartItem.Status is RESERVED, skip it (means already processed)
		if cartItem.Status == "CART_ITEM:SOLD" {
			continue // skip already processed items
		}

		inventory := model.Inventory { 
			Pending: cartItem.Quantity * -1,
			Sold: cartItem.Quantity,
			//Available: cartItem.Quantity * 1,
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

		registerOrchestrationProcess(fmt.Sprintf("%s%v","INVENTORY:SOLD:SUCESSFULL:",cartItem.Product.Sku), &listStepProcess)

		cartItem.Status = "CART_ITEM:SOLD"

		httpClientParameter = go_core_http.HttpClientParameter {
			Url:	fmt.Sprintf("%s%s%v", endpoint2.Url , "/cartItem/", cartItem.ID),
			Method:	"PUT",
			Timeout: endpoint2.HttpTimeout,
			Headers: &headers,
			Body: cartItem,
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

		registerOrchestrationProcess("CART_ITEM:SOLD:SUCESSFULL", &listStepProcess)
	}

	// ---------------------- STEP 4 (update cart -from BASKET to SOLD ) --------------------------
	resOrder.Cart.Status = "CART:SOLD"
	endpoint, err = s.getServiceEndpoint(0)

	if err != nil {
		return nil, err
	}

	httpClientParameter = go_core_http.HttpClientParameter {
			Url:	fmt.Sprintf("%s%s%v", endpoint.Url , "/cart/", resOrder.Cart.ID),
			Method:	"PUT",
			Timeout: endpoint.HttpTimeout,
			Headers: &headers,
			Body: resOrder.Cart,
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

	registerOrchestrationProcess("CART:SOLD:SUCESSFULL", &listStepProcess)

	// ---------------  STEP 5 update order status --------------------
	resOrder.Status = "ORDER:SOLD"
	now := time.Now()
	resOrder.UpdatedAt = &now

	row, err := s.workerRepository.UpdateOrder(ctx, tx, resOrder)
	if err != nil {
		return nil, err
	}
	if row == 0 {
		span.RecordError(erro.ErrUpdate) 
        span.SetStatus(codes.Error, erro.ErrUpdate.Error())
		s.logger.Error().
				Ctx(ctx).
				Err(erro.ErrUpdate).Send()
		return nil, erro.ErrUpdate
	}

	registerOrchestrationProcess("ORDER:SOLD:SUCESSFULL", &listStepProcess)

	// ---------------------- STEP 6 (create a clearance) ----------------------
	listPayment := []model.Payment{}

	endpoint, err = s.getServiceEndpoint(2)
	if err != nil {
		span.RecordError(err) 
        span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	for i := range *order.Payment{
		payment := &(*order.Payment)[i]
		payment.Status = "CLEARANCE:SOLD"
		payment.Order = resOrder
		payment.Transaction = resOrder.Transaction

		httpClientParameter := go_core_http.HttpClientParameter {
			Url:	endpoint.Url + "/payment",
			Method:	"POST",
			Timeout: endpoint.HttpTimeout,
			Headers: &headers,
			Body: payment,
		}

		resPayload, err := s.doHttpCall(ctx,httpClientParameter)
		if err != nil {
			span.RecordError(err) 
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}

		respayment, err := s.parsePaymentFromPayload(ctx, resPayload)
		if err != nil {
			span.RecordError(err) 
			span.SetStatus(codes.Error, err.Error())
			s.logger.Error().
				Ctx(ctx).
				Err(err).Send()
			return nil, err
		}
		
		respayment.Order = nil // clean the order indo in order to avoid (cycle info)
		listPayment = append(listPayment, *respayment)
	}

	registerOrchestrationProcess("CLEARANCE:SOLD:SUCCESSFULL", &listStepProcess)

	// fill it
	resOrder.Payment = &listPayment
	resOrder.StepProcess = &listStepProcess

	return resOrder, nil
}
