package account

import (
	"errors"
	"pandora-pay/blockchain/data_storage/accounts/account/account_balance_homomorphic"
	"pandora-pay/cryptography/crypto"
	"pandora-pay/helpers"
)

type Account struct {
	Key     []byte                                          `json:"-" msgpack:"-"` //hashmap key
	Asset   []byte                                          `json:"-" msgpack:"-"` //collection asset
	Index   uint64                                          `json:"-" msgpack:"-"` //hashmap Index
	Version uint64                                          `json:"version" msgpack:"version"`
	Balance *account_balance_homomorphic.BalanceHomomorphic `json:"balance" msgpack:"balance"`
}

func (account *Account) IsDeletable() bool {
	return false
}

func (account *Account) SetKey(key []byte) {
	account.Key = key
}

func (account *Account) SetIndex(value uint64) {
	account.Index = value
}

func (account *Account) GetIndex() uint64 {
	return account.Index
}

func (account *Account) Validate() error {
	if account.Version != 0 {
		return errors.New("Version is invalid")
	}
	return nil
}

func (account *Account) GetBalance() (result *crypto.ElGamal) {
	return account.Balance.Amount
}

func (account *Account) Serialize(w *helpers.BufferWriter) {
	w.WriteUvarint(account.Version)
	account.Balance.Serialize(w)
}

func (account *Account) Deserialize(r *helpers.BufferReader) (err error) {

	var n uint64
	if n, err = r.ReadUvarint(); err != nil {
		return
	}
	if n != 0 {
		return errors.New("Invalid Account Version")
	}

	account.Version = n
	if err = account.Balance.Deserialize(r); err != nil {
		return
	}

	return
}

func NewAccount(key []byte, index uint64, asset []byte) (*Account, error) {

	balance, err := account_balance_homomorphic.NewBalanceHomomorphicEmptyBalance(key)
	if err != nil {
		return nil, err
	}

	return &Account{
		key,
		asset,
		index,
		0,
		balance,
	}, nil
}

func NewAccountClear(key []byte, index uint64, asset []byte) *Account {
	return &Account{
		key,
		asset,
		index,
		0,
		&account_balance_homomorphic.BalanceHomomorphic{nil},
	}
}
