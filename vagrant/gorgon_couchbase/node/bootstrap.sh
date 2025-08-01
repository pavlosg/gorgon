apt-get -qy update
apt-get -qy install curl golang-go iptables netcat-openbsd procps python3 sudo tar unzip wget

wget https://packages.couchbase.com/releases/couchbase-release/couchbase-release-1.0-noarch.deb
dpkg -i ./couchbase-release-1.0-noarch.deb
apt-get -qy update
apt-get -qy install couchbase-server
rm *.deb

cp rpc-server.service /etc/systemd/system/rpc-server.service

cd src/gorgon_couchbase
go build

systemctl enable rpc-server
systemctl start rpc-server
