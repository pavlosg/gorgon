apt-get -qy update
apt-get -qy install curl golang-go netcat-openbsd python3 sudo tar unzip wget

cd src/gorgon_couchbase
go build
