package service

import (
	"fmt"
	"context"

	"github.com/go-order/internal/domain/model"

	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/codes"

	go_core_http "github.com/eliezerraj/go-core/v2/http"
)

// About get order and complete cart via service
func (s * WorkerService) GetOrder(ctx context.Context, 
								  order *model.Order) (*model.Order, error){
	s.logger.Info().
		Ctx(ctx).
		Str("func","GetOrder").Send()

	// trace and log
	ctx, span := s.tracerProvider.SpanCtx(ctx, "service.GetOrder", trace.SpanKindServer)
	defer span.End()

	// Call a service
	resOrder, err := s.workerRepository.GetOrder(ctx, order)
	if err != nil {
		return nil, err
	}
			
	// prepare headers http for calling services
	headers := s.buildHeaders(ctx)

	// --------------- STEP 1 (get cart and cart itens details) ------------
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

	// call a service via http
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

	// ---------------------- STEP 2 (get payment) --------------------------
	endpoint, err = s.getServiceEndpoint(2)
	if err != nil {
		span.RecordError(err) 
        span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	httpClientParameter = go_core_http.HttpClientParameter {
		Url:	fmt.Sprintf("%s%s%v", endpoint.Url , "/payment/order/" , resOrder.ID ),
		Method:	"GET",
		Timeout: endpoint.HttpTimeout,
		Headers: &headers,
	}

	// call a service via http
	resPayload, err = s.doHttpCall(ctx, httpClientParameter)
	if err != nil {
		span.RecordError(err) 
        span.SetStatus(codes.Error, err.Error())
		s.logger.Error().
			Ctx(ctx).
			Err(err).Send()
		return nil, err
	}

	listPayment, err := s.parseListPaymentFromPayload(ctx, resPayload)
	if err != nil {
		span.RecordError(err) 
        span.SetStatus(codes.Error, err.Error())
		s.logger.Error().
			Ctx(ctx).
			Err(err).Send()
		return nil, err
	}

	// Clean the order data in order to avoid repetition information
	for i := range *listPayment{
		(*listPayment)[i].Order = nil
	}
	
	// ------------------------------------------------------
	// fill response
	resOrder.Cart = *cart
	resOrder.Payment = listPayment
	return resOrder, nil
}