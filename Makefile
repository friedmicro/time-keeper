GO    ?= go
NPM   ?= npm
CMAKE ?= cmake

BIN := bin

.PHONY: all api state timewaybar gtk distractions frontend cellnotify clean

all: api state timewaybar gtk distractions frontend cellnotify

$(BIN):
	mkdir -p $(BIN)

api: $(BIN)
	cd api && $(GO) build -o ../$(BIN)/timekeeper-api .

state: $(BIN)
	cd state && $(GO) build -trimpath -ldflags="-s -w" -o ../$(BIN)/timestate .

timewaybar: $(BIN)
	cd timewaybar && $(GO) build -o ../$(BIN)/timewaybar .

gtk: $(BIN)
	$(CMAKE) -S . -B build-gtk
	$(CMAKE) --build build-gtk
	cp build-gtk/timekeeper-gtk $(BIN)/

distractions: $(BIN)
	$(MAKE) -C notificationhandler
	cp notificationhandler/distractions $(BIN)/

cellnotify: $(BIN)
	cd cellnotify && $(GO) build -o ../$(BIN)/cellnotify .

frontend:
	cd frontend && $(NPM) ci --include=dev && $(NPM) run build

clean:
	rm -rf $(BIN) build-gtk
	$(MAKE) -C notificationhandler clean
	rm -rf frontend/dist frontend/node_modules
