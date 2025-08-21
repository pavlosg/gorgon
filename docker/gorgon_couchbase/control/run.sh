set -o errexit
set -o pipefail
set -o nounset

export PATH=/src/gorgon_couchbase:$PATH

wait_for_node() {
    until nc -q 1 "$1" "$2" < /dev/null ; do sleep 1 ; done
}

wait_for_node n0.local 9090
wait_for_node n1.local 9090
wait_for_node n2.local 9090

NODES=${NODES:-'n0.local,n1.local,n2.local'}

{
    echo No durability
    gorgon_couchbase -gorgon-nodes $NODES -gorgon-concurrency 8 run

    echo majorityPersistActive
    gorgon_couchbase \
        -gorgon-nodes $NODES \
        -gorgon-match '*~*~*' \
        -gorgon-concurrency 10 \
        -durability majorityPersistActive \
        -replicas 2 \
        run

    echo majorityPersistActive client-over-rpc
    gorgon_couchbase \
        -gorgon-nodes $NODES \
        -gorgon-match '*~*~*' \
        -gorgon-concurrency 18 \
        -durability majorityPersistActive \
        -replicas 2 \
        -client-over-rpc \
        run
} 2>&1 | tee gorgon.log

touch .html
tar -czf files.tgz gorgon.log *.html

echo
echo DONE
