# Makefile

binary_name=mylb

build:
	@echo "Building..."
	go build -o $(binary_name) cmd/main.go

install: build
	@echo "Installing to /usr/local/bin..."
	sudo mv $(binary_name) /usr/local/bin/

clean:
	@echo "Cleaning..."
	rm -f $(binary_name)

run:
	go run cmd/main.go