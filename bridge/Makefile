
MAJOR?=0
MINOR?=1
 
VERSION=$(MAJOR).$(MINOR)

BIN := bridge

SRCS = $(wildcard *.c)

OBJDIR := obj

DEPDIR := $(OBJDIR)/.deps

# object files, auto generated from source files
OBJS := $(patsubst %,$(OBJDIR)/%.o,$(basename $(SRCS)))
# dependency files, auto generated from source files
DEPS := $(patsubst %,$(DEPDIR)/%.d,$(basename $(SRCS)))

# compilers (at least gcc and clang) don't create the subdirectories automatically
$(shell mkdir -p $(dir $(OBJS)) >/dev/null)
$(shell mkdir -p $(dir $(DEPS)) >/dev/null)

CC=gcc
CFLAGS=-Wall -O3
LDLIBS=-lqpid-proton -lpthread
LDFLAGS=

DEPFLAGS = -MT $@ -MD -MP -MF $(DEPDIR)/$*.Td

# compile C source files
COMPILE.c = $(CC) $(DEPFLAGS) $(CFLAGS) $(CPPFLAGS) -c -o $@

# link object files to binary
LINK.o = $(LD) $(LDFLAGS) $(LDLIBS) -o $@

# precompile step
PRECOMPILE =
# postcompile step
POSTCOMPILE = mv -f $(DEPDIR)/$*.Td $(DEPDIR)/$*.d

HUB_NAMESPACE = "localhost"
BUILDER_IMAGE_NAME = "sgbridge-builder"
BRIDGE_IMAGE_NAME = "sgbridge"

all: $(BIN)
debug: CFLAGS=-Wall -g
debug: all

.PHONY: clean
clean:
	rm -fr $(OBJDIR) $(DEPDIR)

.PHONY: clean-images
clean-images: version-check
	@echo "+ $@"
	@podman rmi ${HUB_NAMESPACE}/${BUILDER_IMAGE_NAME}:latest  || true
	@podman rmi ${HUB_NAMESPACE}/${BRIDGE_IMAGE_NAME}:latest  || true

.PHONY: builder-image
builder-image: version-check
	@echo "+ $@"
	@buildah bud -t ${HUB_NAMESPACE}/${BUILDER_IMAGE_NAME}:latest -f build/Dockerfile.builder
	@echo 'Done.'

.PHONY: bridge-image
bridge-image: version-check
	@echo "+ $@"
	@buildah bud --build-arg=BUILDER_IMAGE_NAME=${BUILDER_IMAGE_NAME} -t ${HUB_NAMESPACE}/${BRIDGE_IMAGE_NAME}:latest -f build/Dockerfile.sgbridge
	@echo 'Done.'

$(BIN): $(OBJS)
	$(CC) -o $@ $^ $(LDFLAGS) $(LDLIBS)

$(OBJDIR)/%.o: %.c
$(OBJDIR)/%.o: %.c $(DEPDIR)/%.d
	$(PRECOMPILE)
	$(COMPILE.c) $<
	$(POSTCOMPILE)

$(OBJDIR)/%.o : %.c $(DEPDIR)/%.d | $(DEPDIR)
	$(COMPILE.c) $(OUTPUT_OPTION) $<

.PRECIOUS: $(DEPDIR)/%.d
$(DEPDIR)/%.d: ;

#################################
# Utilities
#################################

.PHONY: version-check
version-check:
	@echo "+ $@"
    ifdef VERSION
		@echo "VERSION is ${VERSION}"
    else
		@echo "VERSION is not set!"
		@false;
    endif

-include $(DEPS)
