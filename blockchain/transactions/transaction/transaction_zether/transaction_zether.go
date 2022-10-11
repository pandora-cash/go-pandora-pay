package transaction_zether

import (
	"bytes"
	"errors"
	"fmt"
	"pandora-pay/blockchain/data_storage"
	"pandora-pay/blockchain/genesis"
	"pandora-pay/blockchain/transactions/transaction/transaction_base_interface"
	"pandora-pay/blockchain/transactions/transaction/transaction_zether/transaction_zether_payload"
	"pandora-pay/config/config_coins"
	"pandora-pay/cryptography"
	"pandora-pay/helpers"
	"pandora-pay/helpers/advanced_buffers"
	"strconv"
)

type TransactionZether struct {
	transaction_base_interface.TransactionBaseInterface
	ChainHeight     uint64
	ChainKernelHash []byte
	Payloads        []*transaction_zether_payload.TransactionZetherPayload
	Bloom           *TransactionZetherBloom
}

/**
Zether requires another verification that the bloomed publicKeys, CL, CR are the same
*/
func (tx *TransactionZether) IncludeTransaction(blockHeight uint64, txHash []byte, dataStorage *data_storage.DataStorage) (err error) {

	if tx.ChainHeight > blockHeight {
		return fmt.Errorf("Zether ChainHeight is invalid %d > %d", tx.ChainHeight, blockHeight)
	}

	var chainKernelHash []byte
	if blockHeight > 0 {
		chainKernelHash = dataStorage.DBTx.Get("blockKernelHash_ByHeight" + strconv.FormatUint(tx.ChainHeight, 10))
	} else {
		chainKernelHash = genesis.Genesis.PrevKernelHash
	}

	if !bytes.Equal(chainKernelHash, tx.ChainKernelHash) {
		return errors.New("Zether ChainKernelHash is invalid")
	}

	for payloadIndex, payload := range tx.Payloads {
		if err = payload.IncludePayload(txHash, byte(payloadIndex), tx.Bloom.PublicKeyLists[payloadIndex], blockHeight, dataStorage); err != nil {
			return
		}
	}

	return
}

func (tx *TransactionZether) ComputeFee() (uint64, error) {

	sum := uint64(0)
	for _, payload := range tx.Payloads {
		if bytes.Equal(payload.Asset, config_coins.NATIVE_ASSET_FULL) {
			if err := helpers.SafeUint64Add(&sum, payload.Statement.Fee); err != nil {
				return 0, err
			}
		} else {
			fee := payload.Statement.Fee
			if err := helpers.SafeUint64Mul(&fee, payload.FeeRate); err != nil {
				return 0, err
			}
			fee = fee / helpers.Pow10(payload.FeeLeadingZeros)
			if err := helpers.SafeUint64Add(&sum, fee); err != nil {
				return 0, err
			}
		}
	}

	return sum, nil
}

func (tx *TransactionZether) ComputeAllKeys(out map[string]bool) {

	for payloadIndex, payload := range tx.Payloads {
		payload.ComputeAllKeys(out, tx.Bloom.PublicKeyLists[payloadIndex])
	}

	return
}

func (tx *TransactionZether) Validate() (err error) {

	if len(tx.Payloads) == 0 {
		return errors.New("You need at least one payload")
	}

	for payloadIndex, payload := range tx.Payloads {
		if err = payload.Validate(byte(payloadIndex)); err != nil {
			return
		}
	}

	return
}

func (tx *TransactionZether) VerifySignatureManually(txHash []byte) bool {

	assetMap := map[string]int{}
	for _, payload := range tx.Payloads {
		if payload.Proof.Verify(payload.Asset, assetMap[string(payload.Asset)], tx.ChainKernelHash, payload.Statement, txHash, payload.BurnValue) == false {
			return false
		}
		assetMap[string(payload.Asset)] = assetMap[string(payload.Asset)] + 1
	}

	return true
}

func (tx *TransactionZether) SerializeAdvanced(w *advanced_buffers.BufferWriter, inclSignature bool) {
	w.WriteUvarint(tx.ChainHeight)
	w.Write(tx.ChainKernelHash)

	w.WriteByte(byte(len(tx.Payloads)))
	for _, payload := range tx.Payloads {
		payload.Serialize(w, inclSignature)
	}

}

func (tx *TransactionZether) Serialize(w *advanced_buffers.BufferWriter) {
	tx.SerializeAdvanced(w, true)
}

func (tx *TransactionZether) Deserialize(r *advanced_buffers.BufferReader) (err error) {

	if tx.ChainHeight, err = r.ReadUvarint(); err != nil {
		return
	}

	if tx.ChainKernelHash, err = r.ReadBytes(cryptography.HashSize); err != nil {
		return
	}

	var n byte
	if n, err = r.ReadByte(); err != nil {
		return
	}

	tx.Payloads = make([]*transaction_zether_payload.TransactionZetherPayload, n)
	for i := byte(0); i < n; i++ {
		payload := &transaction_zether_payload.TransactionZetherPayload{}
		if err = payload.Deserialize(r); err != nil {
			return
		}
		tx.Payloads[i] = payload
	}

	return
}

func (tx *TransactionZether) VerifyBloomAll() error {
	if tx.Bloom == nil {
		return errors.New("Tx was not bloomed")
	}
	return tx.Bloom.verifyIfBloomed()
}

func (tx *TransactionZether) GetBloomExtra() any {
	return tx.Bloom
}

func (tx *TransactionZether) SetBloomExtra(bloom any) {
	if tx.Bloom == nil {
		tx.Bloom = bloom.(*TransactionZetherBloom)
	}
}
