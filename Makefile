NAME=$(shell basename $$PWD)
SOURCE=$(shell find . -name '*.go')
COMP=release/completion
DATE=$(shell date +%FT%T%:z)
COMMIT=$(shell git rev-parse HEAD)
VERSION=$(shell git describe --tags)
LDFLAGS=-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.built=$(DATE)
GOFLAGS=-trimpath -ldflags "$(LDFLAGS)"
BIN=release/bin/$(NAME)
ifeq ($(VERSION),)
VERSION=v0.0.0
endif

build: $(BIN) completion man

clean:
	$(RM) -r release dist result

completion: $(COMP)/bash/$(NAME) $(COMP)/zsh/_$(NAME) $(COMP)/fish/$(NAME).fish
man: release/man

install: $(BIN)
	sudo install -m 4755 -o root $(BIN) $$GOPATH/bin

uninstall:
	sudo $(RM) $$GOPATH/bin/$(NAME)

install-to: $(BIN) completion
	@if [ -z $(PREFIX) ]; then echo 'Error: no install prefix. Use "make install PREFIX=/path/to/root"'; echo; exit 1; fi
	cp $(BIN) $(PREFIX)/usr/bin
	cp $(COMP)/bash/$(NAME)

lint:
	golangci-lint run --disable unused

.PHONY: build clean completion man install install-to lint

$(BIN): $(SOURCE)
	go build $(GOFLAGS) -o $@

$(COMP)/zsh/_$(NAME): $(COMP)/zsh $(BIN)
	$(BIN) completion zsh > $@

$(COMP)/bash/$(NAME): $(COMP)/bash $(BIN)
	$(BIN) completion bash > $@

$(COMP)/fish/$(NAME).fish: $(COMP)/fish $(BIN)
	$(BIN) completion fish > $@

$(COMP)/bash $(COMP)/zsh $(COMP)/fish:
	mkdir -p $@

release/bin/_tmp: $(SOURCE)
	go build -ldflags "$(LDFLAGS) -X main.docCmd=true" -o $@

release/man: release/bin/_tmp
	$< doc --man-dir $@

.PHONY: dist
dist:
	goreleaser release --clean --skip=publish --snapshot
