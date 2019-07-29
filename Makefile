# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
BINARY_NAME=chat_server
BINARY_UNIX=$(BINARY_NAME)_unix

all: test build
build: 
	$(GOBUILD) -o $(BINARY_NAME) -v
test: 
	$(GOTEST) -v ./...
clean: 
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_UNIX)
run:
	$(GOBUILD) -o $(BINARY_NAME) -v ./...
	./$(BINARY_NAME)
deps:
#        $(GOGET) github.com/markbates/goth
#        $(GOGET) github.com/markbates/pop
docker: clean linux mac_docker
linux:
	env GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_NAME) -v
mac_docker:
	docker build -t chat:1 .
	docker run -p 9000:9000 chat:1