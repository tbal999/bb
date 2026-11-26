.PHONY: build test smoke demo

build: ## Build the bb binary
	go build -o bb .

test: ## Run vet and the tests
	go vet ./...
	go test ./...

smoke: ## Unit tests + multi-user smoke test on a fake pubnix in a container
	docker build -f test/smoke/Dockerfile -t bb-smoke .
	docker run --rm bb-smoke

demo: ## Shell on a fake pubnix (users admin, alice, bob) with some boards to play with
	docker build -f test/smoke/Dockerfile -t bb-smoke .
	docker run --rm -it bb-smoke bb-demo
