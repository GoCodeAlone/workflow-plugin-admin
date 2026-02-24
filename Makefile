.PHONY: build test install clean ui

BINARY_NAME = workflow-plugin-admin
INSTALL_DIR ?= data/plugins/$(BINARY_NAME)

build:
	go build -o bin/$(BINARY_NAME) ./cmd/$(BINARY_NAME)

test:
	go test ./... -v -race

install: build
	mkdir -p $(DESTDIR)/$(INSTALL_DIR)
	cp bin/$(BINARY_NAME) $(DESTDIR)/$(INSTALL_DIR)/
	cp plugin.json $(DESTDIR)/$(INSTALL_DIR)/

clean:
	rm -rf bin/

ui:
	@echo "Building admin UI from workflow repo..."
	cd /tmp && rm -rf workflow-ui-build && git clone --depth 1 git@github.com:GoCodeAlone/workflow.git workflow-ui-build
	cd /tmp/workflow-ui-build/ui && npm ci && npx vite build
	rm -rf internal/ui_dist && cp -r /tmp/workflow-ui-build/ui/dist internal/ui_dist
	rm -rf /tmp/workflow-ui-build
