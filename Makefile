.PHONY: run run-cron build test test-unit test-integration test-race coverage migrate proto clean mocks

CONFIG_PATH=currency/internal/config/config.yaml
MAIN_PATH=currency/cmd/currency/main.go
CRON_PATH=currency/cmd/cron/main.go
MIGRATOR_PATH=currency/cmd/migrator/main.go
BINARY_NAME=currency.exe

# Запуск приложения
run:
	go run $(MAIN_PATH) --config=$(CONFIG_PATH)

# Запуск планировщика
run-cron:
	go run $(CRON_PATH) --config=$(CONFIG_PATH)

# Сборка бинарника
build:
	go build -o $(BINARY_NAME) $(MAIN_PATH)

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
	rm -f $(BINARY_NAME)

mocks:
	mockery
