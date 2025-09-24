.PHONY : run
run: server.crt
	docker-compose up

.PHONY: build
build: generate
	go build -o cmd/server/server ./cmd/server

.PHONY: clean-gen-proto
clean-gen-proto:
	find proto -type f ! -name "*.proto" -delete

.PHONY: generate
generate: auth.proto health.proto keeper.proto agent.proto

.PHONY: auth.proto
auth.proto:
	protoc \
		--go_out=. \
		--go_opt=paths=source_relative \
		--go-grpc_out=. \
		--go-grpc_opt=paths=source_relative \
		proto/v1/auth/auth.proto

.PHONY: health.proto
health.proto:
	protoc \
		--go_out=. \
		--go_opt=paths=source_relative \
		--go-grpc_out=. \
		--go-grpc_opt=paths=source_relative \
		proto/v1/health/health.proto

.PHONY: metadata.proto
metadata.proto:
	protoc \
		--go_out=. \
		--go_opt=paths=source_relative \
		proto/v1/metadata/metadata.proto

.PHONY: common.proto
common.proto:
	protoc \
		--go_out=. \
		--go_opt=paths=source_relative \
		proto/v1/common/common.proto

.PHONY: keeper.proto
keeper.proto: metadata.proto common.proto
	protoc \
		--go_out=. \
		--go_opt=paths=source_relative \
		--go-grpc_out=. \
		--go-grpc_opt=paths=source_relative \
		proto/v1/keeper/keeper.proto

.PHONY: auth.proto
agent.proto: metadata.proto common.proto
	protoc \
		--go_out=. \
		--go_opt=paths=source_relative \
		proto/v1/agent/agent.proto

.PHONY: mocks
mocks:
	docker run --rm \
		-v $(realpath .):/src \
		-w /src \
		vektra/mockery:latest

.PHONY : lint
lint:
	golangci-lint run -c .golangci.yml
	cat ./golangci-lint/report-unformatted.json | jq > ./golangci-lint/report.json
	rm ./golangci-lint/report-unformatted.json

.PHONY : test
test:
	go test ./... -tags integration_tests -race -coverprofile=cover.out -covermode=atomic
	grep -v \
		-e "/mocks/" \
		-e "/proto/" \
		cover.out > cover.filtered.out

.PHONY : check-coverage
check-coverage:
	go tool cover -html cover.filtered.out

sqlc:
	sqlc generate

.PHONY: migrate
migrate:
	docker run --rm \
		-v $(realpath ./sql/migrations):/migrations \
		--network=gophkeeper-network \
		migrate/migrate:v4.18.3 \
			-path=/migrations \
			-database postgres://gophkeeper:gophkeeper@gophkeeper-database:5432/gophkeeper?sslmode=disable \
			up

.PHONY: migrate-force
migrate-force:
	docker run --rm \
	-v $(realpath ./sql/migrations):/migrations \
	--network=gophkeeper-network \
	migrate/migrate:v4.18.3 \
		-path=/migrations \
		-database postgres://gophkeeper:gophkeeper@gophkeeper-database:5432/gophkeeper?sslmode=disable \
		drop -f

.PHONY : build-server
build-server:
	docker build -t gophkeeper-server:dev -f ./build/dockerfile.server .

.PHONY : run-server
run-server:
	docker run --rm \
	-p 50051:50051 \
	-e RUN_ADDRESS=":50051" \
	-e DATABASE_URI="postgres://gophkeeper:gophkeeper@gophkeeper-database:5432/gophkeeper?sslmode=disable" \
	-e SECRET_KEY="dev-secret" \
	--network gophkeeper-network \
	--name gk-server gophkeeper-server:dev

certs:
	mkdir "./certs"

ca.key: certs
	openssl genrsa -out ./certs/ca.key 4096

ca.crt: ca.key
	openssl req -x509 -new -nodes -key ./certs/ca.key -sha256 -days 365 \
	-subj "/CN=MyLocalCA" -out ./certs/ca.crt

server.key: certs
	openssl genrsa -out ./certs/server.key 2048

server.csr: server.key
	openssl req -new -key ./certs/server.key -subj "/CN=127.0.0.1" -out ./certs/server.csr

server_san.cnf: certs
	printf "[SAN]\nsubjectAltName=IP:127.0.0.1,DNS:localhost" > ./certs/server_san.cnf

server.crt: ca.crt server.csr server_san.cnf
	openssl x509 -req \
	-in ./certs/server.csr \
	-CA ./certs/ca.crt -CAkey ./certs/ca.key -CAcreateserial \
	-out ./certs/server.crt \
	-days 365 -sha256 \
	-extfile ./certs/server_san.cnf -extensions SAN

