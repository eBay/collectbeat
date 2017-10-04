BEAT_NAME=collectbeat
GOPACKAGES?=$(shell go list ./... | grep -v /vendor/)
ES_BEATS?=./_beats

# Path to the libbeat Makefile
-include $(ES_BEATS)/libbeat/scripts/Makefile

# Collects all dependencies and then calls update
.PHONY: collect
collect: notice
update-beats:
	@BEATS_VERSION=$(BEATS_VERSION) sh script/update_beats.sh
	@$(MAKE) update

check-full: check
	@# Validate that all updates were committed
	@$(MAKE) update
	@$(MAKE) check
	@git diff | cat
	@git update-index --refresh
	@git diff-index --exit-code HEAD --

clean-container:
	rm -rf build

container:
	@$(MAKE) clean-container
	mkdir -p build
	GOOS=linux go build -o build/collectbeat -i .
	docker build .

post-update:

.PHONY: notice
notice: python-env
	@echo "Generating NOTICE"
	@$(PYTHON_ENV)/bin/python script/generate_notice.py . -e '_beats' -s "./vendor/github.com/elastic/beats" -c "eBay Inc"  -b "Collectbeat"
