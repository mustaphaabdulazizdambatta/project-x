TARGET=x-tymus
PACKAGES=core database log parser

.PHONY: all build clean
all: build

build:
	@go build -o ./build/$(TARGET) -mod=vendor .

clean:
	@go clean
	@rm -f ./build/$(TARGET)
