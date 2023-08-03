BIN_NAME := watchit
BUILD_DIR := build
BUILD_BIN := ${BUILD_DIR}/${BIN_NAME}
INStALL_DIR := /usr/local/bin
INSTALL_BIN := ${INSTALL_DIR}/${BIN_NAME}

${BUILD_BIN}: main.go go.mod go.sum
	CGO_ENABLED=0 go build -o ${BUILD_BIN}

.PHONY: clean
clean:
	rm ${BUILD_BIN}
	rmdir ${BUILD_DIR}

.PHONY: install
install:
	install --compare --mode 0755 ${BUILD_BIN} ${INSTALL_BIN}

.PHONY: uninstall
uninstall:
	test -f ${INSTALL_BIN} && rm ${INSTALL_BIN}
