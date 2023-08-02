build/watchit: main.go go.mod go.sum
	CGO_ENABLED=0 go build -o build/watchit

.PHONY: install
install:
	install --compare --mode 0755 build/watchit /usr/local/bin/watchit
