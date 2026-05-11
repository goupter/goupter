.PHONY: build test clean lint fmt proto deps tidy install run-example-http run-example-grpc all release-test release-snapshot release tag install-goreleaser help

# Default target
.DEFAULT_GOAL := help

# Help command
help: ## 显示此帮助信息
	@echo "Goupter Makefile 命令:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "快速开始:"
	@echo "  make build          # 构建 CLI 工具"
	@echo "  make test           # 运行测试"
	@echo "  make release-test   # 测试发布配置"
	@echo ""


# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt
GOLINT=golangci-lint

# Binary name
BINARY_NAME=goupter
BINARY_DIR=bin

build: ## 构建 CLI 工具
	$(GOBUILD) -o $(BINARY_DIR)/$(BINARY_NAME) ./cmd/goupter

test: ## 运行测试
	$(GOTEST) -v -race -cover ./...

test-coverage: ## 运行测试并生成覆盖率报告
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

clean: ## 清理构建产物
	rm -rf $(BINARY_DIR)
	rm -f coverage.out coverage.html

lint: ## 运行代码检查
	$(GOLINT) run ./...

fmt: ## 格式化代码
	$(GOFMT) -s -w .

proto: ## 生成 protobuf 文件
	protoc --go_out=. --go-grpc_out=. api/proto/*.proto

deps: ## 下载依赖
	$(GOGET) -v -t -d ./...

tidy: ## 整理依赖
	$(GOMOD) tidy

install: build ## 安装 CLI 工具到 GOPATH
	cp $(BINARY_DIR)/$(BINARY_NAME) $(GOPATH)/bin/

run-example-http: ## 运行 HTTP API 示例
	$(GOCMD) run ./examples/http-api

run-example-grpc: ## 运行 gRPC 示例
	$(GOCMD) run ./examples/grpc-service

all: fmt lint test build ## 运行所有检查和构建

release-test: ## 测试 GoReleaser 配置
	goreleaser build --snapshot --clean

release-snapshot: ## 创建快照版本（用于测试）
	goreleaser release --snapshot --clean

release: ## 发布新版本（需要 git tag）
	goreleaser release --clean

tag: ## 创建并推送新的 git tag
	@read -p "Enter version (e.g., v1.0.0): " version; \
	git tag -a $$version -m "Release $$version"; \
	git push origin $$version; \
	echo "Tag $$version created and pushed"

install-goreleaser: ## 安装 GoReleaser 工具
	@command -v goreleaser >/dev/null 2>&1 || { \
		echo "Installing GoReleaser..."; \
		brew install goreleaser || go install github.com/goreleaser/goreleaser@latest; \
	}
	@echo "GoReleaser is installed"
