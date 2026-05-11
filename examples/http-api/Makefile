.PHONY: build run test clean

BIN_DIR=bin

# 构建所有服务
build:
	@for dir in cmd/*; do \
		if [ -d "$$dir" ]; then \
			name=$$(basename $$dir); \
			echo "Building $$name..."; \
			go build -o $(BIN_DIR)/$$name ./$$dir; \
		fi \
	done

# 运行指定服务: make run SVC=user
run:
	go run ./cmd/$(SVC)

test:
	go test -v -race -cover ./...

clean:
	rm -rf $(BIN_DIR)

tidy:
	go mod tidy
