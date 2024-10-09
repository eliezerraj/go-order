# Expected environment variables (simply export them in your terminal and the commands should work):
# export PATH=$PATH:~/liquibase-4.16.1

# DB_HOST=localhost
# DB_PORT=5432
# DB_NAME=db_eitri
# DB_USERNAME=postgres
# DB_PASSWORD=postgres
# DB_SCHEMA=eitri

docker:
	docker build -t go-order .
#	sudo docker run --env-file ./.env eitri-migration

# This command will apply all migrations configured in `changelog.yaml`
#migrate:
#	liquibase update --url "jdbc:postgresql://${DB_HOST}:${DB_PORT}/${DB_NAME}?currentSchema=${DB_SCHEMA}" --username=${DB_USERNAME} --password=${DB_PASSWORD}