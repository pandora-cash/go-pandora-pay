package api_common

import (
	"errors"
	"net/http"
	"pandora-pay/wallet"
)

type APIWalletEncryptionEncryptRequest struct {
	Password   string `json:"password" msgpack:"password"`
	Difficulty int    `json:"difficulty" msgpack:"difficulty"`
}

type APIWalletEncryptionEncryptReply struct {
	Result bool `json:"result" msgpack:"mnemonic"`
}

func (api *APICommon) EncryptionWalletEncrypt(r *http.Request, args *APIWalletEncryptionEncryptRequest, reply *APIWalletEncryptionEncryptReply, authenticated bool) (err error) {

	if !authenticated {
		return errors.New("Invalid User or Password")
	}

	if err = wallet.Wallet.Encryption.Encrypt(args.Password, args.Difficulty); err != nil {
		return
	}
	reply.Result = true

	return
}
