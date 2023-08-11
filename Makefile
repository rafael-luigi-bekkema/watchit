BIN_NAME := watchit
BUILD_DIR := build
BUILD_BIN := ${BUILD_DIR}/${BIN_NAME}

${BUILD_BIN}: main.go go.mod go.sum
	CGO_ENABLED=0 go build -o ${BUILD_BIN}

.PHONY: clean
clean:
	rm ${BUILD_BIN}
	rmdir ${BUILD_DIR}

.PHONY: install
install:
	go install .

.PHONY: uninstall
uninstall:
	test -f ${INSTALL_BIN} && rm ${INSTALL_BIN}
