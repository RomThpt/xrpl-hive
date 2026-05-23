package main

import (
	"context"
	"fmt"
	"time"

	"github.com/xrpl-commons/xrpl-hive/xrplsim"
)

func main() {
	suite := xrplsim.Suite{
		Name:        "consensus",
		Description: "Consensus agreement across mixed XRPL validator sets.",
	}

	// Test: mixed validators reach consensus and agree on ledger hashes.
	suite.Add(xrplsim.TestSpec{
		Name:        "mixed-validator-hash-agreement",
		Description: "All nodes in a mixed validator set agree on ledger hashes.",
		Run: func(t *xrplsim.T) {
			clients, err := t.Sim.ClientTypes()
			if err != nil {
				t.Fatal("failed to get client types:", err)
			}
			if len(clients) < 2 {
				t.Fatal("need at least 2 client types for consensus test")
			}

			topo := xrplsim.NewTopology(len(clients))

			if err := t.Sim.CreateNetwork(t.SuiteID, "consensus-net"); err != nil {
				t.Fatal("failed to create network:", err)
			}

			// Start nodes.
			var nodes []*xrplsim.Client
			var peerAddrs []string
			for i, cd := range clients {
				c := t.StartClient(cd.Name,
					xrplsim.WithValidatorConfig(topo, i, peerAddrs),
					xrplsim.WithInitialNetworks([]string{"consensus-net"}),
				)
				ip, _ := t.Sim.ContainerNetworkIP(t.SuiteID, "consensus-net", c.Container)
				peerAddrs = append(peerAddrs, fmt.Sprintf("%s:%d", ip, xrplsim.DefaultPeerPort))
				nodes = append(nodes, c)
			}

			// Peer all nodes.
			for i, node := range nodes {
				rpc := xrplsim.NewRPCClient(node.RPCEndpoint())
				for j, peer := range nodes {
					if i == j {
						continue
					}
					peerIP, _ := t.Sim.ContainerNetworkIP(t.SuiteID, "consensus-net", peer.Container)
					rpc.Connect(peerIP, xrplsim.DefaultPeerPort)
				}
			}

			// Wait for all nodes to advance to ledger 10.
			ctx := context.Background()
			targetSeq := 10
			// Cross-impl mixed networks pay an initial peer
			// handshake + catchup tax (rxrpl boots at seq=1 and
			// must adopt the peer's chain before participating in
			// consensus rounds), then average ~12s/ledger in
			// steady state with the default rippled idle interval
			// of 20s. 120s wasn't enough headroom: the test would
			// fire its timeout while consensus was actively
			// advancing. 300s scopes the timeout to "consensus
			// genuinely stuck" rather than "still warming up".
			for i, node := range nodes {
				rpc := xrplsim.NewRPCClient(node.RPCEndpoint())
				if err := rpc.WaitForLedger(ctx, targetSeq, 300*time.Second); err != nil {
					t.Fatalf("node %s did not reach ledger %d: %v", clients[i].Name, targetSeq, err)
				}
			}

			// Compare ledger hashes using oracle.
			var oracleNodes []xrplsim.OracleNode
			for i, node := range nodes {
				oracleNodes = append(oracleNodes, xrplsim.OracleNode{
					Name:   clients[i].Name,
					Client: xrplsim.NewRPCClient(node.RPCEndpoint()),
				})
			}

			oracle := xrplsim.NewOracle(oracleNodes)
			checkSeq := targetSeq - 2 // Check a safely finalized ledger.
			comp, err := oracle.CompareAtSequence(ctx, checkSeq)
			if err != nil {
				t.Fatal("oracle comparison failed:", err)
			}

			if !comp.Agreed {
				t.Logf("DIVERGENCE at ledger %d:", checkSeq)
				for _, d := range comp.Divergences {
					t.Log("  ", d)
				}
				t.Fatal("nodes disagree on ledger hash")
			}

			t.Logf("all %d nodes agree on ledger %d hash", len(nodes), checkSeq)
		},
	})

	xrplsim.MustRun(xrplsim.New(), suite)
}
