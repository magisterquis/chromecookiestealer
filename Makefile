# Makefile
# Build chromecookiestealer
# By J. Stuart McMurray
# Created 20230415
# Last Modified 20230519

.PHONY: all build check clean

BIN=chromecookiestealer

all: check build

build:
	go build -v -trimpath -ldflags "-w -s" -o $(BIN)

check:
	go test
	go vet
	staticcheck

clean:
	rm -f $(BIN)
