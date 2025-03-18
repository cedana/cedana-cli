BINARY_NAME = cedana-cli
INSTALL_PATH = /usr/local/bin
SUDO=sudo

.PHONY: all build install clean

all: build install

build:
	go build -o $(BINARY_NAME)

install: build
	$(SUDO) install $(BINARY_NAME) $(INSTALL_PATH)

clean:
	$(SUDO) rm -f $(INSTALL_PATH)/$(BINARY_NAME)
	$(SUDO) rm -f $(BINARY_NAME)

