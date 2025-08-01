set -o errexit
set -o pipefail
set -o nounset

cd /home/vagrant

until nc -q 1 localhost 8091 < /dev/null ; do sleep 1 ; done

src/gorgon_couchbase/gorgon_couchbase rpc
