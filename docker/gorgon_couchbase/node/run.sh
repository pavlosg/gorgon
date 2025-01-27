set -o errexit
set -o pipefail
set -o nounset

export PATH=/opt/couchbase/bin:$PATH

wait_for_node() {
    until nc -q 1 "$1" "$2" < /dev/null ; do sleep 1 ; done
}

couchbase-server --start

wait_for_node localhost 8091

sleep 1
couchbase-cli node-init \
    -c localhost \
    -u Administrator \
    -p password \
    --node-init-hostname $(hostname)
sleep 1

if [ "$(hostname)" = "n0.local" ] ; then
    couchbase-cli cluster-init \
        -c localhost \
        --cluster-username Administrator \
        --cluster-password password \
        --services data
    sleep 20
    curl -u Administrator:password -X POST \
        'http://localhost:8091/controller/rebalance' \
        -d 'knownNodes=ns_1@n0.local,ns_1@n1.local,ns_1@n2.local' 2> /dev/null
    echo
    sleep 10
    couchbase-cli bucket-create \
        -c localhost \
        -u Administrator \
        -p password \
        --bucket default \
        --bucket-type couchbase \
        --bucket-ramsize 1024 \
        --bucket-eviction-policy fullEviction \
        --bucket-replica 2 \
        --enable-flush 1
else
    sleep 5
    curl -u Administrator:password -X POST \
        "http://$(hostname):8091/node/controller/doJoinCluster" \
        -d 'hostname=n0.local&user=Administrator&password=password&services=kv' 2> /dev/null
    echo
fi

while true ; do
    /src/gorgon_couchbase/gorgon_couchbase rpc
    echo Restarting RPC
    sleep 1
done
