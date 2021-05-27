package store_db_bunt

import (
	"github.com/tidwall/buntdb"
	"os"
	store_db_interface "pandora-pay/store/store-db/store-db-interface"
)

const dbName = "bunt"

type StoreDBBunt struct {
	store_db_interface.StoreDBInterface
	DB   *buntdb.DB
	Name []byte
}

func (store *StoreDBBunt) Close() error {
	return store.DB.Close()
}

func (store *StoreDBBunt) View(callback func(dbTx store_db_interface.StoreDBTransactionInterface) error) error {
	return store.DB.View(func(buntTx *buntdb.Tx) error {
		tx := &StoreDBBuntTransaction{
			buntTx: buntTx,
		}
		return callback(tx)
	})
}

func (store *StoreDBBunt) Update(callback func(dbTx store_db_interface.StoreDBTransactionInterface) error) error {
	return store.DB.Update(func(buntTx *buntdb.Tx) error {
		tx := &StoreDBBuntTransaction{
			buntTx: buntTx,
		}
		return callback(tx)
	})
}

func CreateStoreDBBunt(name string, inMemory bool) (store *StoreDBBunt, err error) {

	store = &StoreDBBunt{
		Name: []byte(name),
	}

	var prefix string
	if !inMemory {
		prefix = "./store"
		if _, err = os.Stat(prefix); os.IsNotExist(err) {
			if err = os.Mkdir(prefix, 0755); err != nil {
				return
			}
		}
		prefix += name + "_store" + "." + dbName
	} else {
		prefix = ":memory:"
	}

	// Open the my.store data file in your current directory.
	// It will be created if it doesn't exist.
	if store.DB, err = buntdb.Open(prefix); err != nil {
		return
	}

	return
}