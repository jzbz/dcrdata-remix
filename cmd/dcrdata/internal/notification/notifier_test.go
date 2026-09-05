// Copyright (c) 2019-2021, The Decred developers

package notification

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/decred/dcrd/chaincfg/chainhash"
	chainjson "github.com/decred/dcrd/rpc/jsonrpc/types/v4"
	"github.com/decred/dcrd/wire"
	"github.com/decred/dcrdata/v8/txhelpers"
)

type dummyNode struct{}

func (node *dummyNode) NotifyBlocks(context.Context) error                { return nil }
func (node *dummyNode) NotifyNewTransactions(context.Context, bool) error { return nil }
func (node *dummyNode) NotifyWinningTickets(context.Context) error        { return nil }

var counter int64
var hashTails = []string{"00", "01", "02", "03", "04", "05", "06", "07", "08", "09"}

func newHash() *chainhash.Hash {
	counter++
	h, _ := chainhash.NewHash([]byte("000000000000000000000000000000" + hashTails[int(counter)%len(hashTails)]))
	return h
}

func (node *dummyNode) GetBestBlock(context.Context) (*chainhash.Hash, int64, error) {
	hash := newHash()
	return hash, counter, nil
}

var commonAncestorHash = newHash()
var commonAncestor = &wire.MsgBlock{
	Header: wire.BlockHeader{
		PrevBlock: *commonAncestorHash,
		Height:    uint32(5),
	},
}

// GetBlock will only be called by rpcutils.CommonAncestor, so it should return
// the same block every time.
func (node *dummyNode) GetBlock(_ context.Context, blockHash *chainhash.Hash) (*wire.MsgBlock, error) {
	return commonAncestor, nil
}
func (node *dummyNode) GetBlockHash(_ context.Context, blockHeight int64) (*chainhash.Hash, error) {
	hash := newHash()
	return hash, nil
}
func (node *dummyNode) GetBlockHeaderVerbose(_ context.Context, hash *chainhash.Hash) (*chainjson.GetBlockHeaderVerboseResult, error) {
	return nil, nil
}

var callCounter int

// testTxHandler will be tested async
var mtx sync.RWMutex
var wg = new(sync.WaitGroup)
var notifier *Notifier

func testTxHandler(_ *chainjson.TxRawResult) error {
	mtx.Lock()
	defer mtx.Unlock()
	defer wg.Done()
	callCounter++
	return nil
}

var testTxHandler2 = testTxHandler

func testBlockHandler(_ *wire.BlockHeader) error {
	defer wg.Done()
	callCounter++
	return nil
}
func testBlockHandlerLite(_ uint32, _ string) error {
	defer wg.Done()
	callCounter++
	return nil
}
func testReorgHandler(reorg *txhelpers.ReorgData) error {
	defer wg.Done()
	callCounter++
	notifier.SetPreviousBlock(reorg.NewChainHead, uint32(reorg.NewChainHeight))
	return nil
}

func TestNotifier(t *testing.T) {

	notifier = NewNotifier()
	signals := notifier.DcrdHandlers()
	notifier.RegisterTxHandlerGroup(testTxHandler, testTxHandler2)
	notifier.RegisterBlockHandlerGroup(testBlockHandler)
	notifier.RegisterBlockHandlerLiteGroup(testBlockHandlerLite)
	notifier.RegisterReorgHandlerGroup(testReorgHandler)
	wg.Add(5)

	ctx, shutdown := context.WithCancel(context.Background())
	defer shutdown()

	notifier.Listen(ctx, &dummyNode{})

	prevBlock := newHash()
	header := wire.BlockHeader{
		PrevBlock: *prevBlock,
		Height:    uint32(counter),
	}
	notifier.previous.hash = *prevBlock
	bytes, _ := header.Bytes()
	signals.OnBlockConnected(bytes, nil)

	oldHash := newHash()
	ohdHeight := int32(counter)
	newHash := newHash()
	newHeight := counter
	signals.OnReorganization(oldHash, ohdHeight, newHash, int32(newHeight))

	signals.OnTxAcceptedVerbose(new(chainjson.TxRawResult))

	wg.Wait()

	if notifier.previous.hash.String() != newHash.String() {
		t.Errorf("unexpected previous.hash after reorg. %s != %s",
			notifier.previous.hash.String(), newHash.String())
	}

	if notifier.previous.height != uint32(newHeight) {
		t.Errorf("unexpected previous.height after reorg. %d != %d",
			notifier.previous.height, uint32(newHeight))
	}

	if callCounter != 5 {
		t.Errorf("callCounter = %d. Should be 5.", callCounter)
	}

	shutdown()
}

// hashFrom builds a distinct, deterministic hash without touching the package
// level counter the other test relies on.
func hashFrom(b byte) chainhash.Hash {
	var h chainhash.Hash
	h[0] = b
	return h
}

// TestProcessBlockHandlerFailure covers the reason handler groups are ordered:
// the stake database is connected before the block is stored in PostgreSQL, so
// a group that runs after an earlier one failed advances one store past the
// other. dcrdata cannot repair that itself — the only rewind fires when the
// stake database is ahead — and the next batch sync panics on it. A failing
// handler must therefore abandon the block entirely.
func TestProcessBlockHandlerFailure(t *testing.T) {
	prev := hashFrom(0xab)

	t.Run("failure abandons the block", func(t *testing.T) {
		n := NewNotifier()
		n.previous.hash = prev
		n.previous.height = 99

		var firstRan, secondRan atomic.Bool
		n.RegisterBlockHandlerGroup(func(*wire.BlockHeader) error {
			firstRan.Store(true)
			return errors.New("stake database unavailable")
		})
		n.RegisterBlockHandlerGroup(func(*wire.BlockHeader) error {
			secondRan.Store(true)
			return nil
		})

		n.processBlock(&wire.BlockHeader{PrevBlock: prev, Height: 100})

		if !firstRan.Load() {
			t.Fatal("first handler group never ran")
		}
		if secondRan.Load() {
			t.Error("second handler group ran after the first failed; this is the " +
				"desync that leaves PostgreSQL ahead of the stake database")
		}
		if n.previous.hash != prev || n.previous.height != 99 {
			t.Errorf("block recorded as connected despite a handler failure: got (%v, %d), want (%v, 99)",
				n.previous.hash, n.previous.height, prev)
		}
	})

	t.Run("failure invokes the fatal handler", func(t *testing.T) {
		n := NewNotifier()
		n.previous.hash = prev

		var fatalCalls atomic.Int32
		n.SetFatalHandler(func() { fatalCalls.Add(1) })
		n.RegisterBlockHandlerGroup(func(*wire.BlockHeader) error {
			return errors.New("stake database unavailable")
		})

		n.processBlock(&wire.BlockHeader{PrevBlock: prev, Height: 100})

		// Without this the notifier goes quiet: it stops connecting blocks while
		// the status monitor still reports ready, because the node and DB
		// heights freeze together. A shutdown makes the failure visible and lets
		// the batch sync recover on restart.
		if got := fatalCalls.Load(); got != 1 {
			t.Errorf("fatal handler called %d times, want 1", got)
		}
	})

	t.Run("success does not invoke the fatal handler", func(t *testing.T) {
		n := NewNotifier()
		n.previous.hash = prev

		var fatalCalls atomic.Int32
		n.SetFatalHandler(func() { fatalCalls.Add(1) })
		n.RegisterBlockHandlerGroup(func(*wire.BlockHeader) error { return nil })

		n.processBlock(&wire.BlockHeader{PrevBlock: prev, Height: 100})

		if got := fatalCalls.Load(); got != 0 {
			t.Errorf("fatal handler called %d times on success, want 0", got)
		}
	})

	t.Run("success still advances", func(t *testing.T) {
		n := NewNotifier()
		n.previous.hash = prev
		n.previous.height = 99

		var secondRan atomic.Bool
		n.RegisterBlockHandlerGroup(func(*wire.BlockHeader) error { return nil })
		n.RegisterBlockHandlerGroup(func(*wire.BlockHeader) error {
			secondRan.Store(true)
			return nil
		})

		bh := &wire.BlockHeader{PrevBlock: prev, Height: 100}
		n.processBlock(bh)

		if !secondRan.Load() {
			t.Error("second handler group did not run after the first succeeded")
		}
		if n.previous.hash != bh.BlockHash() || n.previous.height != 100 {
			t.Errorf("block not recorded as connected: got (%v, %d), want (%v, 100)",
				n.previous.hash, n.previous.height, bh.BlockHash())
		}
	})
}

// TestSignalReorgHandlerFailure is the same guarantee for the reorg path, whose
// groups are ordered the same way.
func TestSignalReorgHandlerFailure(t *testing.T) {
	n := NewNotifier()
	n.node = &dummyNode{}
	prev := hashFrom(0xcd)
	n.previous.hash = prev
	n.previous.height = 99

	var secondRan atomic.Bool
	n.RegisterReorgHandlerGroup(func(*txhelpers.ReorgData) error {
		return errors.New("stake database reorg failed")
	})
	n.RegisterReorgHandlerGroup(func(*txhelpers.ReorgData) error {
		secondRan.Store(true)
		return nil
	})

	n.signalReorg(BranchTips{
		OldChainHead:   hashFrom(0x01),
		OldChainHeight: 100,
		NewChainHead:   hashFrom(0x02),
		NewChainHeight: 100,
	})

	if secondRan.Load() {
		t.Error("second reorg handler group ran after the first failed")
	}
	if n.previous.hash != prev || n.previous.height != 99 {
		t.Errorf("new chain head recorded despite a handler failure: got (%v, %d), want (%v, 99)",
			n.previous.hash, n.previous.height, prev)
	}
}
