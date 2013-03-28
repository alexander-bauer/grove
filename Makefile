program_NAME := grove
GOCOMPILER := go build
GOFLAGS	+= -ldflags "-X main.Version $(shell git describe)"


.PHONY: all install clean disclean

all: $(program_NAME)

$(program_NAME):
	$(GOCOMPILER) $(GOFLAGS)

clean:
	@- $(RM) $(program_NAME)
