FROM ubuntu:24.04

ENV LANG=C.UTF-8

RUN apt-get -qy update
RUN apt-get -qy install curl golang-go netcat-openbsd python3 sudo tar unzip wget

ADD src.tgz /
WORKDIR /src/gorgon_couchbase
RUN go build

WORKDIR /root
COPY init.sh /init.sh
COPY run.sh /run.sh

ENTRYPOINT ["sh", "/init.sh"]
