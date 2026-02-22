.PHONY: run build test migrate proto clean

CONFIG_PATH=currency/internal/config/config.yaml
MAIN_PATH=currency/cmd/currency/main.go
MIGRATOR_PATH=currency/cmd/migrator/main.go
BINARY_NAME=currency.exe

# Запуск приложения
run:
	go run $(MAIN_PATH) --config=$(CONFIG_PATH)

# Сборка бинарника
build:
	go build -o $(BINARY_NAME) $(MAIN_PATH)

# Запуск тестов
test:
	go test ./...

# Запуск мигратора
migrate:
	go run $(MIGRATOR_PATH) --config=$(CONFIG_PATH)

# Генерация gRPC кода из proto
proto:
	protoc --go_out=. --go-grpc_out=. proto/currency/currency_service.proto

# Линтер
lint:
	golangci-lint run ./...

# Удалить собранный бинарник
clean:
	rm -f $(BINARY_NAME)
