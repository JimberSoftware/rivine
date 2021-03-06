// Package explorer provides a glimpse into what the network currently
// looks like.
package explorer

import (
	"errors"

	"github.com/rivine/rivine/modules"
	"github.com/rivine/rivine/persist"
	"github.com/rivine/rivine/types"
)

const (
	// ActiveBSEstimationBlocks is the number of blocks that are used to
	// estimate the active block stake used to generate blocks.
	ActiveBSEstimationBlocks = 200
)

var (
	errNilCS = errors.New("explorer cannot use a nil consensus set")
)

type (
	// blockFacts contains a set of facts about the consensus set related to a
	// certain block. The explorer needs some additional information in the
	// history so that it can calculate certain values, which is one of the
	// reasons that the explorer uses a separate struct instead of
	// modules.BlockFacts.
	blockFacts struct {
		modules.BlockFacts

		Timestamp types.Timestamp
	}

	// An Explorer contains a more comprehensive view of the blockchain,
	// including various statistics and metrics.
	Explorer struct {
		cs             modules.ConsensusSet
		db             *persist.BoltDatabase
		persistDir     string
		bcInfo         types.BlockchainInfo
		chainCts       types.ChainConstants
		rootTarget     types.Target
		genesisBlock   types.Block
		genesisBlockID types.BlockID
	}
)

// New creates the internal data structures, and subscribes to
// consensus for changes to the blockchain
func New(cs modules.ConsensusSet, persistDir string, bcInfo types.BlockchainInfo, chainCts types.ChainConstants) (*Explorer, error) {
	// Check that input modules are non-nil
	if cs == nil {
		return nil, errNilCS
	}

	// Initialize the explorer.
	genesisBlock := chainCts.GenesisBlock()
	e := &Explorer{
		cs:             cs,
		persistDir:     persistDir,
		bcInfo:         bcInfo,
		chainCts:       chainCts,
		rootTarget:     chainCts.RootTarget(),
		genesisBlock:   genesisBlock,
		genesisBlockID: genesisBlock.ID(),
	}

	// Initialize the persistent structures, including the database.
	err := e.initPersist()
	if err != nil {
		return nil, err
	}

	// retrieve the current ConsensusChangeID
	var recentChange modules.ConsensusChangeID
	err = e.db.View(dbGetInternal(internalRecentChange, &recentChange))
	if err != nil {
		return nil, err
	}

	err = cs.ConsensusSetSubscribe(e, recentChange)
	if err != nil {
		// TODO: restart from 0
		return nil, errors.New("explorer subscription failed: " + err.Error())
	}

	return e, nil
}

// Close closes the explorer.
func (e *Explorer) Close() error {
	return e.db.Close()
}
