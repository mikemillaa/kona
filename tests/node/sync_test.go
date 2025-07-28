package node

import (
	"strings"
	"sync"
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// Check that all the nodes in the network are synced to the local safe block and can catch up to the sequencer node.
func TestL2SafeSync(gt *testing.T) {
	t := devtest.ParallelT(gt)

	out := NewMixedOpKona(t)

	nodes := out.L2CLKonaNodes

	checkFuns := make([]dsl.CheckFunc, 0, 2*len(nodes))

	for _, node := range nodes {
		checkFuns = append(checkFuns, node.ReachedFn(types.LocalSafe, 20, 40))
		checkFuns = append(checkFuns, node.MatchedFn(&nodes[0], types.LocalSafe, 40))
	}

	dsl.CheckAll(t, checkFuns...)
}

// Check that all the nodes in the network are synced to the local unsafe block and can catch up to the sequencer node.
func TestL2UnsafeSync(gt *testing.T) {
	t := devtest.ParallelT(gt)

	out := NewMixedOpKona(t)

	nodes := out.L2CLKonaNodes

	checkFuns := make([]dsl.CheckFunc, 0, 2*len(nodes))

	for _, node := range nodes {
		checkFuns = append(checkFuns, node.ReachedFn(types.LocalUnsafe, 40, 40))
		checkFuns = append(checkFuns, node.MatchedFn(&nodes[0], types.LocalUnsafe, 40))
	}

	dsl.CheckAll(t, checkFuns...)
}

// Check that all the kona nodes in the network are synced to the finalized block.
func TestL2FinalizedSync(gt *testing.T) {
	t := devtest.ParallelT(gt)

	out := NewMixedOpKona(t)

	nodes := out.L2CLKonaNodes

	checkFuns := make([]dsl.CheckFunc, 0, 2*len(nodes))

	for _, node := range nodes {
		checkFuns = append(checkFuns, node.ReachedFn(types.Finalized, 10, 400))
	}

	dsl.CheckAll(t, checkFuns...)
}

func isSequencer(node string) bool {
	return strings.Contains(node, string(Sequencer))
}

func filterSequencerCL(nodes []dsl.L2CLNode) []dsl.L2CLNode {
	out := make([]dsl.L2CLNode, 0, len(nodes))
	for _, node := range nodes {
		if isSequencer(node.Escape().ID().Key()) {
			out = append(out, node)
		}
	}
	return out
}

func filterSequencerEL(nodes []dsl.L2ELNode) []dsl.L2ELNode {
	out := make([]dsl.L2ELNode, 0, len(nodes))
	for _, node := range nodes {
		if isSequencer(node.Escape().ID().Key()) {
			out = append(out, node)
		}
	}
	return out
}


func TestSyncWithSequencer(gt *testing.T) {
	t := devtest.ParallelT(gt)

	out := NewMixedOpKona(t)

	nodes := out.L2CLNodes()

	// Find the sequencer nodes.
	sequencers := filterSequencerCL(nodes)
	t.Gate().Equal(len(sequencers), 1, "expected exactly one sequencer")
	sequencer := sequencers[0]

	// Check that all the nodes are lagging behind the sequencer for the local unsafe head.
	var wg sync.WaitGroup
	for _, node := range nodes {
		wg.Add(1)
		go func(node *dsl.L2CLNode) {
			defer wg.Done()
			node.Lagged(&sequencer, types.LocalUnsafe, 40, true)
		}(&node)
	}
	wg.Wait()
}

func TestL2TransactionInclusion(gt *testing.T) {
	t := devtest.ParallelT(gt)
	out := NewMixedOpKona(t)

	originNode := filterSequencerEL(out.L2ELNodes())[0]
	funder := dsl.NewFunder(out.Wallet, out.Faucet, originNode)

	user := funder.NewFundedEOA(eth.OneEther)
	to := out.Wallet.NewEOA(originNode)
	toInitialBalance := to.GetBalance()
	tx := user.Transfer(to.Address(), eth.HalfEther)

	inclusionBlock, err := tx.IncludedBlock.Eval(t.Ctx())
	if err != nil {
		gt.Fatal("transaction receipt not found", "error", err)
	}

	// Ensure the block containing the transaction has propagated to the rest of the network.
	for _, node := range out.L2ELNodes() {
		block := node.WaitForBlockNumber(inclusionBlock.Number)
		blockID := block.ID()

		// It's possible that the block has already been included, and `WaitForBlockNumber` returns a block
		// at a taller height.
		if block.Number > inclusionBlock.Number {
			blockID = node.BlockRefByNumber(inclusionBlock.Number).ID()
		}

		// Ensure that the block ID matches the expected inclusion block hash.
		if blockID.Hash != inclusionBlock.Hash {
			gt.Fatal("transaction not included in block", "node", node.String(), "expectedBlockHash", inclusionBlock.Hash, "actualBlockHash", blockID.Hash)
		}

		// Ensure that the recipient's balance has been updated in the eyes of the EL node.
		to.AsEL(node).VerifyBalanceExact(toInitialBalance.Add(eth.HalfEther))
	}
}
