obj-m += continfo.o

# Apunta al kernel actualmente en ejecución de la VM
KDIR := /lib/modules/$(shell uname -r)/build
PWD  := $(shell pwd)

all:
	$(MAKE) -C $(KDIR) M=$(PWD) modules

clean:
	$(MAKE) -C $(KDIR) M=$(PWD) clean

reload: uninstall install

.PHONY: all clean install uninstall reload
