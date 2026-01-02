DOGGER ?= $(LOCALBIN)/dogger

.PHONY: dogger
dogger: $(DOGGER)
$(DOGGER): $(LOCALBIN)
	@dagger call binary export --path $(DOGGER)

LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	@mkdir -p $(LOCALBIN)

BIN ?= /usr/local/bin
INSTALL ?= sudo install

.PHONY: install
install: dogger
	@$(INSTALL) $(DOGGER) $(BIN)
