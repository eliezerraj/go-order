package service

import (
	"fmt"
	"time"
	"context"
	"net/http"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/go-order/shared/erro"
	"github.com/go-order/internal/domain/model"
	database "github.com/go-order/internal/infrastructure/repo/database"
	
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	go_core_http "github.com/eliezerraj/go-core/v2/http"
	go_core_db_pg "github.com/eliezerraj/go-core/v2/database/postgre"
	go_core_otel_trace "github.com/eliezerraj/go-core/v2/otel/trace"
	go_core_midleware "github.com/eliezerraj/go-core/v2/middleware"
)

type WorkerService struct {
	workerRepository 	*database.WorkerRepository
	logger 				*zerolog.Logger
	tracerProvider 		*go_core_otel_trace.TracerProvider
	httpService			*go_core_http.HttpService
	endpoint			*[]model.Endpoint	
}

// About new worker service
func NewWorkerService(workerRepository *database.WorkerRepository, 
						appLogger 		*zerolog.Logger,
						tracerProvider 	*go_core_otel_trace.TracerProvider,
						endpoint		*[]model.Endpoint) *WorkerService {

	logger := appLogger.With().
						Str("package", "domain.service").
						Logger()
	logger.Info().
			Str("func","NewWorkerService").Send()

	httpService := go_core_http.NewHttpService(&logger)					

	return &WorkerService{
		workerRepository: workerRepository,
		logger: &logger,
		tracerProvider: tracerProvider,
		httpService: httpService,
		endpoint: endpoint,
	}
}

// Helper: Get service endpoint by index with error handling
func (s *WorkerService) getServiceEndpoint(index int) (*model.Endpoint, error) {
	if s.endpoint == nil || len(*s.endpoint) <= index {
		return nil, fmt.Errorf("service endpoint at index %d not found", index)
	}
	return &(*s.endpoint)[index], nil
}

// Helper: Build HTTP headers with request ID
func (s *WorkerService) buildHeaders(ctx context.Context) map[string]string {
	requestID := go_core_midleware.GetRequestID(ctx)
	return map[string]string{
		"Content-Type":  "application/json;charset=UTF-8",
		"X-Request-Id":  requestID,
	}
}

// Helper: Parse cart from HTTP response payload
func (s *WorkerService) parseCartFromPayload(ctx context.Context, payload interface{}) (*model.Cart, error) {
	
	jsonString, err := json.Marshal(payload)
	if err != nil {
		s.logger.Error().Ctx(ctx).Err(err).Send()
		return nil, fmt.Errorf("FAILED to marshal response payload: %w", err)
	}

	cart := &model.Cart{}
	if err := json.Unmarshal(jsonString, cart); err != nil {
		s.logger.Error().Ctx(ctx).Err(err).Send()
		return nil, fmt.Errorf("FAILED to unmarshal cart: %w", err)
	}
	return cart, nil
}

// Helper: Parse cart from HTTP response payload
func (s *WorkerService) parseListPaymentFromPayload(ctx context.Context, payload interface{}) (*[]model.Payment, error) {

	jsonString, err := json.Marshal(payload)
	if err != nil {
		s.logger.Error().Ctx(ctx).Err(err).Send()
		return nil, fmt.Errorf("FAILED to marshal response payload: %w", err)
	}

	payment := &[]model.Payment{}
	if err := json.Unmarshal(jsonString, payment); err != nil {
		s.logger.Error().Ctx(ctx).Err(err).Send()
		return nil, fmt.Errorf("FAILED to unmarshal payment: %w", err)
	}
	return payment, nil
}

// Helper: Parse payment from HTTP response payload
func (s *WorkerService) parsePaymentFromPayload(ctx context.Context, payload interface{}) (*model.Payment, error) {
	
	jsonString, err := json.Marshal(payload)
	if err != nil {
		s.logger.Error().Ctx(ctx).Err(err).Send()
		return nil, fmt.Errorf("FAILED to marshal response payload: %w", err)
	}

	payment := &model.Payment{}
	if err := json.Unmarshal(jsonString, payment); err != nil {
		s.logger.Error().Ctx(ctx).Err(err).Send()
		return nil, fmt.Errorf("FAILED to unmarshal payment: %w", err)
	}
	return payment, nil
}

// about do http call 
func (s *WorkerService) doHttpCall(ctx context.Context,	httpClientParameter go_core_http.HttpClientParameter) (interface{},error) {

	s.logger.Info().
			 Ctx(ctx).
			 Str("func","doHttpCall").Send()

	resPayload, statusCode, err := s.httpService.DoHttp(ctx, httpClientParameter)

	s.logger.Info().
			 Interface("======> httpClientParameter",httpClientParameter).
			 Interface("======> statusCode", statusCode).Send()

	if err != nil {
		s.logger.Error().
				Ctx(ctx).
				Err(err).Send()
		return nil, err
	}
	if statusCode != http.StatusOK && statusCode != http.StatusCreated {
		if statusCode == http.StatusNotFound {
			s.logger.Warn().
					 Ctx(ctx).
					 Err(erro.ErrNotFound).Send()
			return nil, erro.ErrNotFound
		} else {		
			jsonString, err := json.Marshal(resPayload)
			if err != nil {
				s.logger.Error().
						Ctx(ctx).
						Err(err).Send()
				return nil, fmt.Errorf("FAILED to marshal http response: %w", err)
			}			
			
			message := model.APIError{}
			if err := json.Unmarshal(jsonString, &message); err != nil {
				s.logger.Error().
						Ctx(ctx).
						Err(err).Send()
				return nil, fmt.Errorf("FAILED to unmarshal error response: %w", err)
			}

			newErr := fmt.Errorf("http call error: status code %d - message: %s", statusCode, message.Msg)
			s.logger.Error().
					Ctx(ctx).
					Err(newErr).Send()
			return nil, newErr
		}
	}

	return resPayload, nil
}

// register a new step proccess
func registerOrchestrationProcess(nameStepProcess string,
								 listStepProcess *[]model.StepProcess) {

	stepProcess := model.StepProcess{Name: nameStepProcess,
									ProcessedAt: time.Now(),}

	*listStepProcess = append(*listStepProcess, stepProcess)								
}

// About database stats
func (s *WorkerService) Stat(ctx context.Context) (go_core_db_pg.PoolStats){
	s.logger.Info().
			Ctx(ctx).
			Str("func","Stat").Send()

	return s.workerRepository.Stat(ctx)
}

// About check health service
func (s * WorkerService) HealthCheck(ctx context.Context) error {
	s.logger.Info().
			Ctx(ctx).
			Str("func","HealthCheck").Send()

	ctx, span := s.tracerProvider.SpanCtx(ctx, "service.HealthCheck", trace.SpanKindServer)
	defer span.End()

	// Check database health
	ctx, spanDB := s.tracerProvider.SpanCtx(ctx, "DatabasePG.Ping", trace.SpanKindInternal)
	err := s.workerRepository.DatabasePG.Ping()
	spanDB.End()

	if err != nil {
		s.logger.Error().
				 Ctx(ctx).
				 Err(err).Msg("*** Database HEALTH CHECK FAILED ***")
		return erro.ErrHealthCheck
	}

	s.logger.Info().
			 Ctx(ctx).
			 Msg("*** Database HEALTH CHECK SUCCESSFULL ***")
	
	// ----------------------------------------------------------		
	// check service/dependencies 
	endpoint, err := s.getServiceEndpoint(0)
	if err != nil {
		return err
	}
	headers := s.buildHeaders(ctx)

	httpClientParameter := go_core_http.HttpClientParameter {
		Url:	endpoint.Url + "/health",
		Method:	"GET",
		Timeout: endpoint.HttpTimeout,
		Headers: &headers,
	}

	// call a service via http
	_, err = s.doHttpCall(ctx, httpClientParameter)
	if err != nil {
		s.logger.Error().
				Ctx(ctx).
				Err(err).Msgf("*** Service %s HEALTH CHECK FAILED ***", endpoint.HostName)
		return erro.ErrHealthCheck
	}
	s.logger.Info().
			 Ctx(ctx).
			 Str("func","HealthCheck").
			 Msgf("*** Service %s HEALTH CHECK SUCCESSFULL ***", endpoint.HostName)

	// ----------------------------------------------------------
	// call a service via http
	endpoint, err = s.getServiceEndpoint(1)
	if err != nil {
		return err
	}

	httpClientParameter = go_core_http.HttpClientParameter {
		Url:	endpoint.Url + "/health",
		Method:	"GET",
		Timeout: endpoint.HttpTimeout,
		Headers: &headers,
	}

	// call a service via http
	_, err = s.doHttpCall(ctx, httpClientParameter)
	if err != nil {
		s.logger.Error().
				 Ctx(ctx).
				 Err(err).Msgf("*** Service %s HEALTH CHECK FAILED ***", endpoint.HostName)
		return erro.ErrHealthCheck
	}
	s.logger.Info().
			 Ctx(ctx).
			 Msgf("*** Service %s HEALTH CHECK SUCCESSFULL ***", endpoint.HostName)

	// ----------------------------------------------------------
	// call a service via http
	endpoint, err = s.getServiceEndpoint(2)
	if err != nil {
		return err
	}

	httpClientParameter = go_core_http.HttpClientParameter {
		Url:	endpoint.Url + "/health",
		Method:	"GET",
		Timeout: endpoint.HttpTimeout,
		Headers: &headers,
	}

	// call a service via http
	_, err = s.doHttpCall(ctx, httpClientParameter)
	if err != nil {
		s.logger.Error().
				 Ctx(ctx).
				 Err(err).Msgf("*** Service %s HEALTH CHECK FAILED ***", endpoint.HostName)
		return erro.ErrHealthCheck
	}
	s.logger.Info().
			 Ctx(ctx).
			 Msgf("*** Service %s HEALTH CHECK SUCCESSFULL ***", endpoint.HostName)

	// final
	return nil
}

// About get order (usind join, all tables in the same database)
/*func (s * WorkerService) GetOrderV1(ctx context.Context, 
								    order *model.Order) (*model.Order, error){
	s.logger.Info().
			Ctx(ctx).
			Str("func","GetOrderV1").Send()

	// trace
	ctx, span := tracerProvider.SpanCtx(ctx, "service.GetOrderV1", trace.SpanKindServer)
	defer span.End()

	// Call a service
	resOrder, err := s.workerRepository.GetOrderV1(ctx, order)
	if err != nil {
		s.logger.Error().
				 Ctx(ctx).
				 Err(err).Send()
		return nil, err
	}
								
	return resOrder, nil
}*/

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
		s.logger.Error().
			Ctx(ctx).
			Err(err).Send()
		return nil, err
	}

	cart, err := s.parseCartFromPayload(ctx, resPayload)
	if err != nil {
		s.logger.Error().
				 Ctx(ctx).
				 Err(err).Send()
		return nil, err
	}

	// ---------------------- STEP 2 (get payment) --------------------------
	endpoint, err = s.getServiceEndpoint(2)
	if err != nil {
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
		s.logger.Error().
				 Ctx(ctx).
				 Err(err).Send()
		return nil, err
	}

	listPayment, err := s.parseListPaymentFromPayload(ctx, resPayload)
	if err != nil {
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
		return nil, err
	}

	cart, err := s.parseCartFromPayload(ctx, resPayload)
	if err != nil {
		s.logger.Error().
				 Ctx(ctx).
				 Err(err).Send()
		return nil, err
	}

	registerOrchestrationProcess("CART:PENDING:SUCESSFULL", &listStepProcess)
	
	// -------------------------- UPDATE INVENTORY (PENDING) ---------------
	endpoint, err = s.getServiceEndpoint(1)
	if err != nil {
		return nil, err
	}

	for i := range *cart.CartItem {

		cartItem := &(*cart.CartItem)[i]

		inventory := model.Inventory { 
			Pending: cartItem.Quantity,
			Available: cartItem.Quantity * -1,
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
		s.logger.Error().
			Ctx(ctx).
			Err(err).Send()
		return nil, err
	}

	cart, err := s.parseCartFromPayload(ctx, resPayload)
	if err != nil {
		s.logger.Error().
				 Ctx(ctx).
				 Err(err).Send()
		return nil, err
	}

	resOrder.Cart = *cart

	// ---------------------- STEP 2 (update inventory - PENDING TO SOLD and CartItem) --------------------------
	endpoint, err = s.getServiceEndpoint(0)
	if err != nil {
		return nil, err
	}

	endpoint2, err := s.getServiceEndpoint(1)
	if err != nil {
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
			return nil, err
		}

		respayment, err := s.parsePaymentFromPayload(ctx, resPayload)
		if err != nil {
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
