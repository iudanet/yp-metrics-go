

build-agent::
	go build -o cmd/agent/agent cmd/agent/main.go
build-server::
	go build -o cmd/server/server cmd/server/main.go

build:: test build-agent build-server

test_iter1:: build-server
	metricstest -test.v -test.run=^TestIteration1$$ -binary-path=cmd/server/server


test_iter2::build test_iter1
	metricstest -test.v -test.run=^TestIteration2[AB]*$$ -source-path=. -agent-binary-path=cmd/agent/agent


test_iter3::build test_iter1 test_iter2
	metricstest -test.v -test.run=^TestIteration3[AB]*$$ \
	-source-path=. \
	-agent-binary-path=cmd/agent/agent \
	-binary-path=cmd/server/server

# Задание переменных
# SERVER_PORT := 8899
# ADDRESS := localhost:$(SERVER_PORT)
# test_iter1 test_iter2 test_iter3
SERVER_PORT=$(shell random unused-port)
TEMP_FILE=$(shell random tempfile)
test_iter4:: build test_iter1 test_iter2 test_iter3
	SERVER_PORT=$$(random unused-port) \
	ADDRESS="localhost:$(SERVER_PORT)" \
	TEMP_FILE=$(TEMP_FILE) \
	metricstest -test.v -test.run=^TestIteration4$ \
	  -agent-binary-path=cmd/agent/agent \
	  -binary-path=cmd/server/server \
	  -server-port=$(SERVER_PORT) \
	  -source-path=.


test::
	go test ./...
