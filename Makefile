PROGRAM_NAME := grove
GOCOMPILER := go build
GOFLAGS	+= -ldflags "-X main.Version $(shell git describe --dirty=+)"


.PHONY: all install clean

all: $(PROGRAM_NAME)

$(PROGRAM_NAME): $(wildcard *.go)
	$(GOCOMPILER) $(GOFLAGS)

install: $(PROGRAM_NAME)
	cp -rf res/ /usr/local/share/$(PROGRAM_NAME)
	install -m 755 $(PROGRAM_NAME) /usr/local/bin

clean:
	@- $(RM) $(PROGRAM_NAME)
