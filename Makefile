

build-agent::
	go build -o cmd/agent/agent cmd/agent/main.go
build-server::
	go build -o cmd/server/server cmd/server/main.go

build:: build-agent build-server

test_iter1:: build-server
	metricstest -test.v -test.run=^TestIteration1$$ -binary-path=cmd/server/server


test_iter2:: test_iter1 build
	metricstest -test.v -test.run=^TestIteration2[AB]*$$ -source-path=. -agent-binary-path=cmd/agent/agent


test_iter3:: test_iter1 test_iter2 build
	metricstest -test.v -test.run=^TestIteration3[AB]*$$ \
	-source-path=. \
	-agent-binary-path=cmd/agent/agent \
	-binary-path=cmd/server/server
