ifndef CFLAGS
CFLAGS		= -O2 -Wall
endif

INCLUDE		+= -I.
LDFLAGS         += -lpthread -lnuma -lm

is_ppc		:= $(shell (uname -m || uname -p) | grep ppc)
is_x86		:= $(shell (uname -m || uname -p) | grep i.86)
is_x86_64	:= $(shell (uname -m || uname -p) | grep x86_64)

ifneq ($(is_x86),)
# Need to tell gcc we have a reasonably recent cpu to get the atomics.
CFLAGS += -march=i686
endif

IMAGE_BUILD_CMD ?= "docker"
IMAGE_REGISTRY ?= "quay.io"
REGISTRY_NAMESPACE ?= ""
IMAGE_TAG ?= "latest"
FULL_IMAGE_NAME ?= "${IMAGE_REGISTRY}/${REGISTRY_NAMESPACE}/oslat:${IMAGE_TAG}"

all: oslat

oslat: main.o rt-utils.o error.o trace.o
	$(CC) -o $@ $(LDFLAGS) $^

clean:
	rm -f *.o oslat cscope.*

install: oslat
	sudo install oslat /usr/local/bin

cscope:
	cscope -bq *.c

build-container:
	@if [ -z "$(REGISTRY_NAMESPACE)" ]; then\
		echo "REGISTRY_NAMESPACE env-var must be set to your $(IMAGE_REGISTRY) repository";\
		exit 1;\
	fi
	$(IMAGE_BUILD_CMD) build -t $(FULL_IMAGE_NAME) .

push-container:
	@if [ -z "$(REGISTRY_NAMESPACE)" ]; then\
                echo "REGISTRY_NAMESPACE env-var must be set to your $(IMAGE_REGISTRY) repository";\
                exit 1;\
        fi
	$(IMAGE_BUILD_CMD) push $(FULL_IMAGE_NAME)
