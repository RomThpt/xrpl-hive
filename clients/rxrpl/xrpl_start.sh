#!/bin/bash
set -e

CONFIG=/etc/rxrpl/node.toml

# Build TOML config from XRPL_* env vars.
# rxrpl uses a different TOML schema than goXRPL.
cat > $CONFIG <<CFGEOF
[server]
bind = "0.0.0.0:${XRPL_RPC_PORT:-5005}"
admin_ips = ["0.0.0.0"]

[peer]
port = ${XRPL_PEER_PORT:-51235}
max_peers = ${XRPL_PEERS_MAX:-21}
tls_enabled = false
CFGEOF

# Peer discovery.
if [ -n "$XRPL_BOOTNODE" ]; then
    echo 'fixed_peers = [' >> $CONFIG
    IFS=',' read -ra PEERS <<< "$XRPL_BOOTNODE"
    for peer in "${PEERS[@]}"; do
        echo "    \"$peer\"," >> $CONFIG
    done
    echo ']' >> $CONFIG
else
    echo 'fixed_peers = []' >> $CONFIG
fi

# Validator seed as node_seed.
if [ -n "$XRPL_VALIDATOR_SEED" ]; then
    echo "node_seed = \"$XRPL_VALIDATOR_SEED\"" >> $CONFIG
fi

# Validator identity. Without a [validator_identity], rxrpl runs as a
# non-validator: it cannot count itself toward UNL quorum, so a mixed
# network never reaches quorum and every consensus round force-accepts
# at max_consensus_rounds (~20-38s/ledger). Provisioning the identity
# (as xrpl-confluence does) makes rxrpl a real UNL validator that closes
# in lockstep with rippled (~4s/ledger). Reuse the validator seed for
# both the master and ephemeral key in this test harness.
if [ -n "$XRPL_VALIDATOR_SEED" ]; then
    cat >> $CONFIG <<CFGEOF

[validator_identity]
master_secret = "$XRPL_VALIDATOR_SEED"
ephemeral_seed = "$XRPL_VALIDATOR_SEED"
CFGEOF
fi

cat >> $CONFIG <<CFGEOF

[database]
path = "/var/lib/rxrpl/data"
backend = "memory"

[network]
network_id = ${XRPL_NETWORK_ID:-10000}
CFGEOF

# Convert uploaded validators.json -> trusted list in config.
if [ -f /xrpl/validators.json ]; then
    echo '[validators]' >> $CONFIG
    echo 'enabled = true' >> $CONFIG
    echo 'trusted = [' >> $CONFIG
    python3 -c "
import json
data = json.load(open('/xrpl/validators.json'))
for v in data.get('validators', []):
    print(f'    \"{v}\",')
" >> $CONFIG
    echo ']' >> $CONFIG
    # Honor explicit quorum override from the simulator (e.g.
    # `sync`'s 2-of-3 late-join setup). Without this, rxrpl
    # defaults to ceil(N * 0.8) which can be unsatisfiable when the
    # late-joining validator hasn't connected yet.
    if [ -n "$XRPL_VALIDATION_QUORUM" ]; then
        echo "quorum = $XRPL_VALIDATION_QUORUM" >> $CONFIG
    fi
else
    echo '[validators]' >> $CONFIG
    echo 'enabled = false' >> $CONFIG
    echo 'trusted = []' >> $CONFIG
fi

# Log level mapping.
case "${XRPL_LOGLEVEL:-3}" in
    0|1) LEVEL="error" ;;
    2)   LEVEL="warn" ;;
    3)   LEVEL="info" ;;
    4|5) LEVEL="debug" ;;
esac

echo "=== rxrpl config ==="
cat $CONFIG
echo "==================="

# Standalone only when explicitly requested (smoke test sets XRPL_STANDALONE=1).
# Every multi-node sim must run in network mode so the P2P listener opens —
# the first node has no bootnode but still accepts inbound peers and RPC
# connect commands.
if [ "${XRPL_STANDALONE:-0}" = "1" ]; then
    MODE="standalone"
else
    MODE="network"
fi

exec rxrpl --config $CONFIG --log-level $LEVEL run --mode $MODE --bind "0.0.0.0:${XRPL_RPC_PORT:-5005}" --close-interval 3
