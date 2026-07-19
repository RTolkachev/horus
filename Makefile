# Development database (docker-compose.yml). Variables:
#   MYSQL_VERSION  mysql image tag (default 8.0, the support floor; 8.4 = LTS)
#   MYSQL_PORT     host port (default 3307)
MYSQL_VERSION ?= 8.0
MYSQL_PORT    ?= 3307
export MYSQL_VERSION MYSQL_PORT

DSN = horus:horus@tcp(127.0.0.1:$(MYSQL_PORT))/horus_test

.PHONY: db-up db-down db-reset db-shell db-logs build test test-integration

db-up: ## start mysql and wait until healthy
	docker compose up -d --wait

db-down: ## stop mysql, keep data
	docker compose down

db-reset: ## wipe data and start a fresh instance
	docker compose down -v
	$(MAKE) db-up

db-shell: ## mysql client on the dev database
	docker exec -it horus-mysql mysql -uhorus -phorus horus_test

db-logs:
	docker compose logs -f mysql

build:
	go build ./...

test: ## unit tests only (no database needed)
	go test ./...

test-integration: db-up ## integration tests against the container
	HORUS_TEST_DSN="$(DSN)" go test -tags integration ./internal/dbdriver/...
