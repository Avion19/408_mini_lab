TEMPL ?= $(shell go env GOPATH)/bin/templ
TAILWIND ?= ./.tools/tailwindcss

.PHONY: generate css build test run

generate:
	$(TEMPL) generate

css:
	$(TAILWIND) -i ./static/css/input.css -o ./static/css/output.css --minify

build: generate css

test: generate
	go test ./...

run: generate
	go run .
