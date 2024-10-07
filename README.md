# go-order

POC for test purposes (CQRS)

CRUD a order and send a SQS message

## Diagram

go-order (post:add/) ==> SQS (order.fifo)

## Endpoints

+ POST /add

        {
            "order_id": "ORD-10",
            "person_id": "P-001",
            "currency": "BRL",
            "amount": 1.00,
            "tenant_id": "TENANT-200"
        }

+ GET /get/ORD-10

