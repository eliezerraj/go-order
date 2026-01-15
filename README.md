## go-order

   This is workload for POC purpose such as load stress test, gitaction, etc.

   The main purpose is create an order with its carts associated.

## Integration

   This is workload requires go-cart, go-inventory and go-clearance running.

   The integrations are made via http api request.

## Enviroment variables

To run in local machine for local tests creat a .env in /cmd folder

    VERSION=1.0
    ACCOUNT=aws:localhost
    APP_NAME=go-order.localhost
    PORT=7004
    ENV=dev

    DB_HOST= 127.0.0.1 
    DB_PORT=5432
    DB_NAME=postgres
    DB_MAX_CONNECTION=30
    CTX_TIMEOUT=10

    LOG_LEVEL=info #debug, info, error, warning
    OTEL_EXPORTER_OTLP_ENDPOINT = localhost:4317

    OTEL_METRICS=true
    OTEL_STDOUT_TRACER=false
    OTEL_TRACES=true

    OTEL_LOGS=true
    OTEL_STDOUT_LOG_GROUP=true
    LOG_GROUP=/mnt/c/Eliezer/log/go-order.log

    NAME_SERVICE_00=go-cart
    URL_SERVICE_00=http://localhost:7001
    HOST_SERVICE_00=go-cart
    CLIENT_HTTP_TIMEOUT_00=5

    NAME_SERVICE_01=go-inventory
    URL_SERVICE_01=http://localhost:7000
    HOST_SERVICE_01=go-inventory
    CLIENT_HTTP_TIMEOUT_01=5

    NAME_SERVICE_02=go-clearance
    URL_SERVICE_02=http://localhost:7003
    HOST_SERVICE_02=go-clearance
    CLIENT_HTTP_TIMEOUT_02=5
   
## Endpoints

    curl --location 'http://localhost:7004/info'
    curl --location 'http://localhost:7004/order/96'

    curl --location 'http://localhost:7004/order' \
    --header 'Content-Type: application/json' \
    --data '{
        "user_id": "eliezer",
        "currency": "BRL",
        "address": "st. a",
        "cart": {
                    "user_id": "eliezer",
                    "cart_item": [ 
                        {   "product": {
                                "sku": "mobile-100"
                            },
                            "currency": "BRL",
                            "quantity": 1,
                            "price": 10
                        },
                        {
                            "product": {
                                "sku": "mobile-101"
                            },
                            "currency": "BRL",
                            "quantity": 2,
                            "price": 5
                        }
                    ]
            }
    }'

    curl --location 'http://localhost:7004/checkout' \
    --header 'Content-Type: application/json' \
    --data '{
        "id": 94,
        "payment": [
            {
                "type": "CASH",
                "currency": "BRL",
                "amount": 1000
            }
        ]
    }'

## Tables

    CREATE TABLE public.order (
        id 				BIGSERIAL		NOT NULL,
        transaction_id	VARCHAR(100) 	NULL,
        fk_cart_id 		BIGSERIAL 		NOT NULL,
        user_id 		VARCHAR(100) 	NOT NULL,    
        status 			VARCHAR(100) 	NOT NULL,
        currency 		VARCHAR(100) 	NOT NULL,
        amount 			DECIMAL(10,2) 	NOT null DEFAULT 0,
        address			VARCHAR(200) 	NULL,
        created_at		timestamptz 	NOT NULL,
        updated_at		timestamptz 	NULL,
    CONSTRAINT order_pkey PRIMARY KEY (id)
    );

    ALTER TABLE public.order ADD constraint order_fk_cart_id_fkey
    FOREIGN KEY (fk_cart_id) REFERENCES public.cart(id);

    CREATE TYPE order_outbox_event_type AS ENUM (
            'order_created',
            'order_checkout'
    );
        
    CREATE TABLE order_outbox (
            event_id uuid NOT NULL,
            event_type order_outbox_event_type NOT NULL,
            event_date timestamptz NOT NULL DEFAULT now(),
            transaction_id VARCHAR(100) NULL,
            event_metadata json NOT NULL DEFAULT '{}'::json,
            event_data json NOT NULL DEFAULT '{}'::json,
            event_error json NULL DEFAULT '{}'::json,
            CONSTRAINT order_outbox_pkey PRIMARY KEY (event_id)
    )

    CREATE INDEX idx1_order_outbox ON order_outbox USING btree (event_id, event_date);