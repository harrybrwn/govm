NAME=$(shell basename $$PWD)
SOURCE=$(shell find . -name '*.go')
COMP=release/completion
DATE=$(shell date +%FT%T%:z)
COMMIT=$(shell git rev-parse HEAD)
VERSION=$(shell git describe --tags)
LDFLAGS=-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.built=$(DATE)
GOFLAGS=-trimpath -ldflags "$(LDFLAGS)"
BIN=release/bin/$(NAME)

build: $(BIN) completion

clean:
	$(RM) -r release dist result

completion: $(COMP)/bash/$(NAME) $(COMP)/zsh/_$(NAME) $(COMP)/fish/$(NAME).fish

install: $(BIN)
	sudo install -m 4755 -o root $(BIN) $$GOPATH/bin

uninstall:
	sudo $(RM) $$GOPATH/bin/$(NAME)

install-to: $(BIN) completion
	@if [ -z $(PREFIX) ]; then echo 'Error: no install prefix. Use "make install PREFIX=/path/to/root"'; echo; exit 1; fi
	cp $(BIN) $(PREFIX)/usr/bin
	cp $(COMP)/bash/$(NAME)

.PHONY: build clean completion install install-to

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

.PHONY: dist
dist:
	goreleaser release --rm-dist --skip-publish --snapshot

