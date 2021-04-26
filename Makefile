ROOT_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
build:
	go build .

install:
	rm -f /usr/local/bin/tf-measure-refresh
	cd /usr/local/bin/ && sudo ln -s $(ROOT_DIR)/tf-measure-refresh tf-measure-refresh

.PHONY: build
