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
			s.logger.Error().
					Ctx(ctx).
					Err(erro.ErrNotFound).Send()
			return nil, erro.ErrNotFound
		} else {
			s.logger.Error().
					Ctx(ctx).
					Err(erro.ErrBadRequest).Send()
			return nil, erro.ErrBadRequest 
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
			Str("func","Stat").Send()

	return s.workerRepository.Stat(ctx)
}

// About check health service
func (s * WorkerService) HealthCheck(ctx context.Context) error {
	s.logger.Info().
			Str("func","HealthCheck").Send()

	// Check database health
	err := s.workerRepository.DatabasePG.Ping()
	if err != nil {
		s.logger.Error().
				Err(err).Msg("*** Database HEALTH CHECK FAILED ***")
		return erro.ErrHealthCheck
	}

	s.logger.Info().
			Str("func","HealthCheck").
			Msg("*** Database HEALTH CHECK SUCCESSFULL ***")
	
	// check service/dependencies 
	headers := map[string]string{
		"Content-Type":  "application/json;charset=UTF-8",
	}

	httpClientParameter := go_core_http.HttpClientParameter {
		Url:	(*s.appServer.Endpoint)[0].Url + "/health",
		Method:	"GET",
		Timeout: (*s.appServer.Endpoint)[0].HttpTimeout,
		Headers: &headers,
	}

	// ----------------------------------------------------------
	// call a service via http
	_, err = s.doHttpCall(ctx, 
						   httpClientParameter)
	if err != nil {
		s.logger.Error().
				Ctx(ctx).
				Err(err).Msgf("*** Service %s HEALTH CHECK FAILED ***", (*s.appServer.Endpoint)[0].HostName )
		return erro.ErrHealthCheck
	}
	s.logger.Info().
			Str("func","HealthCheck").
			Msgf("*** Service %s HEALTH CHECK SUCCESSFULL ***", (*s.appServer.Endpoint)[0].HostName )

	// ----------------------------------------------------------
	// call a service via http
	httpClientParameter = go_core_http.HttpClientParameter {
		Url:	(*s.appServer.Endpoint)[1].Url + "/health",
		Method:	"GET",
		Timeout: (*s.appServer.Endpoint)[1].HttpTimeout,
		Headers: &headers,
	}

	// call a service via http
	_, err = s.doHttpCall(ctx, 
						   httpClientParameter)
	if err != nil {
		s.logger.Error().
				Ctx(ctx).
				Err(err).Msgf("*** Service %s HEALTH CHECK FAILED ***", (*s.appServer.Endpoint)[1].HostName )
		return erro.ErrHealthCheck
	}
	s.logger.Info().
			Str("func","HealthCheck").
			Msgf("*** Service %s HEALTH CHECK SUCCESSFULL ***", (*s.appServer.Endpoint)[1].HostName )

	// ----------------------------------------------------------
	// call a service via http
	httpClientParameter = go_core_http.HttpClientParameter {
		Url:	(*s.appServer.Endpoint)[3].Url + "/health",
		Method:	"GET",
		Timeout: (*s.appServer.Endpoint)[3].HttpTimeout,
		Headers: &headers,
	}

	// call a service via http
	_, err = s.doHttpCall(ctx, 
						   httpClientParameter)
	if err != nil {
		s.logger.Error().
				Ctx(ctx).
				Err(err).Msgf("*** Service %s HEALTH CHECK FAILED ***", (*s.appServer.Endpoint)[3].HostName )
		return erro.ErrHealthCheck
	}
	s.logger.Info().
			Str("func","HealthCheck").
			Msgf("*** Service %s HEALTH CHECK SUCCESSFULL ***", (*s.appServer.Endpoint)[3].HostName )

	return nil
}

// About get order
func (s * WorkerService) GetOrder(	ctx context.Context, 
									order *model.Order) (*model.Order, error){
	// trace
	ctx, span := tracerProvider.SpanCtx(ctx, "service.GetOrder")
	defer span.End()

	s.logger.Info().
			Ctx(ctx).
			Str("func","GetOrder").Send()

	// Call a service
	resOrder, err := s.workerRepository.GetOrder(ctx, order)
	if err != nil {
		s.logger.Error().
				Ctx(ctx).
				Err(err).Send()
		return nil, err
	}
								
	return resOrder, nil
}

// About get order and complete cart via service
func (s * WorkerService) GetOrderService(	ctx context.Context, 
											order *model.Order) (*model.Order, error){
	// trace and log
	ctx, span := tracerProvider.SpanCtx(ctx, "service.GetOrderService")
	defer span.End()

	s.logger.Info().
			Ctx(ctx).
			Str("func","GetOrderService").Send()

	// Call a service
	resOrder, err := s.workerRepository.GetOrderService(ctx, order)
	if err != nil {
		s.logger.Error().
				Ctx(ctx).
				Err(err).Send()
		return nil, err
	}
			
	// prepare headers http for calling services
	trace_id := fmt.Sprintf("%v",ctx.Value("trace-request-id"))

	headers := map[string]string{
		"Content-Type":  "application/json;charset=UTF-8",
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
		Url:	fmt.Sprintf("%s%s%v", (*s.appServer.Endpoint)[3].Url , "/payment/order/" , resOrder.ID ),
		Method:	"GET",
		Timeout: (*s.appServer.Endpoint)[3].HttpTimeout,
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
	// trace and log
	ctx, span := tracerProvider.SpanCtx(ctx, "service.AddOrder")

	s.logger.Info().
			Ctx(ctx).
			Str("func","AddOrder").Send()

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
	trace_id := fmt.Sprintf("%v",ctx.Value("trace-request-id"))

	headers := map[string]string{
		"Content-Type":  "application/json;charset=UTF-8",
		"X-Request-Id": trace_id,
	}

	// ---------------------- STEP 1 (post/create a cart) --------------------------

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
		s.logger.Error().
				Ctx(ctx).
				Err(err).Send()
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

	registerOrchestrationProcess("CREATE CART:OK", &listStepProcess)
	
	// -------------------------- CREATE A ORDER -----------------------------
	// prepare data
	order.CreatedAt = time.Now()
	order.Cart = cart
	order.Transaction = "order:" + uuid.New().String()
	order.Status = "`WAITING PAYMENT"

	for i := range *order.Cart.CartItem { 
		cartItem := &(*cart.CartItem)[i]
		order.Amount = order.Amount + (cartItem.Price * float64(cartItem.Quantity)) - cartItem.Discount // calc order summary
	}

	// Create order
	resOrder, err := s.workerRepository.AddOrder(ctx, 
												tx, 
												order)
	if err != nil {
		s.logger.Error().
				Ctx(ctx).
				Err(err).Send()
		return nil, err
	}
	order.ID = resOrder.ID

	// --------------------------------------------- 
	order.StepProcess = &listStepProcess
	return order, nil
}

// checkout order
func (s *WorkerService) Checkout(ctx context.Context, 
								order *model.Order) (*model.Order, error){
	// trace and log
	ctx, span := tracerProvider.SpanCtx(ctx, "service.Checkout")

	s.logger.Info().
			Ctx(ctx).
			Str("func","Checkout").Send()

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
	resOrder, err := s.workerRepository.GetOrderService(ctx, order)
	if err != nil {
		s.logger.Error().
				Ctx(ctx).
				Err(err).Send()
		return nil, err
	}

	// create saga orchestration process
	listStepProcess := []model.StepProcess{}

	// 	prepare headers http for calling services
	trace_id := fmt.Sprintf("%v",ctx.Value("trace-request-id"))

	headers := map[string]string{
		"Content-Type":  "application/json;charset=UTF-8",
		"X-Request-Id": trace_id,
	}
	

	// ---------------------- STEP 2 (get cart) --------------------------
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

	// ---------------------- STEP 3 (update inventory - reserved) --------------------------
	for i := range *cart.CartItem{
		cartItem := &(*cart.CartItem)[i]

		bodyInventory := model.Inventory { 
			Reserved: cartItem.Quantity,
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

		cartItem.Status = "RESERVED:PRODUCT"

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
	}

	registerOrchestrationProcess("INVENTORY:RESERVED", &listStepProcess)
	registerOrchestrationProcess("CART_ITEM:RESERVED:PRODUCT", &listStepProcess)
	
	// ---------------------- STEP 4 (update cart -from BASKET to RESERVED ) --------------------------
	resOrder.Cart.Status = "RESERVED"
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

	registerOrchestrationProcess("CART_ITEM:RESERVED", &listStepProcess)

	// ---------------------- STEP 1 (create a clearance) ----------------------

	listPayment := []model.Payment{}
	
	for i := range *order.Payment{
		payment := &(*order.Payment)[i]
		payment.Status = "PENDING"
		payment.Order = resOrder
		payment.Transaction = resOrder.Transaction

		httpClientParameter := go_core_http.HttpClientParameter {
			Url:	(*s.appServer.Endpoint)[3].Url + "/payment",
			Method:	"POST",
			Timeout: (*s.appServer.Endpoint)[3].HttpTimeout,
			Headers: &headers,
			Body: payment,
		}

		resPayload, err := s.doHttpCall(ctx, 
										httpClientParameter)
		if err != nil {
			s.logger.Error().
					Ctx(ctx).
					Err(err).Send()
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

	registerOrchestrationProcess("PAYMENT:SENDED", &listStepProcess)

	// --------------- update order status --------------------

	resOrder.Status = "ORDER-PREAPPROVED:WAITING CLEARANCE"
	now := time.Now()
	resOrder.UpdatedAt = &now

	row, err := s.workerRepository.UpdateOrder(ctx, tx, resOrder)
	if err != nil {
		s.logger.Error().
				Ctx(ctx).
				Err(err).Send()
		return nil, err
	}
	if row == 0 {
		s.logger.Error().
				Ctx(ctx).
				Err(erro.ErrUpdate).Send()
		return nil, erro.ErrUpdate
	}

	// fill it
	resOrder.Payment = &listPayment
	resOrder.StepProcess = &listStepProcess

	return resOrder, nil
}
