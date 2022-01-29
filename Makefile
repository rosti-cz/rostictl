VERSION=0.6
TESTIMAGE_VERSION=v1

.PHONY: all
all: release

.PHONY: preparation
preparation:
	sed -i "s/const version = \"[0-9\.]*\"/const version = \"${VERSION}\"/" main.go
	mkdir -p bin
	go mod tidy

bin/rostictl-${VERSION}.linux.arm: preparation
	env GOOS=linux GOARCH=arm CGO_ENABLED=0 go build -o bin/rostictl-${VERSION}.linux.arm

bin/rostictl-${VERSION}.linux.arm64: preparation
	env GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o bin/rostictl-${VERSION}.linux.arm64
	
bin/rostictl-${VERSION}.linux.i386: preparation
	env GOOS=linux GOARCH=386 CGO_ENABLED=0 go build -o bin/rostictl-${VERSION}.linux.i386
	
bin/rostictl-${VERSION}.linux.amd64: preparation
	env GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bin/rostictl-${VERSION}.linux.amd64

bin/rostictl-${VERSION}.darwin.amd64: preparation
	env GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -o bin/rostictl-${VERSION}.darwin.amd64
	
bin/rostictl-${VERSION}.windows.i386: preparation
	env GOOS=windows GOARCH=386 go build -o bin/rostictl-${VERSION}.windows.i386
	
bin/rostictl-${VERSION}.windows.amd64: preparation
	env GOOS=windows GOARCH=amd64 go build -o bin/rostictl-${VERSION}.windows.amd64

.PHONY: release
release: bin/rostictl-${VERSION}.linux.arm bin/rostictl-${VERSION}.linux.arm64 bin/rostictl-${VERSION}.linux.i386 bin/rostictl-${VERSION}.linux.amd64 bin/rostictl-${VERSION}.darwin.amd64 bin/rostictl-${VERSION}.windows.i386 bin/rostictl-${VERSION}.windows.amd64
	
.PHONY: clean
clean: preparation
	rm bin/*

.PHONY: build-test-image
build-test-image:
	cd contrib/testimage && docker build -t rostictl_test:${TESTIMAGE_VERSION} .

test: build-test-image
	-docker stop rostictl_test &> /dev/null
	docker run --rm -d -p 3222:22 --name rostictl_test rostictl_test:${TESTIMAGE_VERSION}
	go test src/ssh/*.go
	docker stop rostictl_test
