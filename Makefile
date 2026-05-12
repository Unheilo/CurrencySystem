.PHONY: run run-cron run-gateway run-auth build build-gateway build-auth test test-unit test-integration test-race coverage migrate proto clean mocks

CONFIG_PATH=currency/internal/config/config.yaml
GATEWAY_CONFIG_PATH=gateway/internal/config/config.example.yaml
AUTH_CONFIG_PATH=auth/internal/config/config.example.yaml
MAIN_PATH=currency/cmd/currency/main.go
CRON_PATH=currency/cmd/cron/main.go
MIGRATOR_PATH=currency/cmd/migrator/main.go
GATEWAY_PATH=gateway/cmd/main.go
AUTH_PATH=auth/cmd/main.go
BINARY_NAME=currency.exe
GATEWAY_BINARY=gateway.exe
AUTH_BINARY=auth.exe

# Запуск приложения
run:
	go run $(MAIN_PATH) --config=$(CONFIG_PATH)

# Запуск планировщика
run-cron:
	go run $(CRON_PATH) --config=$(CONFIG_PATH)

# Запуск gateway (требует JWT_SECRET в env)
run-gateway:
	go run $(GATEWAY_PATH) --config=$(GATEWAY_CONFIG_PATH)

# Сборка бинарника
build:
	go build -o $(BINARY_NAME) $(MAIN_PATH)

# Сборка gateway
build-gateway:
	go build -o $(GATEWAY_BINARY) $(GATEWAY_PATH)

# Запуск auth (требует JWT_SECRET в env)
run-auth:
	go run $(AUTH_PATH) --config=$(AUTH_CONFIG_PATH)

# Сборка auth
build-auth:
	go build -o $(AUTH_BINARY) $(AUTH_PATH)

# Запуск тестов
test: test-unit

# Запуск тестов с race флагом
test-race:
	go test -race ./...

# Пакеты, исключённые из coverage: сгенерированный protobuf-код и моки
COVER_PKG=$(shell go list ./... | grep -vE '/pkg/currency$$|/mocks$$' | tr '\n' ',' | sed 's/,$$//')

coverage:
	CONFIG_PATH=$(CURDIR)/$(CONFIG_PATH) go test -tags=integration -coverprofile=coverage.out -coverpkg=$(COVER_PKG) ./...
	go tool cover -func=coverage.out | tail -1
	go tool cover -html=coverage.out -o coverage.html
	@echo "open coverage.html in browser"

# Запуск юнит-тестов
test-unit:
		go test -short -race ./...

# Запуск интеграционных тестов
test-integration:
	CONFIG_PATH=$(CURDIR)/$(CONFIG_PATH) go test -tags=integration -race -v ./...

# Запуск мигратора
migrate:
	go run $(MIGRATOR_PATH) --config=$(CONFIG_PATH)

# Генерация gRPC кода из proto
PROTOC=$(shell which protoc || echo protoc)

proto:
	$(PROTOC) --go_out=. --go-grpc_out=. proto/currency/currency_service.proto

# Линтер
lint:
	golangci-lint run ./...

# Удалить собранный бинарник
clean:
	rm -f $(BINARY_NAME) $(GATEWAY_BINARY) $(AUTH_BINARY)

mocks:
	mockery
