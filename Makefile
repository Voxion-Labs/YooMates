# YooMates Automation Makefile
# Engine: Go + Git Rebase-safe Workflow + voxion-bot manifest

SHELL := cmd.exe
.DEFAULT_GOAL := help

# Variables
BRANCH := $(shell git rev-parse --abbrev-ref HEAD 2>NUL || echo main)
BIN_DIR := bin
SERVER_BIN := $(BIN_DIR)/server.exe

.PHONY: help pull push sync manifest build wasm audio-engine test-audio run test clean status

help:
	@echo ===============================================================
	@echo   YooMates Project Automation (Voxion Labs)
	@echo ===============================================================
	@echo   make pull         - Pull latest changes with rebase (git pull --rebase)
	@echo   make sync         - Alias for make pull
	@echo   make push         - Safe sync: rebase pull, update manifest, then push
	@echo   make manifest     - Run voxion-bot commit manifest generator
	@echo   make build        - Compile Go server binary to bin/
	@echo   make wasm         - Compile WebAssembly client search module
	@echo   make audio-engine - Compile C++ low-latency audio processing engine
	@echo   make test-audio   - Run C++ audio engine verification suite
	@echo   make run          - Run local Go server (cmd/server/main.go)
	@echo   make test         - Execute Go unit tests
	@echo   make status       - View current git branch and uncommitted changes
	@echo   make clean        - Remove build artifacts from bin/
	@echo ===============================================================

pull:
	@echo [voxion-bot] Pulling with rebase from origin/$(BRANCH)...
	git pull --rebase origin $(BRANCH)

sync: pull

manifest:
	@echo [voxion-bot] Generating commit manifest...
	go run scripts/manifest.go

push:
	@echo [voxion-bot] Step 1/3: Synchronizing with rebase...
	git pull --rebase origin $(BRANCH)
	@echo [voxion-bot] Step 2/3: Refreshing voxion-bot manifest...
	go run scripts/manifest.go
	-git add voxion-manifest.json
	-git commit -m "chore(manifest): refresh voxion-manifest.json [skip ci]" 2>NUL || ver >NUL
	@echo [voxion-bot] Step 3/3: Pushing to origin/$(BRANCH)...
	git push origin HEAD:$(BRANCH)
	@echo [voxion-bot] Push completed successfully.

build:
	@echo [build] Compiling YooMates Go server binary...
	@if not exist $(BIN_DIR) mkdir $(BIN_DIR)
	go build -v -o $(SERVER_BIN) ./cmd/server/main.go

wasm:
	@echo [wasm] Compiling Go WebAssembly search module...
	@if not exist web\static\wasm mkdir web\static\wasm
	set GOOS=js&& set GOARCH=wasm&& go build -trimpath -ldflags="-s -w" -o web/static/wasm/search.wasm ./cmd/wasm/main.go

audio-engine:
	@echo [c++] Compiling C++ Audio Packet Engine...
	@if not exist $(BIN_DIR) mkdir $(BIN_DIR)
	g++ -std=c++14 -O3 -Wall -Iengine/audio/include engine/audio/src/*.cpp -o $(BIN_DIR)/yoomates-voice.exe

test-audio: audio-engine
	@echo [test] Executing C++ audio test suite...
	$(BIN_DIR)\yoomates-voice.exe --test

run:
	@echo [run] Launching YooMates Go server...
	go run ./cmd/server/main.go

test:
	@echo [test] Running test suite...
	go test -v ./...

status:
	@git status -s

clean:
	@if exist $(BIN_DIR) rmdir /s /q $(BIN_DIR)
	@echo [clean] Build artifacts cleaned.
