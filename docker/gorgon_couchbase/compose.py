import sys

node_count = int(sys.argv[1])

with open('compose.control.yaml', 'r') as f:
    compose = f.read()

with open('compose.node.yaml', 'r') as f:
    f.readline()
    node_template = f.read()

compose = compose.replace('NODE_LIST', ','.join(f'n{i}.local' for i in range(node_count)))
for i in range(node_count):
    node = node_template.replace('NODE_IDX', str(i)).replace('NODE_FWD_PORT', str(8090 + i))
    if i == 0:
        node += '    build: ./node\n'
    compose += node

with open('compose.yaml', 'w') as f:
    f.write(compose)
