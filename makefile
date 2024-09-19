include .envrc

# ==================================================================================== #
# helpers
# ==================================================================================== #
## help: print this help message
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

.PHONY: confirm
confirm:
	@echo -n 'Are you sure? [y/N] ' && read ans && [ $${ans:-N} = y ]

## api.preview: preview the current cmd/api application
.PHONY: api.preview
api.preview:
	go run ./cmd/api \
	-db-dsn=${DATABASE_URL} \
	-cors-allowed-origins=${ALLOWED_ORIGINS} \
	-cors-allowed-headers=${ALLOWED_HEADERS} \
	-cors-allowed-methods=${ALLOWED_METHODS}

## api.build: build the cmd/api application
.PHONY: api.build
api.build:
	@echo 'Building cmd/api...'
	go build -o=./bin/api ./cmd/api
	GOOS=linux GOARCH=amd64 go build -o=./bin/linux_amd64/api ./cmd/api

## api.run: run the compiled cmd/api application
.PHONY: api.run
api.run:
	./bin/api \
	-db-dsn=${DATABASE_URL} \
	-cors-allowed-origins=${ALLOWED_ORIGINS} \
	-cors-allowed-headers=${ALLOWED_HEADERS} \
	-cors-allowed-methods=${ALLOWED_METHODS}

## db.psql: connect to the database using psql
.PHONY: db.psql
db.psql:
	psql ${DATABASE_URL}

## db.migrations.new name=$1: create a new database migration
.PHONY: db.migrations.new
db.migrations.new:
	@echo 'Creating migration files for ${name}...'
	migrate create -seq -ext=.sql -dir=./migrations ${name}

## db.migrations.up: apply all up database migrations
.PHONY: db.migrations.up
db.migrations.up: confirm
	@echo 'Running up migrations...'
	migrate -path ./db/migrations -database ${DATABASE_URL} up

## db.migrations.down: apply all down database migrations
.PHONY: db.migrations.down
db.migrations.down: confirm
	@echo 'Running down migrations...'
	migrate -path ./db/migrations -database ${DATABASE_URL} down

## audit: tidy dependencies and test all code
.PHONY: audit
audit:
	@echo 'Tidying and verifying module dependencies...'
	go mod tidy
	go mod verify
	@echo 'Running tests...'
	go test -race -vet=off ./...