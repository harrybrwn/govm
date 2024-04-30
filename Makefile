NAME=$(shell basename $$PWD)
SOURCE=$(shell find . -name '*.go')
COMP=release/completion
DATE=$(shell date +%FT%T%:z)
COMMIT=$(shell git rev-parse HEAD)
VERSION=$(shell git describe --tags 2>/dev/null)
LDFLAGS=-s -w \
	-X github.com/harrybrwn/govm/cmd/govm/cli.version=$(VERSION) \
	-X github.com/harrybrwn/govm/cmd/govm/cli.commit=$(COMMIT) \
	-X github.com/harrybrwn/govm/cmd/govm/cli.built=$(DATE) \
	-X github.com/harrybrwn/govm/cmd/govm/cli.completion=false
GOFLAGS=-trimpath -ldflags "$(LDFLAGS)"
BIN=release/bin/$(NAME)
GEN=release/bin/gen
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
	go build $(GOFLAGS) -o $@ ./cmd/$(NAME)

$(GEN): $(SOURCE)
	go build -o $@ ./cmd/gen

$(COMP)/zsh/_$(NAME): $(COMP)/zsh $(GEN)
	$(GEN) -completion zsh
$(COMP)/bash/$(NAME): $(COMP)/bash $(GEN)
	$(GEN) -completion bash
$(COMP)/fish/$(NAME).fish: $(COMP)/fish $(GEN)
	$(GEN) -completion fish

$(COMP)/bash $(COMP)/zsh $(COMP)/fish:
	mkdir -p $@

release/man: release/bin/gen
	$< -man-dir $@

.PHONY: dist
dist:
	goreleaser release --clean --skip=publish --snapshot
