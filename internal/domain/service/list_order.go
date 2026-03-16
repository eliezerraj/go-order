package service

import (
	"fmt"
	"context"
	"github.com/go-order/internal/domain/model"
	
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/codes"

	go_core_http "github.com/eliezerraj/go-core/v2/http"
)

func (s * WorkerService) ListOrder(ctx context.Context, 
								  order *model.Order) (*[]model.Order, error){

	s.logger.Info().
		Ctx(ctx).
		Str("func","ListOrder").Send()

	// trace and log
	ctx, span := s.tracerProvider.SpanCtx(ctx, "service.ListOrder", trace.SpanKindServer)
	defer span.End()

	listOrder, err := s.workerRepository.ListOrder(ctx, order)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	// prepare headers http for calling services
	headers := s.buildHeaders(ctx)

	for i := range *listOrder {
		s.logger.Info().
			Ctx(ctx).
			Str("func","ListOrder").
			Interface("order", (*listOrder)[i]).
			Send()
	
		// --------------- STEP 1 (get cart and cart itens details) ------------
		endpoint, err := s.getServiceEndpoint(0)
		if err != nil {
			span.RecordError(err) 
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}

		httpClientParameter := go_core_http.HttpClientParameter {
			Url:	fmt.Sprintf("%s%s%v", endpoint.Url , "/cart/" , (*listOrder)[i].Cart.ID ),
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

		(*listOrder)[i].Cart = *cart

		// ---------------------- STEP 2 (get payment) --------------------------
		endpoint, err = s.getServiceEndpoint(2)
		if err != nil {
			span.RecordError(err) 
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
		httpClientParameter = go_core_http.HttpClientParameter {
			Url:	fmt.Sprintf("%s%s%v", endpoint.Url , "/payment/order/" , (*listOrder)[i].ID ),
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

		(*listOrder)[i].Payment = listPayment

	}

	return listOrder, nil
}