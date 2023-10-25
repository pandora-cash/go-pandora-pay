package api_common

import (
	"errors"
	"net/http"
	"pandora-pay/wallet"
)

type APIWalletGetMnemonicReply struct {
	Mnemonic string `json:"mnemonic" msgpack:"mnemonic"`
}

func (api *APICommon) GetWalletMnemonic(r *http.Request, args *struct{}, reply *APIWalletGetMnemonicReply, authenticated bool) (err error) {

	if !authenticated {
		return errors.New("Invalid User or Password")
	}

	wallet.Wallet.Lock.RLock()
	defer wallet.Wallet.Lock.RUnlock()

	reply.Mnemonic = wallet.Wallet.Mnemonic

	return
}
