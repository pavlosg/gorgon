archive: control/src.tgz node/src.tgz

control/src.tgz: node/src.tgz
	cp $< $@

node/src.tgz: ../../src
	tar -czf $@ -C ../.. src

build: archive
	docker-compose -f docker-compose.yml build

clean:
	rm */src.tgz
