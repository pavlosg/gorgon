all: control/src.tgz node/src.tgz compose.yaml

control/src.tgz: node/src.tgz
	cp $< $@

node/src.tgz: ../../src
	tar -czf $@ -C ../.. src

compose.yaml: compose.control.yaml compose.node.yaml
	python3 compose.py 3

build: all
	docker-compose -f compose.yaml build

clean:
	rm */src.tgz
