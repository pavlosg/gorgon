set -o errexit
set -o pipefail
set -o nounset

export PATH=/home/vagrant/src/gorgon_couchbase:$PATH

wait_for_node() {
    until nc -q 1 "$1" "$2" < /dev/null ; do sleep 1 ; done
}

nodes="$(cat nodes.txt)"

for node in $(tr ',' ' ' < nodes.txt); do
    wait_for_node "$node" 9090
done

{
    echo No durability
    gorgon_couchbase run --nodes "$nodes"

    echo majority_and_persist_on_master
    gorgon_couchbase run \
        --nodes "$nodes" \
        --extras 'db_durability=majority_and_persist_on_master' \
        -E '*~nil'
} 2>&1 | tee gorgon.log

touch .html
tar -czf files.tgz gorgon.log *.html

echo
echo DONE
