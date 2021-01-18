.PHONY: all
all: release

.PHONY: preparation
preparation:
	mkdir -p bin

bin/rostictl.linux.arm: preparation
	env GOOS=linux GOARCH=arm go build -o bin/rostictl.linux.arm

bin/rostictl.linux.arm64: preparation
	env GOOS=linux GOARCH=arm64 go build -o bin/rostictl.linux.arm64
	
bin/rostictl.linux.386: preparation
	env GOOS=linux GOARCH=386 go build -o bin/rostictl.linux.386
	
bin/rostictl.linux.amd64: preparation
	env GOOS=linux GOARCH=amd64 go build -o bin/rostictl.linux.amd64

bin/rostictl.darwin.amd64: preparation
	env GOOS=darwin GOARCH=amd64 go build -o bin/rostictl.darwin.amd64
	
bin/rostictl.darwin.arm64: preparation
	env GOOS=darwin GOARCH=arm64 go build -o bin/rostictl.darwin.arm64

bin/rostictl.windows.x86: preparation
	env GOOS=windows GOARCH=386 go build -o bin/rostictl.windows.x86
	
bin/rostictl.windows.amd64: preparation
	env GOOS=windows GOARCH=amd64 go build -o bin/rostictl.windows.amd64

.PHONY: release
release: bin/rostictl.linux.arm bin/rostictl.linux.arm64 bin/rostictl.linux.386 bin/rostictl.linux.amd64 bin/rostictl.darwin.amd64 bin/rostictl.darwin.arm64 bin/rostictl.windows.x86 bin/rostictl.windows.amd64
	
