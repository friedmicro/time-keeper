CC      ?= cc
PKGS    := libpipewire-0.3 libcurl jansson
CFLAGS  += -O2 -Wall -Wextra $(shell pkg-config --cflags $(PKGS))
LDLIBS  += $(shell pkg-config --libs $(PKGS))

BIN := distractions

all: $(BIN)

$(BIN): main.c
	$(CC) $(CFLAGS) -o $@ $< $(LDLIBS)

install: $(BIN)
	install -Dm755 $(BIN) $(DESTDIR)/usr/local/bin/$(BIN)

clean:
	rm -f $(BIN)

.PHONY: all install clean
