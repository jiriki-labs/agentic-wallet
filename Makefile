BINARY_DIR := bin
JIRIKI_BIN := $(BINARY_DIR)/jiriki
GROCERY_BIN := $(BINARY_DIR)/grocery-demo

.PHONY: build test vet lint demo-dry demo-mock grocery-dev clean

build:
	@mkdir -p $(BINARY_DIR)
	go build -o $(JIRIKI_BIN) ./cmd/jiriki
	go build -o $(GROCERY_BIN) ./cmd/grocery-demo

# TypeScript Grocery402 stack (Nest API :4402 + Next.js :3020)
grocery-dev:
	cd apps/grocery && npm install && npm run dev

test:
	go test ./...

vet:
	go vet ./...

lint:
	@which golangci-lint > /dev/null 2>&1 && golangci-lint run || (echo "golangci-lint not installed, skipping" && exit 0)

demo-dry:
	@echo "Requires jiriki with policy mode dry-run, e.g.: jiriki up --policy configs/policy.dry-run.example.yaml"
	./scripts/demo.sh

demo-mock:
	@echo "Running Go x402 tests + Grocery API e2e (no funded wallet)"
	./scripts/demo-mock.sh

clean:
	rm -rf $(BINARY_DIR)
