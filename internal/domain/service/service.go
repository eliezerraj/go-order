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

	go_core_http "github.com/eliezerraj/go-core/http"
	go_core_db_pg "github.com/eliezerraj/go-core/database/postgre"
	go_core_otel_trace "github.com/eliezerraj/go-core/otel/trace"
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
				Err(err).Msg("*** Database HEALTH FAILED ***")
		return erro.ErrHealthCheck
	}

	s.logger.Info().
			Str("func","HealthCheck").
			Msg("*** Database HEALTH SUCCESSFULL ***")

	return nil
}

// About create a payment
func (s *WorkerService) AddOrder(ctx context.Context, 
								order *model.Order) (*model.Order, error){
	// trace and log
	ctx, span := tracerProvider.SpanCtx(ctx, "service.AddOrder")
	defer span.End()

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
	defer s.workerRepository.DatabasePG.ReleaseTx(conn)

	// handle connection
	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		} else {
			tx.Commit(ctx)
		}
		span.End()
	}()

	// create saga orchestration process
	listStepProcess := []model.StepProcess{}

	// 	prepare headers http for calling services
	trace_id := fmt.Sprintf("%v",ctx.Value("trace-request-id"))

	headers := map[string]string{
		"Content-Type":  "application/json;charset=UTF-8",
		"X-Request-Id": trace_id,
	}

	var httpClientParameter go_core_http.HttpClientParameter

	httpClientParameter = go_core_http.HttpClientParameter {
		Url:	(*s.appServer.Endpoint)[0].Url + "/cart",
		Method:	(*s.appServer.Endpoint)[0].Method,
		Timeout: (*s.appServer.Endpoint)[0].HttpTimeout,
		Headers: &headers,
	}

	httpClientParameter.Body = order.Cart

	// ---------------------- STEP 1 ---------------------------------
	res_payload, statusCode, err := s.httpService.DoHttp(ctx, 
														httpClientParameter)
	if err != nil {
		s.logger.Error().
				Ctx(ctx).
				Err(err).Send()
		return nil, err
	}

	if statusCode != http.StatusOK {
		if statusCode == http.StatusNotFound {
			return nil, erro.ErrNotFound
		} else {
			return nil, erro.ErrBadRequest 
		}
	}

	jsonString, err  := json.Marshal(res_payload)
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
		order.Amount = order.Amount + (cartItem.Price * float64(cartItem.Quantity)) - cartItem.Discount
	}

	// Create order
	resPayment, err := s.workerRepository.AddOrder(ctx, 
													tx, 
													order)
	if err != nil {
		s.logger.Error().
				Ctx(ctx).
				Err(err).Send()
		return nil, err
	}
	order.ID = resPayment.ID

	// --------------------------------------------- 
	order.StepProcess = &listStepProcess
	return order, nil
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

	var httpClientParameter go_core_http.HttpClientParameter

	httpClientParameter = go_core_http.HttpClientParameter {
		Url:	fmt.Sprintf("%s%s%v", (*s.appServer.Endpoint)[2].Url , "/cart/" , resOrder.Cart.ID ),
		Method:	(*s.appServer.Endpoint)[2].Method,
		Timeout: (*s.appServer.Endpoint)[2].HttpTimeout,
		Headers: &headers,
	}

	// call a service via http
	res_payload, statusCode, err := s.httpService.DoHttp(ctx, 
														httpClientParameter)
	if err != nil {
		s.logger.Error().
				Ctx(ctx).
				Err(err).Send()
		return nil, err
	}

	if statusCode != http.StatusOK {
		if statusCode == http.StatusNotFound {
			return nil, erro.ErrNotFound
		} else {
			return nil, erro.ErrBadRequest 
		}
	}

	// convert json to struct
	jsonString, err  := json.Marshal(res_payload)
	if err != nil {
		s.logger.Error().
				Ctx(ctx).
				Err(err).Send()
		return nil, errors.New(err.Error())
	}
	cart := model.Cart{}
	json.Unmarshal(jsonString, &cart)

	// fill response
	resOrder.Cart = cart
	return resOrder, nil
}

// checkout order
func (s *WorkerService) Checkout(ctx context.Context, 
								order *model.Order) (*model.Order, error){
	// trace and log
	ctx, span := tracerProvider.SpanCtx(ctx, "service.Checkout")
	defer span.End()

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
	defer s.workerRepository.DatabasePG.ReleaseTx(conn)

	// handle connection
	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		} else {
			tx.Commit(ctx)
		}
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

	// -----------------------------------
	// call clearance service (request payment)
	headers := map[string]string{
		"Content-Type":  "application/json;charset=UTF-8",
		"X-Request-Id": trace_id,
	}

	payment := model.Payment{
		Transaction: resOrder.Transaction,
		Order: 		resOrder,
		Type: 		order.Payment.Type,
		Status: 	"PENDING",
		Currency: 	resOrder.Currency,
		Amount: 	resOrder.Amount,
	}

	httpClientParameter := go_core_http.HttpClientParameter {
		Url:	(*s.appServer.Endpoint)[3].Url + "/payment",
		Method:	(*s.appServer.Endpoint)[3].Method,
		Timeout: (*s.appServer.Endpoint)[3].HttpTimeout,
		Headers: &headers,
		Body: payment,
	}

	res_payload, statusCode, err := s.httpService.DoHttp(ctx, 
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

	jsonString, err  := json.Marshal(res_payload)
	if err != nil {
		s.logger.Error().
				Ctx(ctx).
				Err(err).Send()
		return nil, errors.New(err.Error())
	}
	var resPayment model.Payment
	json.Unmarshal(jsonString, &resPayment)

	registerOrchestrationProcess("PAYMENT:PENDING", &listStepProcess)

	// -----------------------------------
	// update order status
	resOrder.Status = "PAYMENT-PREAPPROVED"
	now := time.Now()
	resOrder.UpdatedAt = &now

	rows, err := s.workerRepository.UpdateOrder(ctx, tx, resOrder)
	if err != nil {
		s.logger.Error().
				Ctx(ctx).
				Err(err).Send()
		return nil, err
	}
	if rows == 0 {
		s.logger.Error().
				Ctx(ctx).
				Err(erro.ErrUpdate).Send()
		return nil, erro.ErrUpdate
	}

	// fill 
	resOrder.Payment = resPayment

	return resOrder, nil
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