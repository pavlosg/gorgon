FROM ubuntu:24.04

ENV LANG C.UTF-8

RUN apt-get -qy update
RUN apt-get -qy install curl dos2unix golang-go python3 sudo tar unzip vim wget

COPY init.sh /init.sh
RUN dos2unix /init.sh

ADD src.tgz /
WORKDIR /src/gorgon_couchbase
RUN go build

ENTRYPOINT ["bash", "/init.sh"]
