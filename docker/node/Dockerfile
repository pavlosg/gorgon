FROM ubuntu:24.04

ENV LANG C.UTF-8

RUN apt-get -y update
RUN apt-get -qy install curl dos2unix go iptables python3 sudo tar unzip vim wget

RUN groupadd gorgon &&\
    useradd -m gorgon -g gorgon -s /bin/bash &&\
    echo "gorgon:123456" | /usr/sbin/chpasswd && chage -M 99999 gorgon &&\
    echo "gorgon ALL=(ALL) NOPASSWD:ALL" >>/etc/sudoers &&\
    mkdir -p /home/gorgon/.ssh

ADD ./init.sh /init.sh
RUN dos2unix /init.sh

CMD ["bash" "/init.sh"]
