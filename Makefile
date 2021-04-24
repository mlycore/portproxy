LDFLAGS := "-s -w"
.PHONY: test
test:
	cd pkg; go test -tags=local ./...
.PHONY: build
build:
	mkdir -p bin
	go build -ldflags=${LDFLAGS} -o bin/portproxy
docker:
	docker build -t mworks/portproxy:`date +%F`	
run:
	./bin/portproxy -backend="127.0.0.1:3306" -bind=":3307"
portproxy:
	make clean && make build && make run
trace:
	LOGLEVEL=TRACE ./bin/portproxy -backend="127.0.0.1:3306" -bind=":3307"
debug:
	LOGLEVEL=DEBUG ./bin/portproxy -backend="127.0.0.1:3306" -bind=":3307" -verbose
clean:
	rm -r bin
