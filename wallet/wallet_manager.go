package wallet

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/tyler-smith/go-bip32"
	"golang.org/x/exp/slices"
	"math/rand"
	"pandora-pay/address_balance_decrypter"
	"pandora-pay/addresses"
	"pandora-pay/blockchain/data_storage"
	"pandora-pay/blockchain/data_storage/accounts/account"
	"pandora-pay/blockchain/data_storage/plain_accounts/plain_account"
	"pandora-pay/blockchain/data_storage/registrations/registration"
	"pandora-pay/blockchain/forging"
	"pandora-pay/config/config_nodes"
	"pandora-pay/config/globals"
	"pandora-pay/cryptography"
	"pandora-pay/cryptography/crypto"
	"pandora-pay/store"
	"pandora-pay/store/store_db/store_db_interface"
	"pandora-pay/wallet/wallet_address"
	"pandora-pay/wallet/wallet_address/shared_staked"
	"strconv"
)

func (self *wallet) GetAddressesCount() int {
	self.Lock.RLock()
	defer self.Lock.RUnlock()
	return len(self.Addresses)
}

func (self *wallet) GetRandomAddress() *wallet_address.WalletAddress {
	self.Lock.RLock()
	defer self.Lock.RUnlock()
	index := rand.Intn(len(self.Addresses))
	return self.Addresses[index].Clone()
}

func (self *wallet) GetFirstStakedAddress(lock bool) (*wallet_address.WalletAddress, error) {

	if lock {
		self.Lock.RLock()
	}

	var found *wallet_address.WalletAddress
	for _, addr := range self.Addresses {
		if addr.Staked {
			found = addr
			break
		}
	}

	if lock {
		self.Lock.RUnlock()
	}
	if found != nil {
		return found, nil
	}

	return self.AddNewAddress(true, "", true, false, true)
}

func (self *wallet) GetFirstAddressForDevnetGenesisAirdrop() (string, *shared_staked.WalletAddressSharedStakedAddressExported, error) {

	addr, err := self.GetFirstStakedAddress(true)
	if err != nil {
		return "", nil, err
	}

	sharedStakedAddress, err := self.exportSharedStakedAddress(addr, "", false)
	if err != nil {
		return "", nil, err
	}

	return addr.AddressRegistrationEncoded, sharedStakedAddress, nil
}

func (self *wallet) GetWalletAddressByEncodedAddress(addressEncoded string, lock bool) (*wallet_address.WalletAddress, error) {

	address, err := addresses.DecodeAddr(addressEncoded)
	if err != nil {
		return nil, err
	}

	addr := self.GetWalletAddressByPublicKey(address.PublicKey, lock)
	if addr == nil {
		return nil, errors.New("Address was not found")
	}

	return addr, nil
}

func (self *wallet) GetWalletAddressByPublicKeyString(publicKeyStr string, lock bool) (*wallet_address.WalletAddress, error) {
	publicKey, err := base64.StdEncoding.DecodeString(publicKeyStr)
	if err != nil {
		return nil, err
	}
	return self.GetWalletAddressByPublicKey(publicKey, lock), nil
}

func (self *wallet) GetWalletAddressByPublicKey(publicKey []byte, lock bool) *wallet_address.WalletAddress {

	if lock {
		self.Lock.RLock()
		defer self.Lock.RUnlock()
	}

	return self.addressesMap[string(publicKey)].Clone()
}

func (self *wallet) ImportSecretKey(name string, secret []byte, staked, spendRequired bool) (*wallet_address.WalletAddress, error) {

	secretChild, err := bip32.Deserialize(secret)
	if err != nil {
		return nil, err
	}

	start := bip32.FirstHardenedChild

	if self.nonHardening { //non hardened
		start = 0
	}

	privKey, err := secretChild.NewChildKey(start + 0)
	if err != nil {
		return nil, err
	}

	spendPrivKey, err := secretChild.NewChildKey(start + 1)
	if err != nil {
		return nil, err
	}

	privateKey, err := addresses.NewPrivateKey(privKey.Key)
	if err != nil {
		return nil, err
	}

	spendPrivateKey, err := addresses.NewPrivateKey(spendPrivKey.Key)
	if err != nil {
		return nil, err
	}

	addr := &wallet_address.WalletAddress{
		Name:            name,
		SecretKey:       secret,
		PrivateKey:      privateKey,
		SeedIndex:       0,
		IsImported:      true,
		SpendPrivateKey: spendPrivateKey,
		IsMine:          true,
	}

	if err = self.AddAddress(addr, staked, spendRequired, true, false, false, true); err != nil {
		return nil, err
	}

	return addr.Clone(), nil
}

func (self *wallet) AddSharedStakedAddress(addr *wallet_address.WalletAddress, lock, hasAccount bool, account *account.Account, reg *registration.Registration, chainHeight uint64) (err error) {

	if lock {
		self.Lock.Lock()
		defer self.Lock.Unlock()
	}
	if !self.Loaded {
		return errors.New("Wallet was not loaded!")
	}

	if self.Count > config_nodes.DELEGATES_MAXIMUM {
		return errors.New("DELEGATES_MAXIMUM exceeded")
	}

	address, err := addresses.CreateAddr(addr.PublicKey, addr.Staked, addr.SpendPublicKey, nil, nil, 0, nil)
	if err != nil {
		return
	}

	addr.AddressEncoded = address.EncodeAddr()

	if self.addressesMap[string(addr.PublicKey)] != nil {
		return errors.New("Address exists")
	}

	self.Addresses = append(self.Addresses, addr)
	self.addressesMap[string(addr.PublicKey)] = addr

	forging.Forging.Wallet.AddWallet(addr.PublicKey, addr.SharedStaked, hasAccount, account, reg, chainHeight)

	self.Count += 1

	self.updateWallet()

	if err = self.saveWallet(len(self.Addresses)-1, len(self.Addresses), -1, false); err != nil {
		return
	}
	globals.MainEvents.BroadcastEvent("wallet/added", addr)

	return
}

func (self *wallet) AddAddress(addr *wallet_address.WalletAddress, staked, spendRequired, lock bool, incrementSeedIndex, incrementImportedCountIndex, save bool) (err error) {

	if lock {
		self.Lock.Lock()
		defer self.Lock.Unlock()
	}

	if !self.Loaded {
		return errors.New("Wallet was not loaded!")
	}

	if addr.SpendPrivateKey != nil {
		addr.SpendPublicKey = addr.SpendPrivateKey.GeneratePublicKey()
	}

	var spendPublicKey []byte
	if spendRequired {
		if len(addr.SpendPublicKey) != cryptography.PublicKeySize {
			return errors.New("Spend Public Key is missing")
		}
		spendPublicKey = addr.SpendPublicKey
	}

	var addr1, addr2 *addresses.Address

	if addr1, err = addr.PrivateKey.GenerateAddress(staked, spendPublicKey, false, nil, 0, nil); err != nil {
		return
	}
	if addr2, err = addr.PrivateKey.GenerateAddress(staked, spendPublicKey, true, nil, 0, nil); err != nil {
		return
	}

	publicKey := addr.PrivateKey.GeneratePublicKey()

	addr.Staked = staked
	addr.SpendRequired = spendRequired
	addr.AddressEncoded = addr1.EncodeAddr()
	addr.AddressRegistrationEncoded = addr2.EncodeAddr()
	addr.Registration = addr2.Registration
	addr.PublicKey = publicKey

	if addr.PrivateKey != nil {
		if addr.SharedStaked, err = addr.DeriveSharedStaked(); err != nil {
			return
		}
	}

	if self.addressesMap[string(addr.PublicKey)] != nil {
		return errors.New("Address exists")
	}

	self.Addresses = append(self.Addresses, addr)
	self.addressesMap[string(addr.PublicKey)] = addr

	self.Count += 1

	if incrementSeedIndex {
		self.SeedIndex += 1
	}
	if incrementImportedCountIndex {
		addr.Name = "Imported Address " + strconv.Itoa(self.CountImportedIndex)
		self.CountImportedIndex += 1
	}

	if err = forging.Forging.Wallet.AddWallet(addr.PublicKey, addr.SharedStaked, false, nil, nil, 0); err != nil {
		return
	}

	if save {
		self.updateWallet()

		if err = self.saveWallet(len(self.Addresses)-1, len(self.Addresses), -1, false); err != nil {
			return
		}
		globals.MainEvents.BroadcastEvent("wallet/added", addr)
	}

	return

}

func (self *wallet) GenerateKeys(seedIndex uint32, lock bool) ([]byte, []byte, []byte, error) {

	if lock {
		self.Lock.Lock()
		defer self.Lock.Unlock()
	}

	if !self.Loaded {
		return nil, nil, nil, errors.New("Wallet was not loaded!")
	}

	seedExtend := &addresses.SeedExtended{}
	if err := seedExtend.Deserialize(self.Seed); err != nil {
		return nil, nil, nil, err
	}

	masterKey, err := bip32.NewMasterKey(seedExtend.Key)
	if err != nil {
		return nil, nil, nil, err
	}

	start := bip32.FirstHardenedChild

	if seedExtend.Version == addresses.SIMPLE_PRIVATE_KEY || self.nonHardening { //non hardening
		start = 0
	}

	secret, err := masterKey.NewChildKey(start + seedIndex)
	if err != nil {
		return nil, nil, nil, err
	}

	key2, err := secret.NewChildKey(start + 0)
	if err != nil {
		return nil, nil, nil, err
	}

	key3, err := secret.NewChildKey(start + 1)
	if err != nil {
		return nil, nil, nil, err
	}

	secretSerialized, err := secret.Serialize()
	if err != nil {
		return nil, nil, nil, err
	}

	return secretSerialized, key2.Key, key3.Key, nil
}

func (self *wallet) GenerateNextAddress(lock bool) (*addresses.Address, error) {

	//avoid generating the same address twice
	if lock {
		self.Lock.RLock()
		defer self.Lock.RUnlock()
	}

	if !self.Loaded {
		return nil, errors.New("Wallet was not loaded!")
	}

	_, privateKey, _, err := self.GenerateKeys(self.SeedIndex, false)
	if err != nil {
		return nil, err
	}

	privKey, err := addresses.NewPrivateKey(privateKey)
	if err != nil {
		return nil, err
	}

	return privKey.GenerateAddress(false, nil, true, nil, 0, nil)
}

func (self *wallet) AddNewAddress(lock bool, name string, staked, spendRequired, save bool) (*wallet_address.WalletAddress, error) {

	//avoid generating the same address twice
	if lock {
		self.Lock.Lock()
		defer self.Lock.Unlock()
	}

	version := wallet_address.VERSION_NORMAL

	if !self.Loaded {
		return nil, errors.New("Wallet was not loaded!")
	}

	secret, privateKey, spendPrivateKey, err := self.GenerateKeys(self.SeedIndex, false)
	if err != nil {
		return nil, err
	}

	privKey, err := addresses.NewPrivateKey(privateKey)
	if err != nil {
		return nil, err
	}

	spendPrivKey, err := addresses.NewPrivateKey(spendPrivateKey)
	if err != nil {
		return nil, err
	}

	if name == "" {
		name = "Addr_" + strconv.FormatUint(uint64(self.SeedIndex), 10)
	}

	addr := &wallet_address.WalletAddress{
		Version:         version,
		Name:            name,
		SecretKey:       secret,
		PrivateKey:      privKey,
		SpendPrivateKey: spendPrivKey,
		SeedIndex:       self.SeedIndex,
		IsMine:          true,
	}

	if err = self.AddAddress(addr, staked, spendRequired, false, true, false, save); err != nil {
		return nil, err
	}

	return addr.Clone(), nil
}

func (self *wallet) RemoveAddressByIndex(index int, lock bool) (bool, error) {

	if lock {
		self.Lock.Lock()
		defer self.Lock.Unlock()
	}

	if !self.Loaded {
		return false, errors.New("Wallet was not loaded!")
	}

	if index < 0 || index > len(self.Addresses) {
		return false, errors.New("Invalid Address Index")
	}

	addr := self.Addresses[index]

	removing := self.Addresses[index]

	self.Addresses = slices.Delete(self.Addresses, index, index+1) //keep order
	delete(self.addressesMap, string(addr.PublicKey))

	if index+1 == self.Count && addr.IsMine && addr.SeedIndex+1 == self.SeedIndex {
		self.SeedIndex -= 1
	}

	self.Count -= 1

	forging.Forging.Wallet.RemoveWallet(removing.PublicKey, false, nil, nil, 0)

	self.updateWallet()
	if err := self.saveWallet(index, index, self.Count, false); err != nil {
		return false, err
	}
	globals.MainEvents.BroadcastEvent("wallet/removed", addr)

	return true, nil
}

func (self *wallet) RemoveAddress(encodedAddress string, lock bool) (bool, error) {

	addr, err := addresses.DecodeAddr(encodedAddress)
	if err != nil {
		return false, err
	}

	return self.RemoveAddressByPublicKey(addr.PublicKey, lock)
}

func (self *wallet) RemoveAddressByPublicKey(publicKey []byte, lock bool) (bool, error) {

	if lock {
		self.Lock.Lock()
		defer self.Lock.Unlock()
	}

	if !self.Loaded {
		return false, errors.New("Wallet was not loaded!")
	}

	for i, addr := range self.Addresses {
		if bytes.Equal(addr.PublicKey, publicKey) {
			return self.RemoveAddressByIndex(i, false)
		}
	}

	return false, nil
}

func (self *wallet) RenameAddressByPublicKey(publicKey []byte, newName string, lock bool) (bool, error) {

	if lock {
		self.Lock.Lock()
		defer self.Lock.Unlock()
	}

	if !self.Loaded {
		return false, errors.New("Wallet was not loaded!")
	}

	addr := self.GetWalletAddressByPublicKey(publicKey, false)
	if addr == nil {
		return false, nil
	}

	addr.Name = newName

	return true, self.saveWalletAddress(addr, false)
}

func (self *wallet) GetWalletAddress(index int, lock bool) (*wallet_address.WalletAddress, error) {

	if lock {
		self.Lock.RLock()
		defer self.Lock.RUnlock()
	}

	if index < 0 || index >= len(self.Addresses) {
		return nil, errors.New("Invalid Address Index")
	}
	return self.Addresses[index].Clone(), nil
}

func (self *wallet) GetAddressSecretKey(index int) ([]byte, error) { //32 byte

	self.Lock.RLock()
	defer self.Lock.RUnlock()

	if index < 0 || index > len(self.Addresses) {
		return nil, errors.New("Invalid Address Index")
	}
	return self.Addresses[index].SecretKey, nil
}

func (self *wallet) ImportWalletAddressJSON(data []byte) (*wallet_address.WalletAddress, error) {

	addr := &wallet_address.WalletAddress{}
	if err := json.Unmarshal(data, addr); err != nil {
		return nil, errors.New("Error unmarshaling wallet")
	}

	if addr.PrivateKey == nil {
		return nil, errors.New("Private Key is missing")
	}

	self.Lock.RLock()
	defer self.Lock.RUnlock()

	isMine := false
	if self.SeedIndex != 0 {
		key, _, _, err := self.GenerateKeys(addr.SeedIndex, false)
		if err == nil && key != nil && bytes.Equal(key, addr.PrivateKey.Key) {
			isMine = true
		}
	}

	if !isMine {
		addr.SeedIndex = 0
		addr.IsImported = true
	}
	addr.IsMine = true

	if err := self.AddAddress(addr, addr.Staked, addr.SpendRequired, false, false, isMine, true); err != nil {
		return nil, err
	}

	return addr, nil
}

func (self *wallet) DecryptBalance(addr *wallet_address.WalletAddress, encryptedBalance, asset []byte, useNewPreviousValue bool, newPreviousValue uint64, store bool, ctx context.Context, statusCallback func(string)) (uint64, error) {

	if len(encryptedBalance) == 0 {
		return 0, errors.New("Encrypted Balance is nil")
	}

	return address_balance_decrypter.Decrypter.DecryptBalance("wallet", addr.PublicKey, addr.PrivateKey.Key, encryptedBalance, asset, useNewPreviousValue, newPreviousValue, store, ctx, statusCallback)
}

func (self *wallet) DecryptBalanceByPublicKey(publicKey []byte, encryptedBalance, asset []byte, useNewPreviousValue bool, newPreviousValue uint64, store, lock bool, ctx context.Context, statusCallback func(string)) (uint64, error) {

	addr := self.GetWalletAddressByPublicKey(publicKey, lock)
	if addr == nil {
		return 0, errors.New("address was not found")
	}

	return self.DecryptBalance(addr, encryptedBalance, asset, useNewPreviousValue, newPreviousValue, store, ctx, statusCallback)
}

func (self *wallet) TryDecryptBalanceByPublicKey(publicKey []byte, encryptedBalance []byte, lock bool, matchValue uint64) (bool, error) {

	if len(encryptedBalance) == 0 {
		return false, errors.New("Encrypted Balance is nil")
	}

	addr := self.GetWalletAddressByPublicKey(publicKey, lock)
	if addr == nil {
		return false, errors.New("address was not found")
	}

	return self.TryDecryptBalance(addr, encryptedBalance, matchValue)
}

func (self *wallet) TryDecryptBalance(addr *wallet_address.WalletAddress, encryptedBalance []byte, matchValue uint64) (bool, error) {
	balance, err := new(crypto.ElGamal).Deserialize(encryptedBalance)
	if err != nil {
		return false, err
	}

	return addr.PrivateKey.TryDecryptBalance(balance, matchValue), nil
}

func (self *wallet) ImportWalletJSON(data []byte) (err error) {

	wallet2 := createWalletInstance(self.updateNewChainUpdate)
	if err = json.Unmarshal(data, wallet2); err != nil {
		return errors.New("Error unmarshaling wallet")
	}

	self.Lock.Lock()
	defer self.Lock.Unlock()

	self.clearWallet()
	if err = json.Unmarshal(data, self); err != nil {
		return errors.New("Error unmarshaling wallet 2")
	}

	self.addressesMap = make(map[string]*wallet_address.WalletAddress)
	for _, adr := range self.Addresses {
		self.addressesMap[string(adr.PublicKey)] = adr
	}
	self.setLoaded(true)

	globals.MainEvents.BroadcastEvent("wallet/loaded", self.Count)

	return self.saveWalletEntire(false)
}

func (self *wallet) GetDelegatesCount() int {
	self.Lock.RLock()
	defer self.Lock.RUnlock()

	return self.DelegatesCount
}

func (self *wallet) SetNonHardening(value bool) {
	self.Lock.Lock()
	defer self.Lock.Unlock()
	self.nonHardening = value
}

func (self *wallet) ScanAddresses() error {

	self.Lock.Lock()
	defer self.Lock.Unlock()

	return store.StoreBlockchain.DB.View(func(reader store_db_interface.StoreDBTransactionInterface) (err error) {

		dataStorage := data_storage.NewDataStorage(reader)

		var reg *registration.Registration
		var plainAcc *plain_account.PlainAccount

		fails := 0
		for {

			var addr *addresses.Address
			if addr, err = self.GenerateNextAddress(false); err != nil {
				return
			}

			if reg, err = dataStorage.Regs.Get(string(addr.PublicKey)); err != nil {
				return
			}
			if plainAcc, err = dataStorage.PlainAccs.Get(string(addr.PublicKey)); err != nil {
				return
			}

			if reg != nil || plainAcc != nil {
				for i := 0; i < fails; i++ {
					if _, err = self.AddNewAddress(false, "", false, false, true); err != nil {
						return
					}
				}
				if _, err = self.AddNewAddress(false, "", reg != nil && reg.Staked, reg != nil && reg.SpendPublicKey != nil, true); err != nil {
					return
				}
				fails = 0
				continue
			}

			fails++
			if fails > 100 {
				break
			}

		}

		for i := len(self.Addresses) - 1; i > 0; i-- {
			addr := self.Addresses[i]

			if reg, err = dataStorage.Regs.Get(string(addr.PublicKey)); err != nil {
				return
			}
			if plainAcc, err = dataStorage.PlainAccs.Get(string(addr.PublicKey)); err != nil {
				return
			}
			if reg == nil && plainAcc == nil {
				if _, err = self.RemoveAddressByIndex(i, false); err != nil {
					return
				}
			}
		}

		return

	})

}

func (self *wallet) Close() {

}
