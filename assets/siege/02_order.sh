#!/bin/bash

# variabels

export AUTH_TOKEN=

export URL_HOST=https://go-api-global.architecture.caradhras.io/order

#-----------------------------------------------------
URL_GET="${URL_HOST}/info"

STATUS_CODE=$(curl -s -w " HTTP:%{http_code}" "$URL_GET" \
	--header "Content-Type: application/json" \
	--header "Authorization: $AUTH_TOKEN ")

if echo "$STATUS_CODE" | grep -q "HTTP:200"; then
	echo "HTTP:200 GET /info"
else
	echo -e "\e[31m** ERROR $STATUS_CODE ==> /info\e[0m"
fi

#-----------------------------------------------------

RANDOM_ORDER=$((RANDOM % 999 + 1))
URL_GET="${URL_HOST}/order/${RANDOM_ORDER}"

STATUS_CODE=$(curl -s -w " HTTP:%{http_code}" "$URL_GET" \
	--header "Content-Type: application/json" \
	--header "Authorization: $AUTH_TOKEN ")

if echo "$STATUS_CODE" | grep -q "HTTP:200"; then
  	echo "HTTP:200 GET /order/${RANDOM_ORDER}"
elif echo "$STATUS_CODE" | grep -q "HTTP:404"; then
	echo -e "\e[38;2;255;165;0m** ERROR $STATUS_CODE ==> /order/${RANDOM_ORDER}\e[0m"
else
	echo -e "\e[31m** ERROR $STATUS_CODE ==> /order/${RANDOM_ORDER}\e[0m"
fi

#------------------------------------------
RANDOM_INV=$((RANDOM % 99 + 1))
RANDOM_VAL=$((RANDOM % 9 + 1))

URL_POST="${URL_HOST}/order"
PAYLOAD='{"user_id": "siege","currency": "BRL","address": "st. a","cart":{"user_id": "eliezer","cart_item":[{"product": {"sku": "JUICE-'$RANDOM_INV'"},"currency": "BRL","quantity": 1'${RANDOM_VAL}',"price": '${RANDOM_VAL}'},{"product": {"sku": "MOBILE-'$RANDOM_INV'"},"currency": "BRL","quantity": '${RANDOM_VAL}',"price": '${RANDOM_VAL}' }]}}'

STATUS_CODE=$(curl -s -w " HTTP:%{http_code}" "$URL_POST" \
	--header "Content-Type: application/json" \
	--header "Authorization: $AUTH_TOKEN" \
	--data "$PAYLOAD")

if echo "$STATUS_CODE" | grep -q "HTTP:201"; then
  	echo "HTTP:201 POST order"
elif echo "$STATUS_CODE" | grep -q "HTTP:404"; then
	echo -e "\e[38;2;255;165;0m** ERROR $STATUS_CODE ==> /order\e[0m"
else
	echo -e "\e[31m** ERROR $STATUS_CODE ==> /order\e[0m"
fi

#------------------------------------------
RANDOM_ORDER=$((RANDOM % 999 + 1))
RANDOM_VAL=$((RANDOM % 999 + 1))

URL_POST="${URL_HOST}/checkout"
PAYLOAD='{"id": '${RANDOM_ORDER}',"payment": [{"type": "CASH","currency": "BRL","amount": '${RANDOM_VAL}' }]}'

STATUS_CODE=$(curl -s -w " HTTP:%{http_code}" "$URL_POST" \
	--header "Content-Type: application/json" \
	--header "Authorization: $AUTH_TOKEN" \
	--data "$PAYLOAD")

if echo "$STATUS_CODE" | grep -q "HTTP:200"; then
  	echo "HTTP:200 POST /checkout"
elif echo "$STATUS_CODE" | grep -q "HTTP:404"; then
	echo -e "\e[38;2;255;165;0m** ERROR $STATUS_CODE ==> /checkout\e[0m"
else
	echo -e "\e[31m** ERROR $STATUS_CODE ==> /checkout\e[0m"
fi