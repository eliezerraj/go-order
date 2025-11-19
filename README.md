# go-order

    workload for POC purpose

# tables

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