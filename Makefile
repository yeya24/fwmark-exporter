GOARCH := $(if $(GOARCH),$(GOARCH),amd64)
GO=GO15VENDOREXPERIMENT="1" CGO_ENABLED=0 GOARCH=$(GOARCH) GO111MODULE=on go

PACKAGE_LIST  := go list ./...| grep -vE "cmd"
PACKAGES  := $$($(PACKAGE_LIST))
FILES_TO_FMT  := $(shell find . -path -prune -o -name '*.go' -print)

LDFLAGS += -X "main.version=$(shell git describe --tags --dirty --always)"
LDFLAGS += -X "main.commit=$(shell git rev-parse HEAD)"
LDFLAGS += -X "main.date=$(shell date -u '+%Y-%m-%d %I:%M:%S')"

GOBUILD=$(GO) build -ldflags '$(LDFLAGS)'

APP			:= fwmark-exporter
VERSION		:= 0.2

REGISTRY			:= docker-us.byted.org/cdn/
IMAGE				= $(APP):$(VERSION)
LATEST_IMAGE		= $(APP):latest
FULL_IMAGE			= $(REGISTRY)$(IMAGE)
LATEST_FULL_IMAGE	= $(REGISTRY)$(LATEST_IMAGE)

all: format build

format: vet fmt

fmt:
	@echo "gofmt"
	@gofmt -w ${FILES_TO_FMT}
	@git diff --exit-code .

build: mod
	$(GOBUILD) -o ./bin/fwmark-exporter main.go

vet:
	go vet ./...

mod:
	@echo "go mod tidy"
	GO111MODULE=on go mod tidy
	@git diff --exit-code -- go.mod

docker-build:
	docker build . \
		--build-arg CODE_LOGIN=${CODE_LOGIN} \
		--build-arg CODE_TOKEN=${CODE_TOKEN} \
		-t $(IMAGE)
	docker tag $(IMAGE) $(LATEST_IMAGE)
	docker image prune -f

docker-push: build
	docker tag $(IMAGE) $(FULL_IMAGE)
	docker push $(FULL_IMAGE)
	docker tag $(IMAGE) $(LATEST_FULL_IMAGE)
	docker push $(LATEST_FULL_IMAGE)
