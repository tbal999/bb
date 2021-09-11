.PHONY: docker
docker: ## Run the Project in the docker container used for it's action.
	docker build . -t bb
	docker run -v $(shell pwd):/github/workspace bb