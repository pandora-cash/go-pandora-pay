package forging

import (
	"bytes"
	"encoding/binary"
	"errors"
	"pandora-pay/addresses"
	"pandora-pay/blockchain/accounts"
	"pandora-pay/blockchain/accounts/account"
	"pandora-pay/config"
	"pandora-pay/gui"
	"pandora-pay/helpers"
	"pandora-pay/helpers/multicast"
	"pandora-pay/store"
	store_db_interface "pandora-pay/store/store-db/store-db-interface"
)

type ForgingWallet struct {
	addresses             []*ForgingWalletAddress
	addressesMap          map[string]*ForgingWalletAddress
	workersAddresses      []int
	workers               []*ForgingWorkerThread
	updateAccounts        *multicast.MulticastChannel
	updateWalletAddressCn chan *ForgingWalletAddressUpdate
	loadBalancesCn        chan struct{}
	workersCreatedCn      <-chan []*ForgingWorkerThread
	workersDestroyedCn    <-chan struct{}
	forging               *Forging
}

type ForgingWalletAddressUpdate struct {
	delegatedPriv helpers.HexBytes
	pubKey        helpers.HexBytes
}

type ForgingWalletAddress struct {
	delegatedPrivateKey *addresses.PrivateKey
	delegatedPublicKey  helpers.HexBytes //20 byte
	publicKey           helpers.HexBytes //20byte
	publicKeyStr        string
	account             *account.Account
	workerIndex         int
}

func (walletAddr *ForgingWalletAddress) clone() *ForgingWalletAddress {
	return &ForgingWalletAddress{
		walletAddr.delegatedPrivateKey,
		walletAddr.delegatedPublicKey,
		walletAddr.publicKey,
		walletAddr.publicKeyStr,
		walletAddr.account,
		walletAddr.workerIndex,
	}
}

func (w *ForgingWallet) AddWallet(delegatedPriv []byte, pubKey []byte) {
	w.updateWalletAddressCn <- &ForgingWalletAddressUpdate{
		delegatedPriv,
		pubKey,
	}
	return
}

func (w *ForgingWallet) RemoveWallet(DelegatedPublicKey []byte) { //20 byte
	w.AddWallet(nil, DelegatedPublicKey)
}

func (w *ForgingWallet) accountUpdated(addr *ForgingWalletAddress) {
	if addr.workerIndex != -1 {
		w.workers[addr.workerIndex].addWalletAddressCn <- addr.clone()
	}
}

func (w *ForgingWallet) accountInserted(addr *ForgingWalletAddress) {
	min := 0
	index := -1
	for i := 0; i < len(w.workersAddresses); i++ {
		if i == 0 || min > w.workersAddresses[i] {
			min = w.workersAddresses[i]
			index = i
		}
	}

	addr.workerIndex = index
	if index != -1 {
		w.workersAddresses[index]++
		w.workers[index].addWalletAddressCn <- addr.clone()
	}
}

func (w *ForgingWallet) accountRemoved(addr *ForgingWalletAddress) {
	if addr.workerIndex != -1 {
		w.workers[addr.workerIndex].removeWalletAddressCn <- addr.publicKeyStr
		w.workersAddresses[addr.workerIndex]--
		addr.workerIndex = -1
	}
}

func (w *ForgingWallet) processUpdates() {

	var err error

	updateAccountsCn := w.updateAccounts.AddListener()
	defer w.updateAccounts.RemoveChannel(updateAccountsCn)

	for {
		select {
		case workers, ok := <-w.workersCreatedCn:
			if !ok {
				return
			}
			w.workers = workers
			w.workersAddresses = make([]int, len(workers))
			for _, addr := range w.addresses {
				w.accountInserted(addr)
			}
		case _, ok := <-w.workersDestroyedCn:
			if !ok {
				return
			}
			w.workers = []*ForgingWorkerThread{}
			w.workersAddresses = []int{}
			for _, addr := range w.addresses {
				addr.workerIndex = -1
			}
		case update, ok := <-w.updateWalletAddressCn:
			if !ok {
				return
			}
			key := string(update.pubKey)

			//let's delete it
			if update.delegatedPriv == nil {

				if w.addressesMap[key] != nil {
					delete(w.addressesMap, key)
					for i, address := range w.addresses {
						if bytes.Equal(address.publicKey, update.pubKey) {
							w.addresses = append(w.addresses[:i], w.addresses[:i+1]...)
							w.accountRemoved(address)
							break
						}
					}
				}

			} else {

				delegatedPrivateKey := &addresses.PrivateKey{Key: update.delegatedPriv}
				delegatedPublicKey := delegatedPrivateKey.GeneratePublicKey()

				//let's read the balance
				if err = store.StoreBlockchain.DB.View(func(reader store_db_interface.StoreDBTransactionInterface) (err error) {

					chainHeight, _ := binary.Uvarint(reader.Get("chainHeight"))

					accsCollection := accounts.NewAccountsCollection(reader)
					accs, err := accsCollection.GetMap(config.NATIVE_TOKEN_FULL)
					if err != nil {
						return
					}

					var acc *account.Account
					if acc, err = accs.GetAccount(update.pubKey); err != nil {
						return
					}
					if acc == nil {
						return errors.New("Account was not found")
					}
					if acc.NativeExtra == nil {
						return errors.New("NativeExtra was not found")
					}

					if err = acc.NativeExtra.RefreshDelegatedStake(chainHeight); err != nil {
						return
					}

					if acc.NativeExtra.DelegatedStake == nil || !bytes.Equal(acc.NativeExtra.DelegatedStake.DelegatedPublicKey, delegatedPublicKey) {
						return errors.New("Delegated stake is not matching")
					}

					address := w.addressesMap[key]
					if address == nil {
						address = &ForgingWalletAddress{
							delegatedPrivateKey,
							delegatedPublicKey,
							update.pubKey,
							string(update.pubKey),
							acc,
							-1,
						}
						w.addresses = append(w.addresses, address)
						w.addressesMap[key] = address
						w.accountInserted(address)
					} else {
						address.delegatedPrivateKey = delegatedPrivateKey
						address.delegatedPublicKey = delegatedPublicKey
						address.account = acc
						w.accountUpdated(address)
					}

					return
				}); err != nil {
					gui.GUI.Error(err)
				}

			}
		case accsCollectionData, ok := <-updateAccountsCn:
			if !ok {
				return
			}

			accsCollection := accsCollectionData.(*accounts.AccountsCollection)
			accs, err := accsCollection.GetExistingMap(config.NATIVE_TOKEN_FULL)
			if err != nil {
				return
			}

			for k, v := range accs.HashMap.Committed {
				if w.addressesMap[k] != nil {
					if v.Stored == "update" {
						w.addressesMap[k].account = v.Element.(*account.Account)
						w.accountUpdated(w.addressesMap[k])
					} else if v.Stored == "delete" {
						w.addressesMap[k].account = nil
						w.accountUpdated(w.addressesMap[k])
					}
				}
			}

		case _, ok := <-w.loadBalancesCn:
			if !ok {
				return
			}

			if err = store.StoreBlockchain.DB.View(func(reader store_db_interface.StoreDBTransactionInterface) (err error) {

				accsCollection := accounts.NewAccountsCollection(reader)
				accs, err := accsCollection.GetMap(config.NATIVE_TOKEN_FULL)
				if err != nil {
					return
				}

				for _, address := range w.addresses {

					var acc *account.Account
					if acc, err = accs.GetAccount(address.publicKey); err != nil {
						return
					}

					address.account = acc
					w.accountUpdated(address)
				}

				return nil
			}); err != nil {
				gui.GUI.Error(err)
			}
		}

	}

}

func (w *ForgingWallet) loadBalances() {
	w.loadBalancesCn <- struct{}{}
}
