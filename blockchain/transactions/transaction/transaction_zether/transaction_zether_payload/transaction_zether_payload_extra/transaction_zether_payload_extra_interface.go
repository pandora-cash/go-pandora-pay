package transaction_zether_payload_extra

import (
	"pandora-pay/blockchain/data_storage"
	"pandora-pay/blockchain/transactions/transaction/transaction_zether/transaction_zether_registrations"
	"pandora-pay/cryptography/crypto"
	"pandora-pay/helpers/advanced_buffers"
)

type TransactionZetherPayloadExtraInterface interface {
	BeforeIncludeTxPayload(txHash []byte, payloadRegistrations *transaction_zether_registrations.TransactionZetherDataRegistrations, payloadIndex byte, payloadAsset []byte, payloadBurnValue uint64, payloadStatement *crypto.Statement, publicKeyList [][]byte, blockHeight uint64, dataStorage *data_storage.DataStorage) error
	AfterIncludeTxPayload(txHash []byte, payloadRegistrations *transaction_zether_registrations.TransactionZetherDataRegistrations, payloadIndex byte, payloadAsset []byte, payloadBurnValue uint64, payloadStatement *crypto.Statement, publicKeyList [][]byte, blockHeight uint64, dataStorage *data_storage.DataStorage) error
	Validate(payloadRegistrations *transaction_zether_registrations.TransactionZetherDataRegistrations, payloadIndex byte, payloadAsset []byte, payloadBurnValue uint64, payloadStatement *crypto.Statement, payloadParity bool) error
	Serialize(w *advanced_buffers.BufferWriter, inclSignature bool)
	Deserialize(r *advanced_buffers.BufferReader) error
	VerifyExtraSignature(hashForSignature []byte, payloadStatement *crypto.Statement) bool
	ComputeAllKeys(out map[string]bool)
	UpdateStatement(payloadStatement *crypto.Statement) error
}

func SerializeToBytes(self TransactionZetherPayloadExtraInterface, inclSignature bool) []byte {
	w := advanced_buffers.NewBufferWriter()
	self.Serialize(w, inclSignature)
	return w.Bytes()
}
