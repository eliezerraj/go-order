#docker build -t go-order .
FROM golang:1.22 As builder

RUN apt-get update && apt-get install bash && apt-get install -y --no-install-recommends ca-certificates

WORKDIR /app
COPY . .

WORKDIR /app/cmd
RUN go build -o go-order -ldflags '-linkmode external -w -extldflags "-static"'
RUN wget https://truststore.pki.rds.amazonaws.com/us-east-2/us-east-2-bundle.pem

FROM alpine

WORKDIR /app
COPY --from=builder /app/cmd/go-order .
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/cmd/us-east-2-bundle.pem .

CMD ["/app/go-order"]