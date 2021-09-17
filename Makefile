BINARY_NAME=helmscp


build:
	@go mod download
	@GOARCH=amd64 GOOS=darwin go build -o ${BINARY_NAME}-darwin main.go
	@GOARCH=amd64 GOOS=linux go build -o ${BINARY_NAME}-linux main.go
	@GOARCH=amd64 GOOS=windows go build -o ${BINARY_NAME}-windows main.go

clean:
	go clean
	rm ${BINARY_NAME}-darwin
	rm ${BINARY_NAME}-linux
	rm ${BINARY_NAME}-windows

dep:
	go mod download