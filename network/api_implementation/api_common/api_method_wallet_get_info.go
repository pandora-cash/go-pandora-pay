package api_common

import (
	"errors"
	"net/http"
	"pandora-pay/helpers"
	"pandora-pay/wallet"
)

type APIWalletGetInfoReply struct {
	Version    wallet.Version `json:"mnemonic" msgpack:"mnemonic"`
	Encryption struct {
		Encrypted  wallet.EncryptedVersion `json:"encryptedVersion" msgpack:"encryptedVersion"`
		Salt       helpers.Base64          `json:"salt" msgpack:"salt"`
		Difficulty int                     `json:"difficulty" msgpack:"difficulty"`
	} `json:"encryption" msgpack:"encryption"`
	SeedIndex          uint32 `json:"seedIndex" msgpack:"seedIndex"`
	Count              int    `json:"count" msgpack:"count"`
	CountImportedIndex int    `json:"countImportedIndex" msgpack:"countImportedIndex"`
	Loaded             bool   `json:"loaded" msgpack:"loaded"`
	DelegatesCount     int    `json:"delegatesCount" msgpack:"delegatesCount"`
}

func (api *APICommon) GetWalletInfo(r *http.Request, args *struct{}, reply *APIWalletGetInfoReply, authenticated bool) (err error) {

	if !authenticated {
		return errors.New("Invalid User or Password")
	}

	wallet.Wallet.Lock.RLock()
	defer wallet.Wallet.Lock.RUnlock()

	reply.Version = wallet.Wallet.Version
	reply.Encryption.Encrypted = wallet.Wallet.Encryption.Encrypted
	reply.Encryption.Salt = wallet.Wallet.Encryption.Salt
	reply.Encryption.Difficulty = wallet.Wallet.Encryption.Difficulty
	reply.SeedIndex = wallet.Wallet.SeedIndex
	reply.Count = wallet.Wallet.Count
	reply.CountImportedIndex = wallet.Wallet.CountImportedIndex
	reply.Loaded = wallet.Wallet.Loaded
	reply.DelegatesCount = wallet.Wallet.DelegatesCount

	return
}
