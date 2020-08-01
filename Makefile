VERSION ?= v1.0

build:
	docker run --rm -v $(shell pwd):/go/src/deploy  -e CGO_ENABLED=0 -e GOBIN=/go/src/deploy -e GOOS=linux golang:1.14.3-alpine go build -a -installsuffix cgo -o /go/src/deploy/deploy /go/src/deploy/main.go

image:
	docker build -t registry.nanjun/tekton/deploy:${VERSION} .