package block

import (
	"github.com/stretchr/testify/assert"
	"pandora-pay/addresses"
	"pandora-pay/cryptography"
	"pandora-pay/helpers"
	"pandora-pay/helpers/advanced_buffers"
	"testing"
	"time"
)

var (
	merkleHash     = cryptography.SHA3([]byte("MerkleHash"))
	prevHash       = cryptography.SHA3([]byte("PrevHash"))
	prevKernelHash = cryptography.SHA3([]byte("PrevKernelHash"))
)

func TestBlock_Serialize(t *testing.T) {
	var err error

	privateKey := addresses.GenerateNewPrivateKey()
	publicKey := privateKey.GeneratePublicKey()

	blk := Block{
		BlockHeader:    &BlockHeader{Version: 0, Height: 0},
		MerkleHash:     merkleHash,
		PrevHash:       prevHash,
		PrevKernelHash: prevKernelHash,
		Timestamp:      uint64(time.Now().Unix()),
	}

	buf := blk.SerializeManualToBytes()
	assert.Equal(t, len(buf) < 30, false, "Invalid serialization")

	blk2 := &Block{BlockHeader: &BlockHeader{}}

	r := advanced_buffers.NewBufferReader(buf)
	err = blk2.Deserialize(r)
	assert.NoError(t, err, "Error...?")

	assert.Equal(t, blk2.SerializeManualToBytes(), blk.SerializeManualToBytes(), "Serialization/Deserialization doesn't work")

}

func TestBlock_SerializeForSigning(t *testing.T) {

	var err error

	privateKey := addresses.GenerateNewPrivateKey()
	publicKey := privateKey.GeneratePublicKey()

	blockHeader := &BlockHeader{Version: 0, Height: 0}
	blk := Block{
		BlockHeader:    blockHeader,
		MerkleHash:     merkleHash,
		PrevHash:       prevHash,
		PrevKernelHash: prevKernelHash,
		Timestamp:      uint64(time.Now().Unix()),
	}

	hash := blk.SerializeForSigning()

	var signature []byte

	signature, err = privateKey.Sign(hash)
	assert.NoError(t, err, "Signing raised an error")

	assert.NotEqual(t, signature, helpers.EmptyBytes(cryptography.SignatureSize), "Invalid signature")
}
