CLANG ?= clang
STRIP ?= llvm-strip

BPF_TARGET_ARCH ?= x86
BPF_CFLAGS := -O2 -g -target bpf -D__TARGET_ARCH_$(BPF_TARGET_ARCH)
BPF_SOURCES := $(wildcard *.bpf.c)
BPF_OBJECTS := $(BPF_SOURCES:.bpf.c=.bpf.o)

.PHONY: all clean

all: $(BPF_OBJECTS)

%.bpf.o: %.bpf.c
	$(CLANG) $(BPF_CFLAGS) -c $< -o $@

clean:
	rm -f $(BPF_OBJECTS)
