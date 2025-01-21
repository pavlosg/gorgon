all: control/src.tgz node/src.tgz

control/src.tgz: node/src.tgz
	cp $< $@

node/src.tgz: ../../src
	tar -czf $@ -C ../.. src

build: all
	docker-compose -f docker-compose.yml build

clean:
	rm control/*.tgz node/*.tgz
