DIRS := $(wildcard */.)
BUILDDIRS = $(DIRS:%=build-%)
PUSHDIRS = $(DIRS:%=push-%)

all: $(BUILDDIRS)

.PHONY: $(BUILDDIRS)
build: $(BUILDDIRS)
$(BUILDDIRS):
	$(MAKE) -C $(@:build-%=%) build

.PHONY: $(PUSHDIRS)
push: $(PUSHDIRS)
$(PUSHDIRS):
	$(MAKE) -C $(@:push-%=%) push
