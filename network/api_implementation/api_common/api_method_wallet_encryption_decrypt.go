package api_common

import (
	"errors"
	"net/http"
	"pandora-pay/wallet"
)

type APIWalletEncryptionDecryptRequest struct {
	Password string `json:"password" msgpack:"password"`
}

type APIWalletEncryptionDecryptReply struct {
	Result bool `json:"result" msgpack:"mnemonic"`
}

func (api *APICommon) EncryptionWalletDecrypt(r *http.Request, args *APIWalletEncryptionDecryptRequest, reply *APIWalletEncryptionDecryptReply, authenticated bool) (err error) {

	if !authenticated {
		return errors.New("Invalid User or Password")
	}

	if err = wallet.Wallet.Encryption.Decrypt(args.Password); err != nil {
		return
	}
	reply.Result = true

	return
}
