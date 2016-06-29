package consensus

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/NebulousLabs/Sia/build"
	"github.com/NebulousLabs/Sia/modules"
	"github.com/NebulousLabs/Sia/persist"

	"github.com/NebulousLabs/bolt"
)

const (
	// DatabaseFilename contains the filename of the database that will be used
	// when managing consensus.
	DatabaseFilename = modules.ConsensusDir + ".db"
	logFile          = modules.ConsensusDir + ".log"
)

// loadDB pulls all the blocks that have been saved to disk into memory, using
// them to fill out the ConsensusSet.
func (cs *ConsensusSet) loadDB() error {
	// Open the database - a new bolt database will be created if none exists.
	err := cs.openDB(filepath.Join(cs.persistDir, DatabaseFilename))
	if err != nil {
		return err
	}

	// Walk through initialization for Sia.
	return cs.db.Update(func(tx *bolt.Tx) error {
		// Check if the database has been initialized.
		if !dbInitialized(tx) {
			return cs.initDB(tx)
		}

		// Check that inconsistencies have not been detected in the database.
		if inconsistencyDetected(tx) {
			return errors.New("database contains inconsistencies")
		}

		// Check that the genesis block is correct - typically only incorrect
		// in the event of developer binaries vs. release binaires.
		genesisID, err := getPath(tx, 0)
		if build.DEBUG && err != nil {
			panic(err)
		}
		if genesisID != cs.blockRoot.Block.ID() {
			return errors.New("Blockchain has wrong genesis block, exiting.")
		}
		return nil
	})
}

// initPersist initializes the persistence structures of the consensus set, in
// particular loading the database and preparing to manage subscribers.
func (cs *ConsensusSet) initPersist() error {
	// Create the consensus directory.
	err := os.MkdirAll(cs.persistDir, 0700)
	if err != nil {
		return err
	}

	// Initialize the logger.
	cs.log, err = persist.NewFileLogger(filepath.Join(cs.persistDir, logFile))
	if err != nil {
		return err
	}

	// Try to load an existing database from disk - a new one will be created
	// if one does not exist.
	err = cs.loadDB()
	if err != nil {
		return err
	}
	return nil
}
