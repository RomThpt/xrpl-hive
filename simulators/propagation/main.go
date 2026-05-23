package main

import (
	"context"
	"fmt"
	"time"

	"github.com/xrpl-commons/xrpl-hive/xrplsim"
)

func main() {
	suite := xrplsim.Suite{
		Name:        "propagation",
		Description: "Transaction propagation across XRPL implementations.",
	}

	// Test: submit payment to one impl, verify on another.
	suite.Add(xrplsim.TestSpec{
		Name:        "cross-impl-payment",
		Description: "Submit payment to implementation A, verify account exists on implementation B.",
		Run: func(t *xrplsim.T) {
			clients, err := t.Sim.ClientTypes()
			if err != nil {
				t.Fatal("failed to get client types:", err)
			}
			if len(clients) < 2 {
				t.Fatal("need at least 2 client types for cross-impl propagation test")
			}

			topo := xrplsim.NewTopology(len(clients))

			// Create network.
			if err := t.Sim.CreateNetwork(t.SuiteID, "testnet"); err != nil {
				t.Fatal("failed to create network:", err)
			}

			// Start one node of each client type.
			var nodes []*xrplsim.Client
			var peerAddrs []string
			for i, cd := range clients {
				c := t.StartClient(cd.Name,
					xrplsim.WithValidatorConfig(topo, i, peerAddrs),
					xrplsim.WithInitialNetworks([]string{"testnet"}),
				)
				ip, err := t.Sim.ContainerNetworkIP(t.SuiteID, "testnet", c.Container)
				if err != nil {
					t.Fatal("failed to get container IP:", err)
				}
				peerAddrs = append(peerAddrs, fmt.Sprintf("%s:%d", ip, xrplsim.DefaultPeerPort))
				nodes = append(nodes, c)
			}

			// Connect all nodes to each other via admin RPC.
			for i, node := range nodes {
				rpc := xrplsim.NewRPCClient(node.RPCEndpoint())
				for j, peer := range nodes {
					if i == j {
						continue
					}
					peerIP, _ := t.Sim.ContainerNetworkIP(t.SuiteID, "testnet", peer.Container)
					if err := rpc.Connect(peerIP, xrplsim.DefaultPeerPort); err != nil {
						t.Logf("warning: connect %s -> %s failed: %v", clients[i].Name, clients[j].Name, err)
					}
				}
			}

			// Wait for all nodes to reach ledger 5.
			// Cross-impl mixed networks pay an initial peer
			// handshake + catchup tax (~30-60s) before consensus
			// rounds can finalize, then average ~12s/ledger in
			// steady state. 120s leaves no headroom for that
			// startup phase even though the test only asks for
			// seq=5; bump to 300s so the timeout only fires on
			// real failures.
			ctx := context.Background()
			for i, node := range nodes {
				rpc := xrplsim.NewRPCClient(node.RPCEndpoint())
				if err := rpc.WaitForLedger(ctx, 5, 300*time.Second); err != nil {
					t.Fatalf("node %s did not reach ledger 5: %v", clients[i].Name, err)
				}
			}

			// Submit payment via node 0.
			rpc0 := xrplsim.NewRPCClient(nodes[0].RPCEndpoint())
			result, err := rpc0.SubmitPayment(
				xrplsim.GenesisSecret,
				xrplsim.GenesisAddress,
				"ra8sezk7XT7JRgE1myhUBZJUDCUH3qrWMU",
				"100000000",
			)
			if err != nil {
				t.Fatal("submit payment failed:", err)
			}
			t.Logf("submitted via %s: %s = %s", clients[0].Name, result.TxHash, result.EngineResult)

			// Wait for propagation. 3 ledgers (5→8) at ~12s/ledger
			// = ~36s, plus tx propagation + inclusion overhead.
			// 60s was tight even in steady state; 120s gives the
			// transaction time to land without falsely tripping.
			rpc1 := xrplsim.NewRPCClient(nodes[1].RPCEndpoint())
			if err := rpc1.WaitForLedger(ctx, 8, 120*time.Second); err != nil {
				t.Fatalf("node %s did not advance: %v", clients[1].Name, err)
			}

			// Verify account exists on node 1.
			acct, err := rpc1.AccountInfo("ra8sezk7XT7JRgE1myhUBZJUDCUH3qrWMU")
			if err != nil {
				t.Fatalf("account not found on %s: %v", clients[1].Name, err)
			}
			t.Logf("account balance on %s: %s", clients[1].Name, acct.Balance)
		},
	})

	xrplsim.MustRun(xrplsim.New(), suite)
}
