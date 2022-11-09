GOFLAGS=-trimpath -ldflags "-s -w"
NAME=$(shell basename $$PWD)
SOURCE=$(shell find . -name '*.go')
COMP=release/completion
BIN=release/bin/$(NAME)

build: $(BIN) completion

clean:
	$(RM) -r release dist

completion: $(COMP)/bash/$(NAME) $(COMP)/zsh/_$(NAME) $(COMP)/fish/$(NAME).fish

install: uninstall $(BIN)
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

