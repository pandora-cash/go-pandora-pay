package api_common

import (
	"errors"
	"net/http"
	"pandora-pay/wallet"
)

type APIWalletEncryptionRemoveReply struct {
	Result bool `json:"result" msgpack:"mnemonic"`
}

func (api *APICommon) EncryptionWalletRemove(r *http.Request, args *struct{}, reply *APIWalletEncryptionRemoveReply, authenticated bool) (err error) {

	if !authenticated {
		return errors.New("Invalid User or Password")
	}

	if err = wallet.Wallet.Encryption.RemoveEncryption(); err != nil {
		return
	}
	reply.Result = true

	return
}
