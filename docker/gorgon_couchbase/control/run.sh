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

nodes='n0.local,n1.local,n2.local'

{
    echo No durability
    gorgon_couchbase run --nodes $nodes

    echo majority_and_persist_on_master
    gorgon_couchbase run \
        --nodes $nodes \
        --extras 'db_durability=majority_and_persist_on_master' \
        -E '*~nil'
} 2>&1 | tee gorgon.log

touch .html
tar -czf files.tgz gorgon.log *.html

echo
echo DONE
