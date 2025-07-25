FROM ubuntu:24.04

ENV LANG=C.UTF-8

RUN apt-get -qy update
RUN apt-get -qy install curl dos2unix golang-go iptables netcat-openbsd procps python3 sudo tar unzip wget

WORKDIR /root
RUN wget https://packages.couchbase.com/releases/couchbase-release/couchbase-release-1.0-noarch.deb
RUN dpkg -i ./couchbase-release-1.0-noarch.deb
RUN apt-get -qy update
RUN apt-get -qy install couchbase-server-community
RUN rm *.deb

COPY init.sh /init.sh
RUN dos2unix /init.sh
COPY run.sh /run.sh
RUN dos2unix /run.sh

ADD src.tgz /
WORKDIR /src/gorgon_couchbase
RUN go build

WORKDIR /root
ENTRYPOINT ["sh", "/init.sh"]
