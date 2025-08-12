.PHONY: build
build: generate
	go build -o cmd/server/server ./cmd/server

.PHONY: clean-gen-proto
clean-gen-proto:
	find proto -type f ! -name "*.proto" -delete

.PHONY: generate
generate: auth.proto health.proto keeper.proto

auth.proto:
	protoc \
		--go_out=. \
		--go_opt=paths=source_relative \
		--go-grpc_out=. \
		--go-grpc_opt=paths=source_relative \
		proto/v1/auth/auth.proto

health.proto:
	protoc \
		--go_out=. \
		--go_opt=paths=source_relative \
		--go-grpc_out=. \
		--go-grpc_opt=paths=source_relative \
		proto/v1/health/health.proto

keeper.proto:
	protoc \
		--go_out=. \
		--go_opt=paths=source_relative \
		--go-grpc_out=. \
		--go-grpc_opt=paths=source_relative \
		proto/v1/keeper/keeper.proto

.PHONY : lint
lint:
	golangci-lint run --fix -c .golangci.yml > ./golangci-lint/report-unformatted.json

.PHONY : _golangci-lint-format-report
_golangci-lint-format-report:
	cat ./golangci-lint/report-unformatted.json | jq > ./golangci-lint/report.json
	rm ./golangci-lint/report-unformatted.json

.PHONY : test
test:
	go test ./... -tags integration_tests -race -coverprofile=cover.out -covermode=atomic
	grep -v "/pkg/pgcontainer/" cover.out > cover.filtered.out
