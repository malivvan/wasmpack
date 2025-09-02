.PHONY: sample build serve

sample:
	@go run ./main.go -o ./sample/main.js -n init ./sample/main.go

serve:
	python -m http.server 8222 -d sample

build:
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o build/wasmexec .