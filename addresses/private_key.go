package addresses

import (
	"crypto/ed25519"
	"errors"
	"pandora-pay/config"
	"pandora-pay/cryptography"
	"pandora-pay/helpers"
)

type PrivateKey struct {
	Key []byte `json:"key" msgpack:"key"` //32 byte
}

func (pk *PrivateKey) GeneratePublicKey() []byte {
	return pk.Key[32:]
}

func (pk *PrivateKey) GenerateAddress(paymentID []byte, paymentAmount uint64, paymentAsset []byte) (*Address, error) {
	publicKey := pk.GeneratePublicKey()

	version := SIMPLE_PUBLIC_KEY

	return NewAddr(config.NETWORK_SELECTED, version, publicKey, paymentID, paymentAmount, paymentAsset)
}

//make sure message is a hash to avoid leaking any parts of the private key
func (pk *PrivateKey) Sign(message []byte) ([]byte, error) {
	return ed25519.Sign(pk.Key, message), nil
}

func (pk *PrivateKey) Decrypt(message []byte) ([]byte, error) {
	return nil, errors.New("Encryption is not supported right now")
}

func GenerateNewPrivateKey() *PrivateKey {
	privateKey := helpers.RandomBytes(cryptography.PrivateKeySize)
	return &PrivateKey{Key: privateKey}
}

func CreatePrivateKeyFromSeed(key []byte) (*PrivateKey, error) {
	if len(key) != cryptography.PrivateKeySize {
		return nil, errors.New("Private key length is invalid")
	}
	return &PrivateKey{Key: key}, nil
}
