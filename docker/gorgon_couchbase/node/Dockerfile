FROM ubuntu:24.04

ENV LANG=C.UTF-8

RUN apt-get -qy update
RUN apt-get -qy install curl golang-go iptables netcat-openbsd procps python3 sudo tar unzip wget

WORKDIR /root
RUN wget https://packages.couchbase.com/releases/couchbase-release/couchbase-release-1.0-noarch.deb
RUN dpkg -i ./couchbase-release-1.0-noarch.deb
RUN apt-get -qy update
RUN apt-get -qy install couchbase-server
RUN rm *.deb

RUN mkdir /root/deps
ADD deps.tgz /root/deps
WORKDIR /root/deps/src/gorgon_couchbase
RUN go mod download

ADD src.tgz /
WORKDIR /src/gorgon_couchbase
RUN go build

WORKDIR /root
COPY init.sh run.sh /

ENTRYPOINT ["sh", "/init.sh"]
