all: clean media

clean:
	@rm -rf ./wallet-apidoc*

media:
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -o ./wallet-apidoc