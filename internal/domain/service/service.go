package service

import (
	"fmt"
	"time"
	"errors"
	"context"
	"net/http"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/go-order/shared/erro"
	"github.com/go-order/internal/domain/model"
	database "github.com/go-order/internal/infrastructure/repo/database"

	go_core_http "github.com/eliezerraj/go-core/v2/http"
	go_core_db_pg "github.com/eliezerraj/go-core/v2/database/postgre"
	go_core_otel_trace "github.com/eliezerraj/go-core/v2/otel/trace"
)

var tracerProvider go_core_otel_trace.TracerProvider

type WorkerService struct {
	appServer			*model.AppServer
	workerRepository	*database.WorkerRepository
	logger 				*zerolog.Logger
	httpService			*go_core_http.HttpService		 	
}

// About new worker service
func NewWorkerService(appServer	*model.AppServer,
					  workerRepository *database.WorkerRepository, 
					  appLogger *zerolog.Logger) *WorkerService {
	logger := appLogger.With().
						Str("package", "domain.service").
						Logger()
	logger.Info().
			Str("func","NewWorkerService").Send()

	httpService := go_core_http.NewHttpService(&logger)					

	return &WorkerService{
		appServer: appServer,
		workerRepository: workerRepository,
		logger: &logger,
		httpService: httpService,
	}
}

// about do http call 
func (s *WorkerService) doHttpCall(ctx context.Context,
									httpClientParameter go_core_http.HttpClientParameter) (interface{},error) {
	s.logger.Info().
			 Ctx(ctx).
			 Str("func","doHttpCall").Send()

	resPayload, statusCode, err := s.httpService.DoHttp(ctx, 
														httpClientParameter)
	if err != nil {
		s.logger.Error().
				Ctx(ctx).
				Err(err).Send()
		return nil, err
	}

	if statusCode != http.StatusOK {
		if statusCode == http.StatusNotFound {
			s.logger.Warn().
					 Ctx(ctx).
					 Err(erro.ErrNotFound).Send()
			return nil, erro.ErrNotFound
		} else {		
			jsonString, err  := json.Marshal(resPayload)
			if err != nil {
				s.logger.Error().
						 Ctx(ctx).
						 Err(err).Send()
				return nil, errors.New(err.Error())
			}			
			
			message := model.APIError{}
			json.Unmarshal(jsonString, &message)

			newErr := errors.New(fmt.Sprintf("http call error: status code %d - message: %s", message.StatusCode ,message.Msg))
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

	ctx, span := tracerProvider.SpanCtx(ctx, "service.HealthCheck")
	defer span.End()

	// Check database health
	_, spanDB := tracerProvider.SpanCtx(ctx, "DatabasePG.Ping")
	err := s.workerRepository.DatabasePG.Ping()
	if err != nil {
		s.logger.Error().
				 Ctx(ctx).
				 Err(err).Msg("*** Database HEALTH CHECK FAILED ***")
		return erro.ErrHealthCheck
	}
	spanDB.End()

	s.logger.Info().
			 Ctx(ctx).
			 Str("func","HealthCheck").
			 Msg("*** Database HEALTH CHECK SUCCESSFULL ***")
	
	// ----------------------------------------------------------		
	// check service/dependencies 
	headers := map[string]string{
		"Content-Type":  "application/json;charset=UTF-8",
	}

	ctxService00, spanService00 := tracerProvider.SpanCtx(ctx, "health.service." + (*s.appServer.Endpoint)[0].HostName )

	httpClientParameter := go_core_http.HttpClientParameter {
		Url:	(*s.appServer.Endpoint)[0].Url + "/health",
		Method:	"GET",
		Timeout: (*s.appServer.Endpoint)[0].HttpTimeout,
		Headers: &headers,
	}

	// call a service via http
	_, err = s.doHttpCall(ctxService00, 
						   httpClientParameter)
	if err != nil {
		s.logger.Error().
				Ctx(ctxService00).
				Err(err).Msgf("*** Service %s HEALTH CHECK FAILED ***", (*s.appServer.Endpoint)[0].HostName )
		spanService00.End()		
		return erro.ErrHealthCheck
	}
	s.logger.Info().
			 Ctx(ctxService00).
			 Str("func","HealthCheck").
			 Msgf("*** Service %s HEALTH CHECK SUCCESSFULL ***", (*s.appServer.Endpoint)[0].HostName )

	spanService00.End()

	// ----------------------------------------------------------
	// call a service via http
	ctxService01, spanService01 := tracerProvider.SpanCtx(ctx, "health.service." + (*s.appServer.Endpoint)[1].HostName )

	httpClientParameter = go_core_http.HttpClientParameter {
		Url:	(*s.appServer.Endpoint)[1].Url + "/health",
		Method:	"GET",
		Timeout: (*s.appServer.Endpoint)[1].HttpTimeout,
		Headers: &headers,
	}

	// call a service via http
	_, err = s.doHttpCall(ctxService01, 
						   httpClientParameter)
	if err != nil {
		s.logger.Error().
				 Ctx(ctxService01).
				 Err(err).Msgf("*** Service %s HEALTH CHECK FAILED ***", (*s.appServer.Endpoint)[1].HostName )
		return erro.ErrHealthCheck
	}
	s.logger.Info().
			 Ctx(ctxService01).
			 Str("func","HealthCheck").
			 Msgf("*** Service %s HEALTH CHECK SUCCESSFULL ***", (*s.appServer.Endpoint)[1].HostName )

	spanService01.End()
	
	// ----------------------------------------------------------
	// call a service via http
	ctxService02, spanService02 := tracerProvider.SpanCtx(ctx, "health.service." + (*s.appServer.Endpoint)[2].HostName )

	httpClientParameter = go_core_http.HttpClientParameter {
		Url:	(*s.appServer.Endpoint)[2].Url + "/health",
		Method:	"GET",
		Timeout: (*s.appServer.Endpoint)[2].HttpTimeout,
		Headers: &headers,
	}

	// call a service via http
	_, err = s.doHttpCall(ctxService02, 
						   httpClientParameter)
	if err != nil {
		s.logger.Error().
				 Ctx(ctxService02).
				 Err(err).Msgf("*** Service %s HEALTH CHECK FAILED ***", (*s.appServer.Endpoint)[2].HostName )
		return erro.ErrHealthCheck
	}
	s.logger.Info().
			 Ctx(ctxService02).
			 Str("func","HealthCheck").
			 Msgf("*** Service %s HEALTH CHECK SUCCESSFULL ***", (*s.appServer.Endpoint)[2].HostName )

	spanService02.End()

	// final
	return nil
}

// About get order (usind join, all tables in the same database)
func (s * WorkerService) GetOrderV1(ctx context.Context, 
								    order *model.Order) (*model.Order, error){
	s.logger.Info().
			Ctx(ctx).
			Str("func","GetOrderV1").Send()

	// trace
	ctx, span := tracerProvider.SpanCtx(ctx, "service.GetOrderV1")
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
}

// About get order and complete cart via service
func (s * WorkerService) GetOrder(ctx context.Context, 
								  order *model.Order) (*model.Order, error){
	s.logger.Info().
			 Ctx(ctx).
			 Str("func","GetOrder").Send()

	// trace and log
	ctx, span := tracerProvider.SpanCtx(ctx, "service.GetOrder")
	defer span.End()

	// Call a service
	resOrder, err := s.workerRepository.GetOrder(ctx, order)
	if err != nil {
		return nil, err
	}
			
	// prepare headers http for calling services
	trace_id := fmt.Sprintf("%v",ctx.Value("request-id"))

	headers := map[string]string{
		"Content-Type": "application/json;charset=UTF-8",
		"X-Request-Id": trace_id,
	}

	// --------------- STEP 1 (get cart and cart itens details) ------------
	httpClientParameter := go_core_http.HttpClientParameter {
		Url:	fmt.Sprintf("%s%s%v", (*s.appServer.Endpoint)[0].Url , "/cart/" , resOrder.Cart.ID ),
		Method:	"GET",
		Timeout: (*s.appServer.Endpoint)[0].HttpTimeout,
		Headers: &headers,
	}

	// call a service via http
	resPayload, err := s.doHttpCall(ctx, 
									httpClientParameter)
	if err != nil {
		s.logger.Error().
				 Ctx(ctx).
				 Err(err).Send()
		return nil, err
	}

	// convert json to struct
	jsonString, err  := json.Marshal(resPayload)
	if err != nil {
		s.logger.Error().
				 Ctx(ctx).
				 Err(err).Send()
		return nil, errors.New(err.Error())
	}
	cart := model.Cart{}
	json.Unmarshal(jsonString, &cart)

	// ---------------------- STEP 2 (get payment) --------------------------
	httpClientParameter = go_core_http.HttpClientParameter {
		Url:	fmt.Sprintf("%s%s%v", (*s.appServer.Endpoint)[2].Url , "/payment/order/" , resOrder.ID ),
		Method:	"GET",
		Timeout: (*s.appServer.Endpoint)[2].HttpTimeout,
		Headers: &headers,
	}

	// call a service via http
	resPayload, err = s.doHttpCall(ctx, 
									httpClientParameter)
	if err != nil {
		s.logger.Error().
				 Ctx(ctx).
				 Err(err).Send()
		return nil, err
	}

	// convert json to struct
	jsonString, err = json.Marshal(resPayload)
	if err != nil {
		s.logger.Error().
				 Ctx(ctx).
				 Err(err).Send()
		return nil, errors.New(err.Error())
	}
	listPayment := []model.Payment{}
	json.Unmarshal(jsonString, &listPayment)

	// Clean the order data in order to avoid cycle information
	for i := range listPayment{
		listPayment[i].Order = nil
	}
	
	// ------------------------------------------------------
	// fill response
	resOrder.Cart = cart
	resOrder.Payment = &listPayment
	return resOrder, nil
}

// About create a payment
func (s *WorkerService) AddOrder(ctx context.Context, 
								order *model.Order) (*model.Order, error){
	s.logger.Info().
			 Ctx(ctx).
			 Str("func","AddOrder").Send()

	// trace and log
	ctx, span := tracerProvider.SpanCtx(ctx, "service.AddOrder")

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
	trace_id := fmt.Sprintf("%v",ctx.Value("request-id"))

	headers := map[string]string{
		"Content-Type":  "application/json;charset=UTF-8",
		"X-Request-Id": trace_id,
	}

	// ---------------------- STEP 1 (create a cart) --------------------------
	// prepare data
	order.Cart.Status = "CART:POSTED"

	httpClientParameter := go_core_http.HttpClientParameter {
		Url:	(*s.appServer.Endpoint)[0].Url + "/cart",
		Method:	"POST",
		Timeout: (*s.appServer.Endpoint)[0].HttpTimeout,
		Headers: &headers,
		Body: order.Cart,
	}

	// call a service via http
	resPayload, err := s.doHttpCall(ctx, 
									httpClientParameter)

	if err != nil {		
		return nil, err
	}

	jsonString, err  := json.Marshal(resPayload)
	if err != nil {
		s.logger.Error().
				 Ctx(ctx).
				 Err(err).Send()
		return nil, errors.New(err.Error())
	}
	cart := model.Cart{}
	json.Unmarshal(jsonString, &cart)

	registerOrchestrationProcess("CART:POSTED:SUCESSFULL", &listStepProcess)

	// -------------------------- UPADATE INVENTORY (PENDING) ---------------
	for i := range *cart.CartItem {
		cartItem := &(*cart.CartItem)[i]

		bodyInventory := model.Inventory { 
			Pending: cartItem.Quantity,
			Available: cartItem.Quantity * -1,
		}

		httpClientParameter = go_core_http.HttpClientParameter {
			Url:	fmt.Sprintf("%s%s%v", (*s.appServer.Endpoint)[1].Url , "/inventory/product/", cartItem.Product.Sku),
			Method:	"PUT",
			Timeout: (*s.appServer.Endpoint)[1].HttpTimeout,
			Headers: &headers,
			Body: bodyInventory,
		}

		_, err = s.doHttpCall(ctx, 
							  httpClientParameter)
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
	order.Cart = cart
	order.Transaction = "order:" + uuid.New().String()
	order.Status = "ORDER:POSTED"

	for i := range *order.Cart.CartItem { 
		cartItem := &(*cart.CartItem)[i]
		order.Amount = order.Amount + (cartItem.Price * float64(cartItem.Quantity)) - cartItem.Discount // calc order summary
	}

	// Create order
	resOrder, err := s.workerRepository.AddOrder(ctx, 
												tx, 
												order)
	if err != nil {
		return nil, err
	}
	order.ID = resOrder.ID

	registerOrchestrationProcess("ORDER:POSTED:SUCESSFULL", &listStepProcess)

	// --------------------------------------------- 
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
	ctx, span := tracerProvider.SpanCtx(ctx, "service.Checkout")

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

	// 	prepare headers http for calling services
	trace_id := fmt.Sprintf("%v",ctx.Value("request-id"))

	headers := map[string]string{
		"Content-Type":  "application/json;charset=UTF-8",
		"X-Request-Id": trace_id,
	}
	
	// ---------------------- STEP 1 (get cart) --------------------------
	httpClientParameter := go_core_http.HttpClientParameter {
		Url:	fmt.Sprintf("%s%s%v", (*s.appServer.Endpoint)[0].Url , "/cart/" , resOrder.Cart.ID ),
		Method:	"GET",
		Timeout: (*s.appServer.Endpoint)[0].HttpTimeout,
		Headers: &headers,
	}

	resPayload, err := s.doHttpCall(ctx, 
									httpClientParameter)
	if err != nil {
		s.logger.Error().
				 Ctx(ctx).
				 Err(err).Send()
		return nil, err
	}

	jsonString, err := json.Marshal(resPayload)
	if err != nil {
		s.logger.Error().
				Ctx(ctx).
				Err(err).Send()
		return nil, errors.New(err.Error())
	}
	var cart model.Cart
	json.Unmarshal(jsonString, &cart)

	resOrder.Cart = cart

	// ---------------------- STEP 2 (update inventory - PENDING TO SOLD and CartItem) --------------------------
	for i := range *cart.CartItem {
		cartItem := &(*cart.CartItem)[i]

		// If cartItem.Status is RESERVED, skip it (means already processed)
		if cartItem.Status == "CART_ITEM:RESERVED" {
			continue // skip already processed items
		}

		bodyInventory := model.Inventory { 
			Pending: cartItem.Quantity * -1,
			Sold: cartItem.Quantity,
		}

		httpClientParameter = go_core_http.HttpClientParameter {
			Url:	fmt.Sprintf("%s%s%v", (*s.appServer.Endpoint)[1].Url , "/inventory/product/", cartItem.Product.Sku),
			Method:	"PUT",
			Timeout: (*s.appServer.Endpoint)[1].HttpTimeout,
			Headers: &headers,
			Body: bodyInventory,
		}

		_, err = s.doHttpCall(ctx, 
							  httpClientParameter)
		if err != nil {
			s.logger.Error().
					Ctx(ctx).
					Err(err).Send()
			return nil, err
		}

		registerOrchestrationProcess(fmt.Sprintf("%s%v","INVENTORY:RESERVED:SUCESSFULL:",cartItem.Product.Sku), &listStepProcess)

		cartItem.Status = "CART_ITEM:RESERVED"

		httpClientParameter = go_core_http.HttpClientParameter {
			Url:	fmt.Sprintf("%s%s%v", (*s.appServer.Endpoint)[0].Url , "/cartItem/", cartItem.ID),
			Method:	"PUT",
			Timeout: (*s.appServer.Endpoint)[0].HttpTimeout,
			Headers: &headers,
			Body: cartItem,
		}

		_, err = s.doHttpCall(ctx, 
							  httpClientParameter)

		if err != nil {
			s.logger.Error().
					Ctx(ctx).
					Err(err).Send()
			return nil, err
		}

		registerOrchestrationProcess("CART_ITEM:RESERVED:SUCESSFULL", &listStepProcess)
	}

	// ---------------------- STEP 4 (update cart -from BASKET to RESERVED ) --------------------------
	resOrder.Cart.Status = "CART:RESERVED"
	
	httpClientParameter = go_core_http.HttpClientParameter {
			Url:	fmt.Sprintf("%s%s%v", (*s.appServer.Endpoint)[0].Url , "/cart/", resOrder.Cart.ID),
			Method:	"PUT",
			Timeout: (*s.appServer.Endpoint)[0].HttpTimeout,
			Headers: &headers,
			Body: resOrder.Cart,
	}

	_, err = s.doHttpCall(ctx, 
						  httpClientParameter)
	if err != nil {
		s.logger.Error().
				 Ctx(ctx).
				 Err(err).Send()
		return nil, err
	}

	registerOrchestrationProcess("CART:RESERVED:SUCESSFULL", &listStepProcess)

	// ---------------  STEP 5 update order status --------------------
	resOrder.Status = "ORDER:RESERVED"
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

	registerOrchestrationProcess("ORDER:RESERVED:SUCESSFULL", &listStepProcess)

	// ---------------------- STEP 6 (create a clearance) ----------------------
	listPayment := []model.Payment{}
	
	for i := range *order.Payment{
		payment := &(*order.Payment)[i]
		payment.Status = "CLEARANCE:RESERVED"
		payment.Order = resOrder
		payment.Transaction = resOrder.Transaction

		httpClientParameter := go_core_http.HttpClientParameter {
			Url:	(*s.appServer.Endpoint)[2].Url + "/payment",
			Method:	"POST",
			Timeout: (*s.appServer.Endpoint)[2].HttpTimeout,
			Headers: &headers,
			Body: payment,
		}

		resPayload, err := s.doHttpCall(ctx, 
										httpClientParameter)
		if err != nil {
			return nil, err
		}

		jsonString, err  := json.Marshal(resPayload)
		if err != nil {
			s.logger.Error().
					Ctx(ctx).
					Err(err).Send()
			return nil, errors.New(err.Error())
		}
		var resPayment model.Payment
		json.Unmarshal(jsonString, &resPayment)

		resPayment.Order = nil // clean the order indo in order to avoid (cycle info)
		listPayment = append(listPayment, resPayment)
	}

	registerOrchestrationProcess("CLEARANCE:RESERVED:SUCCESSFULL", &listStepProcess)

	// fill it
	resOrder.Payment = &listPayment
	resOrder.StepProcess = &listStepProcess

	return resOrder, nil
}
