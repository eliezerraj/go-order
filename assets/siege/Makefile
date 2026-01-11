SHELL := bash

# Default target
all: env load

# Show environment variables
env:
	@echo "---------------------------------------------------------"
	
load:
	@echo "Run Load Inventory ..."
	
	@echo "Run load all ..."
	
	echo "****** INICIO DO PROCESSO ******"; \

	@for ((i=0; i<=1000; i++)); do \
		echo "--- 02 ORDER  ---"; \
		"/mnt/c/Eliezer/workspace/github.com/go-inventory/assets/siege/02_order.sh"; \
	done

	echo "****** FIM DO PROCESSO ******"; \

.PHONY: all env load