# Makefile for XenGate

# برنامه و متغیرهای نسخه
APPNAME := xengate
VERSION := 0.0.1
PACKAGE := com.xengate.$(APPNAME)

# مسیرها
BUILDDIR := build
DISTDIR := dist
RESDIR := res

# زمان ساخت
BUILDTIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GOVERSION := $(shell go version | cut -d' ' -f3)

# fyne-cross متغیرها
FYNE_CROSS := $(HOME)/go/bin/fyne-cross
# FYNE_FLAGS := -pull -metadata BuildTime="$(BUILDTIME)" -metadata GoVersion="$(GOVERSION)"

# LDFLAGS := -X \"fyne.io/fyne/v2/app.MetadataCustom=BuildForOS={{.OS}}\" \
#            -X \"fyne.io/fyne/v2/app.MetadataCustom=BuildTime=$(BUILDTIME)\" \
#            -X \"fyne.io/fyne/v2/app.MetadataCustom=GoVersion=$(GOVERSION)\"

FYNE_FLAGS := -pull

# تنظیمات مربوط به نسخه و آیکون
ICON := $(RESDIR)/appicon.png

.PHONY: all clean dist windows linux darwin android ios check-fyne-cross

# همه پلتفرم‌ها
all: check-fyne-cross clean windows linux darwin

# پاک کردن فایل‌های قبلی
clean:
	@echo "Cleaning build directories..."
	@rm -rf $(BUILDDIR) $(DISTDIR)
	@mkdir -p $(BUILDDIR) $(DISTDIR)

# بررسی نصب fyne-cross
check-fyne-cross:
	@if ! command -v $(FYNE_CROSS) > /dev/null; then \
		echo "Installing fyne-cross..."; \
		go install github.com/fyne-io/fyne-cross@latest; \
	fi

# Windows builds
windows: check-fyne-cross
	@echo "Building for Windows..."
	$(FYNE_CROSS) windows \
		-arch=amd64,386 \
		$(FYNE_FLAGS) \
		-app-id=$(PACKAGE) \
		-app-version=$(VERSION) \
		-icon=$(ICON) \
		-output $(APPNAME)

# Linux builds
linux: check-fyne-cross
	@echo "Building for Linux..."
	$(FYNE_CROSS) linux \
		-arch=amd64,386,arm64 \
		$(FYNE_FLAGS) \
		-app-id=$(PACKAGE) \
		-app-version=$(VERSION) \
		-icon=$(ICON) \
		-output $(APPNAME)

# ARM builds
arm: check-fyne-cross
	@echo "Building for ARM..."
	$(FYNE_CROSS) linux \
		-arch=arm \
		$(FYNE_FLAGS) \
		-app-id=$(PACKAGE) \
		-app-version=$(VERSION) \
		-icon=$(ICON) \
		-output $(APPNAME)

# macOS builds
darwin: check-fyne-cross
	@echo "Building for macOS..."
	$(FYNE_CROSS) darwin \
		-arch=amd64,arm64 \
		$(FYNE_FLAGS) \
		-app-id=$(PACKAGE) \
		-app-version=$(VERSION) \
		-icon=$(ICON) \
		-output $(APPNAME)

KEYSTORE := keys/my-release-key.keystore
KEYSTORE_PASSWORD := Ximbesto110
KEY_ALIAS := keyx
KEY_PASSWORD := Xolinna110

# Android build
# android: check-fyne-cross
# 	@echo "Building for Android..."
# 	$(FYNE_CROSS) android \
# 		-app-id=$(PACKAGE) \
# 		-app-version=$(VERSION) \
# 		-icon=$(ICON) \
# 		-keystore=$(KEYSTORE) \
# 		-keystore-pass=$(KEYSTORE_PASSWORD) \
# 		-key-pass=$(KEY_PASSWORD)
android:
	@echo "Building for Android..."
	fyne package -os android \
		-app-id=$(PACKAGE) \
		-name $(APPNAME) \
		-icon $(ICON) \
		-release

# iOS build
ios: check-fyne-cross
	@echo "Building for iOS..."
	$(FYNE_CROSS) ios \
		-app-id=$(PACKAGE) \
		-app-version=$(VERSION) \
		-icon=$(ICON) \
		$(FYNE_FLAGS) \
		-metadata BuildForOS="ios" \
		-output $(APPNAME)

# ساخت فایل‌های توزیع
dist: all
	@echo "Creating distribution packages..."
	@mkdir -p $(DISTDIR)
	@cd $(BUILDDIR) && \
	for f in $(APPNAME)* ; do \
		if [ -f "$$f" ]; then \
			platform=$${f#$(APPNAME)-}; \
			echo "Packaging $$platform..."; \
			zip -r ../$(DISTDIR)/$(APPNAME)-$$platform-$(VERSION).zip $$f; \
		fi \
	done

# ساخت نسخه توسعه
dev:
	@echo "Building development version..."
	go build -v \
		-ldflags "$(LDFLAGS)" \
		-o $(BUILDDIR)/$(APPNAME)

# نصب نسخه توسعه
install: dev
	@echo "Installing development version..."
	cp $(BUILDDIR)/$(APPNAME) $(GOPATH)/bin/$(APPNAME)

# اجرای تست‌ها
test:
	@echo "Running tests..."
	go test -v ./...

# بررسی کد
lint:
	@echo "Running linters..."
	golangci-lint run

# نمایش نسخه
version:
	@echo "$(APPNAME) version $(VERSION)"
	@echo "Build time: $(BUILDTIME)"
	@echo "Go version: $(GOVERSION)"

# راهنما
help:
	@echo "Available targets:"
	@echo "  all      - Build for all platforms"
	@echo "  windows  - Build for Windows (386, amd64)"
	@echo "  linux    - Build for Linux (386, amd64, arm64)"
	@echo "  darwin   - Build for macOS (amd64, arm64)"
	@echo "  android  - Build for Android"
	@echo "  ios      - Build for iOS"
	@echo "  dist     - Create distribution packages"
	@echo "  dev      - Build development version"
	@echo "  install  - Install development version"
	@echo "  clean    - Clean build directories"
	@echo "  test     - Run tests"
	@echo "  lint     - Run linters"
	@echo "  version  - Show version info"