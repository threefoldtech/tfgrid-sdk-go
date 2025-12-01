DIRS := "activation-service" "farmerbot" "grid-cli" "grid-client" "grid-proxy" "gridify" "monitoring-bot" "rmb-sdk-go" "user-contracts-mon" "tfrobot"

# Build configuration
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Detect host OS and architecture
HOST_OS := $(shell uname -s | tr '[:upper:]' '[:lower:]')
HOST_ARCH := $(shell uname -m)

# Normalize architecture names
ifeq ($(HOST_ARCH),x86_64)
	HOST_ARCH := amd64
endif
ifeq ($(HOST_ARCH),aarch64)
	HOST_ARCH := arm64
endif

# Target OS and architecture (can be overridden)
GOOS ?= $(HOST_OS)
GOARCH ?= $(HOST_ARCH)

# Binary names
CLI_BINARY := tfcmd
GUI_BINARY := grid-agent-gui
ifeq ($(GOOS),windows)
	CLI_BINARY := tfcmd.exe
	GUI_BINARY := grid-agent-gui.exe
endif

# Build directories
BUILD_DIR := build
DIST_DIR := dist
CLI_BUILD_DIR := $(BUILD_DIR)/grid-cli
GUI_BUILD_DIR := $(BUILD_DIR)/grid-agent

# Install directories
ifeq ($(GOOS),darwin)
	INSTALL_DIR ?= /usr/local/bin
	GUI_INSTALL_DIR ?= /Applications
else ifeq ($(GOOS),linux)
	INSTALL_DIR ?= $(HOME)/.local/bin
	GUI_INSTALL_DIR ?= $(HOME)/.local/bin
else ifeq ($(GOOS),windows)
	INSTALL_DIR ?= $(USERPROFILE)/bin
	GUI_INSTALL_DIR ?= $(USERPROFILE)/bin
endif

# Go build flags
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildTime=$(BUILD_TIME)"

.PHONY: all build install clean help
.PHONY: build-grid-cli build-grid-agent-gui build-grid-agent-with-grid-cli
.PHONY: install-grid-cli install-grid-agent-gui install-grid-agent-with-grid-cli
.PHONY: build-grid-cli-linux build-grid-cli-darwin build-grid-cli-windows
.PHONY: build-grid-agent-gui-linux build-grid-agent-gui-darwin build-grid-agent-gui-windows
.PHONY: build-grid-agent-with-grid-cli-all-platforms package

# Default target
all: build-grid-agent-with-grid-cli

# ============================================================================
# GRID-CLI Build Targets
# ============================================================================

build-grid-cli:
	@echo "🔨 Building tfcmd for $(GOOS)/$(GOARCH)..."
	@mkdir -p $(CLI_BUILD_DIR)/$(GOOS)-$(GOARCH)
	cd grid-cli && GOOS=$(GOOS) GOARCH=$(GOARCH) go build $(LDFLAGS) -o ../$(CLI_BUILD_DIR)/$(GOOS)-$(GOARCH)/$(CLI_BINARY) .
	@echo "✅ Built: $(CLI_BUILD_DIR)/$(GOOS)-$(GOARCH)/$(CLI_BINARY)"

build-grid-cli-linux:
	@$(MAKE) build-grid-cli GOOS=linux GOARCH=amd64

build-grid-cli-darwin:
	@$(MAKE) build-grid-cli GOOS=darwin GOARCH=amd64

build-grid-cli-darwin-arm64:
	@$(MAKE) build-grid-cli GOOS=darwin GOARCH=arm64

build-grid-cli-windows:
	@$(MAKE) build-grid-cli GOOS=windows GOARCH=amd64

# ============================================================================
# GRID-AGENT-GUI Build Targets
# ============================================================================

build-grid-agent-gui:
	@echo "🔨 Building grid-agent-gui for $(GOOS)/$(GOARCH)..."
	@mkdir -p $(GUI_BUILD_DIR)/$(GOOS)-$(GOARCH)
	@if ! command -v wails >/dev/null 2>&1; then \
		echo "❌ Error: wails is not installed"; \
		echo "Install with: go install github.com/wailsapp/wails/v2/cmd/wails@latest"; \
		exit 1; \
	fi
	cd grid-agent-gui && wails build -tags webkit2_41 -platform $(GOOS)/$(GOARCH) -o ../../../$(GUI_BUILD_DIR)/$(GOOS)-$(GOARCH)/$(GUI_BINARY)
	@echo "✅ Built: $(GUI_BUILD_DIR)/$(GOOS)-$(GOARCH)/$(GUI_BINARY)"

build-grid-agent-gui-linux:
	@$(MAKE) build-grid-agent-gui GOOS=linux GOARCH=amd64

build-grid-agent-gui-darwin:
	@$(MAKE) build-grid-agent-gui GOOS=darwin GOARCH=amd64

build-grid-agent-gui-darwin-arm64:
	@$(MAKE) build-grid-agent-gui GOOS=darwin GOARCH=arm64

build-grid-agent-gui-windows:
	@$(MAKE) build-grid-agent-gui GOOS=windows GOARCH=amd64

# ============================================================================
# Combined Build Targets
# ============================================================================

build-grid-agent-with-grid-cli: build-grid-cli build-grid-agent-gui
	@echo "✅ Build complete for $(GOOS)/$(GOARCH)"

build-grid-agent-with-grid-cli-all-platforms:
	@echo "🔨 Building for all platforms..."
	@$(MAKE) build-grid-cli-linux
	@$(MAKE) build-grid-cli-darwin
	@$(MAKE) build-grid-cli-darwin-arm64
	@$(MAKE) build-grid-cli-windows
	@$(MAKE) build-grid-agent-gui-linux
	@$(MAKE) build-grid-agent-gui-darwin
	@$(MAKE) build-grid-agent-gui-darwin-arm64
	@$(MAKE) build-grid-agent-gui-windows
	@echo "✅ All platforms built successfully"

# ============================================================================
# Install Targets
# ============================================================================

install-grid-cli: build-grid-cli
	@echo "📦 Installing tfcmd to $(INSTALL_DIR)..."
	@mkdir -p $(INSTALL_DIR)
	@cp $(CLI_BUILD_DIR)/$(GOOS)-$(GOARCH)/$(CLI_BINARY) $(INSTALL_DIR)/$(CLI_BINARY)
	@chmod +x $(INSTALL_DIR)/$(CLI_BINARY)
	@echo "✅ Installed: $(INSTALL_DIR)/$(CLI_BINARY)"
	@echo ""
	@echo "Usage: $(CLI_BINARY) --help"

install-grid-agent-gui: build-grid-agent-gui
	@echo "📦 Installing grid-agent-gui to $(GUI_INSTALL_DIR)..."
	@mkdir -p $(GUI_INSTALL_DIR)
ifeq ($(GOOS),darwin)
	@if [ -f "$(GUI_BUILD_DIR)/$(GOOS)-$(GOARCH)/$(GUI_BINARY).app" ]; then \
		cp -r $(GUI_BUILD_DIR)/$(GOOS)-$(GOARCH)/$(GUI_BINARY).app $(GUI_INSTALL_DIR)/; \
		echo "✅ Installed: $(GUI_INSTALL_DIR)/$(GUI_BINARY).app"; \
	else \
		cp $(GUI_BUILD_DIR)/$(GOOS)-$(GOARCH)/$(GUI_BINARY) $(GUI_INSTALL_DIR)/$(GUI_BINARY); \
		chmod +x $(GUI_INSTALL_DIR)/$(GUI_BINARY); \
		echo "✅ Installed: $(GUI_INSTALL_DIR)/$(GUI_BINARY)"; \
	fi
else ifeq ($(GOOS),linux)
	@cp $(GUI_BUILD_DIR)/$(GOOS)-$(GOARCH)/$(GUI_BINARY) $(GUI_INSTALL_DIR)/$(GUI_BINARY)
	@chmod +x $(GUI_INSTALL_DIR)/$(GUI_BINARY)
	@echo "✅ Installed: $(GUI_INSTALL_DIR)/$(GUI_BINARY)"
	@echo "📝 Installing icon..."
	@mkdir -p $(HOME)/.local/share/icons/hicolor/512x512/apps
	@cp grid-agent-gui/build/appicon.png $(HOME)/.local/share/icons/hicolor/512x512/apps/grid-agent-gui.png
	@echo "✅ Icon installed"
	@echo "📝 Creating desktop entry..."
	@mkdir -p $(HOME)/.local/share/applications
	@echo "[Desktop Entry]" > $(HOME)/.local/share/applications/grid-agent-gui.desktop
	@echo "Name=ThreeFold Grid Agent" >> $(HOME)/.local/share/applications/grid-agent-gui.desktop
	@echo "Comment=AI-powered ThreeFold Grid management" >> $(HOME)/.local/share/applications/grid-agent-gui.desktop
	@echo "Exec=env PATH=$(INSTALL_DIR):/usr/local/bin:/usr/bin:/bin $(GUI_INSTALL_DIR)/$(GUI_BINARY)" >> $(HOME)/.local/share/applications/grid-agent-gui.desktop
	@echo "Icon=$(HOME)/.local/share/icons/hicolor/512x512/apps/grid-agent-gui.png" >> $(HOME)/.local/share/applications/grid-agent-gui.desktop
	@echo "Terminal=false" >> $(HOME)/.local/share/applications/grid-agent-gui.desktop
	@echo "Type=Application" >> $(HOME)/.local/share/applications/grid-agent-gui.desktop
	@echo "Categories=Utility;Development;" >> $(HOME)/.local/share/applications/grid-agent-gui.desktop
	@chmod +x $(HOME)/.local/share/applications/grid-agent-gui.desktop
	@echo "✅ Desktop entry created"
else
	@cp $(GUI_BUILD_DIR)/$(GOOS)-$(GOARCH)/$(GUI_BINARY) $(GUI_INSTALL_DIR)/$(GUI_BINARY)
	@echo "✅ Installed: $(GUI_INSTALL_DIR)/$(GUI_BINARY)"
endif
	@echo ""
	@echo "Usage: $(GUI_BINARY)"

install-grid-agent-with-grid-cli: install-grid-cli install-grid-agent-gui
	@echo ""
	@echo "✅ Installation complete!"
	@echo ""
	@echo "Get started:"
	@echo "  $(CLI_BINARY) login"
	@echo "  $(CLI_BINARY) chat"
	@echo "  $(GUI_BINARY)"

# ============================================================================
# Clean Target
# ============================================================================

clean:
	@echo "🧹 Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR) $(DIST_DIR)
	@echo "✅ Clean complete"

# ============================================================================
# Release targets
# ============================================================================

mainnet-release:
	cd grid-client && go get github.com/threefoldtech/tfchain/clients/tfchain-client-go@5d6a2dd
	go work sync
	make tidy

release-rmb:
	@echo "Release RMB..." 
	git tag -a "rmb-sdk-go/${VERSION}" -m "release rmb-sdk-go/${VERSION}" && \
  git push origin rmb-sdk-go/${VERSION}

release:
	@echo "Running release script..." 
	chmod +x release.sh 
	./release.sh

# ============================================================================
# Other targets
# ============================================================================

lint:
	for DIR in ${DIRS} ; do \
		cd $$DIR && golangci-lint run -c ../.golangci.yml --timeout 10m && cd ../ ; \
	done

tidy:
	for DIR in ${DIRS} ; do \
		cd $$DIR && go mod tidy && cd ../ ; \
	done
