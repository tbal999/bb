.PHONY: build test docker smoke

build: ## Build the bb binary
	go build -o bb .

test: ## Run vet and the tests
	go vet ./...
	go test ./...

docker: ## Run the tests in a Linux container
	docker build . -t bb-test
	docker run --rm bb-test

smoke: ## Unit tests + multi-user smoke test on a fake pubnix in a container
	docker build -f test/smoke/Dockerfile -t bb-smoke .
	docker run --rm bb-smoke
