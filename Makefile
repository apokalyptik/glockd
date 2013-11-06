UNAME := $(shell uname)
GOPATH := $(shell if [ "$(UNAME)" = "Darwin" ]; then greadlink -f ./; else readlink -f ./; fi)/
CMD := env GOPATH=$(GOPATH) go

all: build

build:
	$(CMD) install ./src/github.com/garyburd/go-websocket/websocket/
	$(CMD) build

clean:
	rm -rvf pkg
	rm -vf glockd

run: clean build exec

exec:
	./glockd -pidfile=glockd.pid

env:
	env
