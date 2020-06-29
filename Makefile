all: webpack build

webpack:
	npm run build

build:
	GOOS=linux GOARCH=amd64 go build -o ./dist/aws-athena-plugin_linux_amd64 ./pkg
	GOOS=darwin GOARCH=amd64 go build -o ./dist/aws-athena-plugin_darwin_amd64 ./pkg
