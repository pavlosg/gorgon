FROM ubuntu:24.04

ENV LANG=C.UTF-8

RUN apt-get -qy update
RUN apt-get -qy install curl dos2unix golang-go netcat-openbsd python3 sudo tar unzip wget

COPY init.sh /init.sh
RUN dos2unix /init.sh

WORKDIR /src/gorgon_couchbase
ADD src.tgz /
RUN go build

WORKDIR /root
COPY run.sh /run.sh
RUN dos2unix /run.sh

ENTRYPOINT ["sh", "/init.sh"]
