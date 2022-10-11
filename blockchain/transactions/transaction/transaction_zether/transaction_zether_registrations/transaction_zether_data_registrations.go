package transaction_zether_registrations

import (
	"errors"
	"fmt"
	"pandora-pay/blockchain/data_storage"
	"pandora-pay/blockchain/data_storage/registrations"
	"pandora-pay/blockchain/transactions/transaction/transaction_zether/transaction_zether_registrations/transaction_zether_registration"
	"pandora-pay/config"
	"pandora-pay/cryptography/bn256"
	"pandora-pay/helpers/advanced_buffers"
)

type TransactionZetherDataRegistrations struct {
	Registrations []*transaction_zether_registration.TransactionZetherDataRegistration
}

func (self *TransactionZetherDataRegistrations) ValidateRegistrations(publickeylist []*bn256.G1) (err error) {

	if len(publickeylist) == 0 || len(publickeylist) > config.TRANSACTIONS_ZETHER_RING_MAX {
		return errors.New("Invalid PublicKeys length")
	}

	for i, reg := range self.Registrations {
		if reg != nil {
			if registrations.VerifyRegistrationPoint(publickeylist[i], reg.RegistrationStaked, reg.RegistrationSpendPublicKey, reg.RegistrationSignature) == false {
				return fmt.Errorf("Registration is invalid for %d", i)
			}
		}
	}

	return
}

func (self *TransactionZetherDataRegistrations) RegisterNow(asset []byte, dataStorage *data_storage.DataStorage, publicKeyList [][]byte) (err error) {

	for i, reg := range self.Registrations {
		if reg != nil {
			if _, err = dataStorage.CreateRegistration(publicKeyList[i], reg.RegistrationStaked, reg.RegistrationSpendPublicKey); err != nil {
				return
			}
		}
	}

	for i, reg := range self.Registrations {
		if reg != nil && reg.RegistrationType == transaction_zether_registration.NOT_REGISTERED {
			if _, _, err = dataStorage.CreateAccount(asset, publicKeyList[i], true); err != nil {
				return
			}
		} else {
			if _, _, err = dataStorage.GetOrCreateAccount(asset, publicKeyList[i], true); err != nil {
				return
			}
		}
	}

	return
}

func (self *TransactionZetherDataRegistrations) Serialize(w *advanced_buffers.BufferWriter) {

	count := uint64(0)
	for _, registration := range self.Registrations {
		if registration != nil {
			count += 1
		}
	}

	w.WriteUvarint(count)
	for i, registration := range self.Registrations {
		if registration != nil {
			w.WriteUvarint(uint64(i))
			registration.Serialize(w)
		}
	}
}

func (self *TransactionZetherDataRegistrations) Deserialize(r *advanced_buffers.BufferReader, ringSize int) (err error) {

	var count uint64
	if count, err = r.ReadUvarint(); err != nil {
		return
	}

	self.Registrations = make([]*transaction_zether_registration.TransactionZetherDataRegistration, ringSize)

	for i := uint64(0); i < count; i++ {

		var index uint64
		if index, err = r.ReadUvarint(); err != nil {
			return
		}

		if index >= uint64(ringSize) {
			return errors.New("Registration Index is invalid")
		}
		if self.Registrations[index] != nil {
			return errors.New("Registration already exists")
		}

		self.Registrations[index] = &transaction_zether_registration.TransactionZetherDataRegistration{}
		if err = self.Registrations[index].Deserialize(r); err != nil {
			return
		}
	}
	return
}
