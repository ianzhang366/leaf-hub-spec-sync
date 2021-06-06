
# This makefile defines the following targets
#
#   - all (default) - downloads vendor libs, and build executable
#   - vendor - download all third party libraries and puts them inside vendor directory
#   - clean-vendor - removes third party libraries from vendor directory
#   - lh-spec-sync - builds leaf-hub-spec-sync as an executable and puts it under build/bin
#   - docker-build - builds docker image locally for running the components using docker
#   - docker-push - pushes the local docker image to 'docker.io' docker registry
#   - clean - cleans the build area (all executables under build/bin)
#   - clean-all - superset of 'clean' that also removes vendor dir

.PHONY: all				##downloads vendor libs, and build executable
all: vendor lh-spec-sync

.PHONY: vendor			##download all third party libraries and puts them inside vendor directory
vendor:
	@go mod vendor

.PHONY: clean-vendor			##removes third party libraries from vendor directory
clean-vendor:
	-@rm -rf vendor

.PHONY: lh-spec-sync			##builds leaf-hub-spec-sync as an executable and puts it under build/bin
lh-spec-sync:
	@go build -o build/bin/lh-spec-sync cmd/main.go

.PHONY: docker-build			##builds docker image locally for running the components using docker
docker-build: all
	@docker build -t leaf-hub-spec-sync -f build/Dockerfile .

.PHONY: docker-push			##pushes the local docker image to 'docker.io' docker registry
docker-push: docker-build
	@docker tag leaf-hub-spec-sync ${IMAGE}
	@docker push ${IMAGE}

.PHONY: clean			##cleans the build area (all executables under build/bin)
clean:
	@rm -rf build/bin

.PHONY: clean-all			##superset of 'clean' that also removes vendor dir
clean-all: clean-vendor clean

.PHONY: help				##show this help message
help:
	@echo "usage: make [target]\n"; echo "options:"; \fgrep -h "##" $(MAKEFILE_LIST) | fgrep -v fgrep | sed -e 's/\\$$//' | sed -e 's/##//' | sed 's/.PHONY:*//' | sed -e 's/^/  /'; echo "";