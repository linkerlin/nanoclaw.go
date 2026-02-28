.PHONY: all build test test-unit test-integration test-e2e run clean setup

BINARY_NAME=nanoclaw
BUILD_DIR=./build

# 默认目标: 构建
all: build

build:
	go build -ldflags "-s -w" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/nanoclaw

# 运行所有测试
test: test-unit

# 单元测试（快速，无外部依赖）
test-unit:
	go test -v -race -short ./internal/... -run "Test(LoadConfig|Config|DB_Save|DB_Get|Message|Task|Queue|Domain)" 2>&1 | head -100

# 集成测试（需要LLM API）
test-integration:
	@if [ -z "$(OPENAI_API_KEY)" ]; then \
		echo "Warning: OPENAI_API_KEY not set, some tests will be skipped"; \
	fi
	go test -v -race -tags=integration ./internal/... -run "TestAgent" 2>&1 | head -100

# 端到端测试（完整流程）
test-e2e:
	@echo "E2E tests require full setup, running basic checks..."
	go build -o /tmp/nanoclaw_e2e ./cmd/nanoclaw && echo "Build OK" && rm /tmp/nanoclaw_e2e

# 运行并查看覆盖率
test-coverage:
	go test -v -race -coverprofile=coverage.out ./internal/...
	go tool cover -html=coverage.out -o coverage.html

run:
	go run ./cmd/nanoclaw

clean:
	rm -rf $(BUILD_DIR)
	rm -f data/*.db
	rm -f coverage.out coverage.html

setup:
	@echo "Creating nanoclaw user..."
	@sudo useradd -r -s /bin/false -M nanoclaw 2>/dev/null || true
	@sudo mkdir -p /var/run/nanoclaw
	@sudo chown nanoclaw:nanoclaw /var/run/nanoclaw 2>/dev/null || true
	@sudo chmod 755 /var/run/nanoclaw
	@echo "Setup complete"

deps:
	go get github.com/charmbracelet/bubbletea/v2@latest
	go get github.com/charmbracelet/bubbles/v2@latest
	go get github.com/charmbracelet/lipgloss@latest
	go get golang.org/x/sync/semaphore@latest
	go get github.com/yuin/gopher-lua@latest
	go get github.com/sashabaranov/go-openai@latest
	go get github.com/robfig/cron/v3@latest
	go get github.com/google/uuid@latest
	go get modernc.org/sqlite@latest
