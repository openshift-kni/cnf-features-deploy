GOPATH  ?= $(HOME)/go
GOBIN ?= $(GOPATH)/bin
export MCMAKER := $(GOBIN)/mcmaker
export GOFLAGS ?= -mod=vendor

DST := ../source-crs
CHK := .chk

GEN_DIRS := $(dir $(wildcard */build.sh))
GEN_TGTS := $(patsubst %/,$(DST)/%.yaml,$(GEN_DIRS))
CHK_TGTS := $(patsubst %/,$(CHK)/%.diff,$(GEN_DIRS))
TESTS := $(wildcard */test.sh)

.PHONY: all clean force cleanchk check test

all: test $(GEN_TGTS)

clean: cleanchk
	rm -rf $(GEN_TGTS)

force: clean all

cleanchk:
	rm -rf $(CHK)

check: cleanchk $(CHK_TGTS) test

$(DST)/%.yaml: %/build.sh %/*
	@$(MAKE) $(MCMAKER)
	(cd $(<D); ./$(<F)) >$@

$(CHK)/%.yaml: %/build.sh %/*
	@$(MAKE) $(MCMAKER)
	@[[ -d $(@D) ]] || mkdir -p $(@D)
	@(cd $(<D); ./$(<F)) >$@

$(CHK)/%.diff: $(CHK)/%.yaml
	@echo -n "Checking in-source $(DST)/$(<F) for consistency ... "; \
	if diff -u $(DST)/$(<F) $< >$@; then \
	  echo "Passed"; \
	else \
	  echo "Failed"; \
	  sed 's/^/  /' $@; \
	  exit 1; \
	fi

test:
	@failures=0; for t in $(TESTS); do \
	  testdir=$$(dirname $$t); \
	  echo -n "Testing $$t ... "; \
	  pushd $$testdir >/dev/null; \
	  logfile=/tmp/$$$$-$$testdir.out; \
	  if ./test.sh &>$$logfile; then \
	    echo "Passed"; \
	  else \
	    echo "Failed"; \
	    (( failures += 1 )); \
	    sed 's/^/  /' $$logfile; \
	  fi; \
	  rm -f $$logfile; \
	  popd >/dev/null; \
	done; \
	exit $$failures

$(MCMAKER):
	go install github.com/lack/mcmaker
