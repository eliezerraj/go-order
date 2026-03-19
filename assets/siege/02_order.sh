#!/bin/bash

# variabels

export AUTH_TOKEN=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ0b2tlbl91c2UiOiJhY2Nlc3MiLCJpc3MiOiJsYW1iZGEtZ28taWRlbnRpZHkiLCJ2ZXJzaW9uIjoiMS4yIiwidXNlcm5hbWUiOiJhZG1pbi0xMDEiLCJqd3RfaWQiOiIyOGJkNWEyYy1iMjE4LTRiODUtODE3My1iZGVkMzdhNzM1OWYiLCJraWQiOiJhdXRoLWtleTpzZXJ2ZXItcHVibGljLmtleSIsInRpZXIiOiJ0aWVyMSIsImFwaV9hY2Nlc3Nfa2V5IjoiQVBJX0FDQ0VTU19LRVlfQURNSU5fMDAxIiwic2NvcGUiOlsidG9vbDppbmZvIiwidG9vbDpoZWFsdGgiLCJ0b29sOmdldF9wcm9kdWN0IiwiYWRtaW4iXSwiZXhwIjoxNzc0MTgyNjY5fQ.202WuDQxKubP7ZkXV0QZyr-GNr4W5wMJNzhOzr7SV_0

export URL_HOST=https://go-api-global.architecture.caradhras.io/order
export URL_HOST=http://localhost:7004

# -----------------------------
# normal random generator
# mean, std, min, max
# -----------------------------
normal_rand() {
    local mean=$1
    local std=$2
    local min=$3
    local max=$4

    awk -v mean="$mean" -v std="$std" -v min="$min" -v max="$max" '
    BEGIN {
		srand(systime() + PROCINFO["pid"])
		for (attempt = 0; attempt < 100; attempt++) {
			u1 = rand()
			if (u1 == 0) {
				u1 = 0.000001
			}

			u2 = rand()
			z = sqrt(-2 * log(u1)) * cos(2 * 3.141592653589793 * u2)
			val = mean + z * std

			if (val >= min && val <= max) {
				printf "%d", int(val + 0.5)
				exit
			}
		}

		fallback = mean
		if (fallback < min) fallback = min
		if (fallback > max) fallback = max

		printf "%d", int(fallback + 0.5)
    }'
}

json_get() {
	local field=$1

	python3 -c 'import json, sys
data = json.load(sys.stdin)
value = data.get(sys.argv[1], "")
print("" if value is None else value)' "$field"
}

#--------------- POST ORDER ---------------------------

#RANDOM_PRODUCT=$((RANDOM % 99 + 1))
#PRODUCT_1="milk-$RANDOM_PRODUCT"

PRODUCT_1="coffee-23"
PRICE_1=80

RANDOM_MEAN=$((RANDOM % 20 + 22))
RANDOM_STD=$((RANDOM_MEAN / 4))
if [ "$RANDOM_STD" -lt 2 ]; then
	RANDOM_STD=2
fi
RANDOM_MIN=$((RANDOM_MEAN - (3 * RANDOM_STD)))
if [ "$RANDOM_MIN" -lt 1 ]; then
	RANDOM_MIN=1
fi

RANDOM_MAX=$((RANDOM_MEAN + (3 * RANDOM_STD)))
RANDOM_QTD=$(normal_rand $RANDOM_MEAN $RANDOM_STD $RANDOM_MIN $RANDOM_MAX)

PAYLOAD=$(cat <<EOF
	{
		"user_id": "siege",
		"currency": "BRL",
		"address": "st. a",
		"cart": {
			"user_id": "siege",
			"cart_item": [
				{
					"product": {
							"sku": "${PRODUCT_1}"
						},
						"currency": "BRL",
						"quantity": ${RANDOM_QTD},
						"price": ${PRICE_1}
				}
			]
		}
	}
EOF
)

echo "------------------------------"
#echo "$PAYLOAD"
echo  "sku": "${PRODUCT_1} quantity: ${RANDOM_QTD} price: ${PRICE_1}"
echo "------------------------------"

#----------------------- INFO------------------------------
URL_GET="${URL_HOST}/info"

STATUS_CODE=$(curl -s -w " HTTP:%{http_code}" "$URL_GET" \
	--header "Content-Type: application/json" \
	--header "Authorization: $AUTH_TOKEN ")

if echo "$STATUS_CODE" | grep -q "HTTP:200"; then
	echo "HTTP:200 GET /info"
else
	echo -e "\e[31m** ERROR $STATUS_CODE ==> /info\e[0m"
fi

#--------------- POST ORDER ---------------------------
URL_POST="${URL_HOST}/order"
ORDER_RESPONSE=$(curl -s -w "\n%{http_code}" "$URL_POST" \
	--header "Content-Type: application/json" \
	--header "Authorization: $AUTH_TOKEN" \
	--data "$PAYLOAD")

ORDER_HTTP_CODE=$(printf '%s\n' "$ORDER_RESPONSE" | tail -n1)
ORDER_BODY=$(printf '%s\n' "$ORDER_RESPONSE" | sed '$d')
ORDER_ID=$(printf '%s' "$ORDER_BODY" | json_get id)

if [ "$ORDER_HTTP_CODE" = "201" ]; then
  	echo "HTTP:201 POST /order id = ${ORDER_ID}"
elif [ "$ORDER_HTTP_CODE" = "404" ]; then
	echo -e "\e[38;2;255;165;0m** ERROR $ORDER_BODY HTTP:$ORDER_HTTP_CODE ==> /order id = ${ORDER_ID}\e[0m"
else
	echo -e "\e[31m** ERROR $ORDER_BODY HTTP:$ORDER_HTTP_CODE ==> /order id = ${ORDER_ID}\e[0m"
fi

#--------------- POST CHECKOUT ---------------------------
RANDOM_VAL=$((RANDOM % 999 + 1))

URL_POST="${URL_HOST}/checkout"
PAYLOAD='{"id": '${ORDER_ID}',"payment": [{"type": "CASH","currency": "BRL","amount": '${RANDOM_VAL}' }]}'

STATUS_CODE=$(curl -s -w " HTTP:%{http_code}" "$URL_POST" \
	--header "Content-Type: application/json" \
	--header "Authorization: $AUTH_TOKEN" \
	--data "$PAYLOAD")

if echo "$STATUS_CODE" | grep -q "HTTP:200"; then
  	echo "HTTP:200 POST /checkout id = ${ORDER_ID}"
elif echo "$STATUS_CODE" | grep -q "HTTP:404"; then
	echo -e "\e[38;2;255;165;0m** ERROR $STATUS_CODE ==> /checkout id = ${ORDER_ID}\e[0m"
else
	echo -e "\e[31m** ERROR $STATUS_CODE ==> /checkout id = ${ORDER_ID}\e[0m"
fi

#-----------------------GET ORDER------------------------------
URL_GET="${URL_HOST}/order/${ORDER_ID}"

STATUS_CODE=$(curl -s -w " HTTP:%{http_code}" "$URL_GET" \
	--header "Content-Type: application/json" \
	--header "Authorization: $AUTH_TOKEN ")

if echo "$STATUS_CODE" | grep -q "HTTP:200"; then
  	echo "HTTP:200 GET /order/${ORDER_ID}"
elif echo "$STATUS_CODE" | grep -q "HTTP:404"; then
	echo -e "\e[38;2;255;165;0m** ERROR $STATUS_CODE ==> /order/${ORDER_ID}\e[0m"
else
	echo -e "\e[31m** ERROR $STATUS_CODE ==> /order/${ORDER_ID}\e[0m"
fi
