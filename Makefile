RANDOM_PORT:=$(shell random unused-port)
SERVER_PORT=$(RANDOM_PORT)
TEMP_FILE=$(shell random tempfile)

VERSION := $(shell git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
COMMIT := $(shell git rev-parse --short HEAD)
DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

build-agent::
	go build \
	    -ldflags="-X main.buildVersion=$(VERSION) \
			-X main.buildDate=$(DATE) \
			-X main.buildCommit=$(COMMIT)" \
		-o cmd/agent/agent cmd/agent/main.go

build-server::
	go build \
	    -ldflags="-X main.buildVersion=$(VERSION) \
			-X main.buildDate=$(DATE) \
			-X main.buildCommit=$(COMMIT)" \
		-o cmd/server/server cmd/server/main.go
run-server::
	go run cmd/server/main.go -k testKey -crypto-key=test_private.pem

run-agent::
	go run cmd/agent/main.go -k testKey -r 2 -p 1 -crypto-key=test_public.pem

build::  test test_race build-agent build-server

staticlint::
	go run cmd/staticlint/main.go ./...

go_generate::
	go generate ./...
statictest::
	go vet -vettool=$(shell which statictest) ./...

test_iter1:: build-server
	metricstest -test.v -test.run=^TestIteration1$$ -binary-path=cmd/server/server


test_iter2::build test_iter1
	metricstest -test.v -test.run=^TestIteration2[AB]*$$ -source-path=. -agent-binary-path=cmd/agent/agent


test_iter3::build test_iter2
	metricstest -test.v -test.run=^TestIteration3[AB]*$$ \
	-source-path=. \
	-agent-binary-path=cmd/agent/agent \
	-binary-path=cmd/server/server


test_iter4:: build test_iter3
	SERVER_PORT=$(SERVER_PORT) \
	ADDRESS="localhost:$(SERVER_PORT)" \
	TEMP_FILE=$(shell random tempfile) \
	metricstest -test.v -test.run=^TestIteration4$ \
	  -agent-binary-path=cmd/agent/agent \
	  -binary-path=cmd/server/server \
	  -server-port=$(SERVER_PORT) \
	  -source-path=.

test_iter5::  build test_iter4
	SERVER_PORT=$(SERVER_PORT)\
	ADDRESS="localhost:$(SERVER_PORT)" \
    	TEMP_FILE=$(shell random tempfile) \
    	metricstest -test.v -test.run=^TestIteration5$ \
    	-agent-binary-path=cmd/agent/agent \
    	-binary-path=cmd/server/server \
    	-server-port=$(SERVER_PORT) \
    	-source-path=.

test_iter6::  build  test_iter5
	SERVER_PORT=$(SERVER_PORT)\
	ADDRESS="localhost:$(SERVER_PORT)" \
    	TEMP_FILE=$(shell random tempfile) \
    	metricstest -test.v -test.run=^TestIteration6$ \
    	-agent-binary-path=cmd/agent/agent \
    	-binary-path=cmd/server/server \
    	-server-port=$(SERVER_PORT) \
    	-source-path=.


test_iter7::  build test_iter6
	SERVER_PORT=$(SERVER_PORT)\
	ADDRESS="localhost:$(SERVER_PORT)" \
    	TEMP_FILE=$(shell random tempfile) \
    	metricstest -test.v -test.run=^TestIteration7$ \
    	-agent-binary-path=cmd/agent/agent \
    	-binary-path=cmd/server/server \
    	-server-port=$(SERVER_PORT) \
    	-source-path=.

test_iter8::  build test_iter7
	SERVER_PORT=$(SERVER_PORT)\
	ADDRESS="localhost:$(SERVER_PORT)" \
    	TEMP_FILE=$(shell random tempfile) \
    	metricstest -test.v -test.run=^TestIteration8$ \
    	-agent-binary-path=cmd/agent/agent \
    	-binary-path=cmd/server/server \
    	-server-port=$(SERVER_PORT) \
    	-source-path=.
test_iter9:: build test_iter8
	ADDRESS="localhost:$(SERVER_PORT)" \
    	metricstest -test.v -test.run=^TestIteration9$ \
        -file-storage-path=$(TEMP_FILE) \
    	-agent-binary-path=cmd/agent/agent \
    	-binary-path=cmd/server/server \
    	-server-port=$(SERVER_PORT) \
    	-source-path=.

test_iter10:: build test_iter9
	ADDRESS="localhost:$(SERVER_PORT)" \
    	metricstest -test.v -test.run=^TestIteration10[AB]$ \
        -file-storage-path=$(TEMP_FILE) \
    	-agent-binary-path=cmd/agent/agent \
    	-binary-path=cmd/server/server \
    	-server-port=$(SERVER_PORT) \
        -database-dsn='postgres://metrics:yandex@localhost:5432/metrics_db?sslmode=disable' \
    	-source-path=.


test_iter11:: build test_iter10
	ADDRESS="localhost:$(SERVER_PORT)" \
    	metricstest -test.v -test.run=^TestIteration11$ \
        -file-storage-path=$(TEMP_FILE) \
    	-agent-binary-path=cmd/agent/agent \
    	-binary-path=cmd/server/server \
    	-server-port=$(SERVER_PORT) \
        -database-dsn='postgres://metrics:yandex@localhost:5432/metrics_db?sslmode=disable' \
    	-source-path=.

test_iter12:: build test_iter11
	ADDRESS="localhost:$(SERVER_PORT)" \
    	metricstest -test.v -test.run=^TestIteration12$ \
        -file-storage-path=$(TEMP_FILE) \
    	-agent-binary-path=cmd/agent/agent \
    	-binary-path=cmd/server/server \
    	-server-port=$(SERVER_PORT) \
        -database-dsn='postgres://metrics:yandex@localhost:5432/metrics_db?sslmode=disable' \
    	-source-path=.

test_iter13:: build test_iter12
	ADDRESS="localhost:$(SERVER_PORT)" \
    	metricstest -test.v -test.run=^TestIteration13$ \
        -file-storage-path=$(TEMP_FILE) \
    	-agent-binary-path=cmd/agent/agent \
    	-binary-path=cmd/server/server \
    	-server-port=$(SERVER_PORT) \
        -database-dsn='postgres://metrics:yandex@localhost:5432/metrics_db?sslmode=disable' \
    	-source-path=.

test_iter14:: build  test_iter13
	ADDRESS="localhost:$(SERVER_PORT)" \
    	metricstest -test.v -test.run=^TestIteration14/TestCollectAgentMetrics/gauge/TotalAlloc$ \
    	-agent-binary-path=cmd/agent/agent \
    	-binary-path=cmd/server/server \
        -key="KeyTest" \
    	-server-port=$(SERVER_PORT) \
        -database-dsn='postgres://metrics:yandex@localhost:5432/metrics_db?sslmode=disable' \
    	-source-path=.


test:: fmt go_generate staticlint
		TEST_DATABASE_DSN='postgres://metrics:yandex@localhost:5432/metrics_db?sslmode=disable' go test  --timeout=50s ./...
test_race::
	go test -v -race ./...

fmt::
	gofmt -w .
	goimports -w .
	go run golang.org/x/tools/go/analysis/passes/fieldalignment/cmd/fieldalignment@latest -fix ./...



test_coverage::
	TEST_DATABASE_DSN='postgres://metrics:yandex@localhost:5432/metrics_db?sslmode=disable' go test -coverprofile=coverage.out ./...
	grep -v "mock_" coverage.out > coverage_filtered.out
	go tool cover -func=coverage_filtered.out
	# go tool cover -html=coverage_filtered.out


generate-test-keys:
	@echo "Generating test RSA keys..."
	# Generate private key
	openssl genrsa -out test_private.pem 2048
	# Extract public key from private key
	openssl rsa -in test_private.pem -pubout -out test_public.pem
	@echo "Test keys generated:"
	@echo "Private key: test_private.pem"
	@echo "Public key: test_public.pem"
	@echo ""
	@echo "Usage:"
	@echo "  Server: go run cmd/server/main.go -crypto-key=test_private.pem"
	@echo "  Agent:  go run cmd/agent/main.go -crypto-key=test_public.pem"

go_update_all::
	go get -u ./...
	go mod tidy
