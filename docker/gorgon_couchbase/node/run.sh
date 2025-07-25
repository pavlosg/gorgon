set -o errexit
set -o pipefail
set -o nounset

export PATH=/opt/couchbase/bin:$PATH

wait_for_node() {
    until nc -q 1 "$1" "$2" < /dev/null ; do sleep 1 ; done
}

couchbase-server --start

wait_for_node localhost 8091

while true ; do
    /src/gorgon_couchbase/gorgon_couchbase rpc
    echo Restarting RPC
    sleep 1
done
