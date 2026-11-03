.PHONY: build test integration
build:
	go build -trimpath -o dist/jsreportcard-cli ./cmd/jsreportcard-cli
test:
	go test -race ./...
	go vet ./...
	bash -n scripts/*.sh
	node --check internal/checker/worker.cjs
	./scripts/test-prepare-release.sh
integration:
	./scripts/integration.sh
