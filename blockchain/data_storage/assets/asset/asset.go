package asset

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"pandora-pay/config/config_assets"
	"pandora-pay/config/config_coins"
	"pandora-pay/cryptography"
	"pandora-pay/helpers"
	"pandora-pay/helpers/advanced_buffers"
	"regexp"
	"strings"
)

var regexAssetName = regexp.MustCompile("^([a-zA-Z0-9]+ )+[a-zA-Z0-9]+$|^[a-zA-Z0-9]+")
var regexAssetTicker = regexp.MustCompile("^[A-Z0-9]+$") // only lowercase ascii is allowed. No space allowed
var regexAssetDescription = regexp.MustCompile("[\\w|\\W]+")

type Asset struct {
	PublicKeyHash            []byte `json:"-" msgpack:"-"` //hashmap key
	Index                    uint64 `json:"-" msgpack:"-"` //hashMap index
	Version                  uint64 `json:"version,omitempty" msgpack:"version,omitempty"`
	CanUpgrade               bool   `json:"canUpgrade,omitempty" msgpack:"canUpgrade,omitempty"`                             //upgrade different setting s
	CanMint                  bool   `json:"canMint,omitempty" msgpack:"canMint,omitempty"`                                   //increase supply
	CanBurn                  bool   `json:"canBurn,omitempty" msgpack:"canBurn,omitempty"`                                   //decrease supply
	CanChangeUpdatePublicKey bool   `json:"canChangeUpdatePublicKey,omitempty" msgpack:"canChangeUpdatePublicKey,omitempty"` //can change key
	CanChangeSupplyPublicKey bool   `json:"canChangeSupplyPublicKey,omitempty" msgpack:"canChangeSupplyPublicKey,omitempty"` //can change supply key
	CanPause                 bool   `json:"canPause,omitempty" msgpack:"canPause,omitempty"`                                 //can pause (suspend transactions)
	CanFreeze                bool   `json:"canFreeze,omitempty" msgpack:"canFreeze,omitempty"`                               //freeze supply changes
	DecimalSeparator         byte   `json:"decimalSeparator,omitempty" msgpack:"decimalSeparator,omitempty"`
	MaxSupply                uint64 `json:"maxSupply,omitempty" msgpack:"maxSupply,omitempty"`
	Supply                   uint64 `json:"supply,omitempty" msgpack:"supply,omitempty"`
	UpdatePublicKey          []byte `json:"updatePublicKey,omitempty" msgpack:"updatePublicKey,omitempty"` //33 byte
	SupplyPublicKey          []byte `json:"supplyPublicKey,omitempty" msgpack:"supplyPublicKey,omitempty"` //33 byte
	Name                     string `json:"name" msgpack:"name"`
	Ticker                   string `json:"ticker" msgpack:"ticker"`
	Identification           string `json:"identification" msgpack:"identification"`
	Description              string `json:"description,omitempty" msgpack:"description,omitempty"`
	Data                     []byte `json:"data,omitempty" msgpack:"data,omitempty"`
}

func (asset *Asset) IsDeletable() bool {
	return false
}

func (asset *Asset) SetKey(key []byte) {
	if !bytes.Equal(key, asset.PublicKeyHash) {
		asset.PublicKeyHash = key
		asset.setIdentification()
	}
}

func (asset *Asset) SetIndex(value uint64) {
	asset.Index = value
}

func (asset *Asset) GetIndex() uint64 {
	return asset.Index
}

func (asset *Asset) Validate() error {
	if asset.DecimalSeparator > config_assets.ASSETS_DECIMAL_SEPARATOR_MAX_BYTE {
		return errors.New("asset decimal separator is invalid")
	}

	if len(asset.Name) > 15 || len(asset.Name) < 3 {
		return errors.New("asset name length is invalid")
	}
	if len(asset.Ticker) > 10 || len(asset.Ticker) < 2 {
		return errors.New("asset ticker length is invalid")
	}
	if len(asset.Description) > 1024 {
		return errors.New("asset description length is invalid")
	}
	if len(asset.Data) > 5120 {
		return errors.New("asset data length is invalid")
	}

	if !regexAssetName.MatchString(asset.Name) {
		return errors.New("Asset name is invalid")
	}
	if !regexAssetTicker.MatchString(asset.Ticker) {
		return errors.New("Asset ticker is invalid")
	}
	if !regexAssetDescription.MatchString(asset.Description) {
		return errors.New("Asset description is invalid")
	}

	if len(asset.PublicKeyHash) != cryptography.PublicKeyHashSize {
		return errors.New("Asset Public key is invalid")
	}

	if !bytes.Equal(asset.PublicKeyHash, config_coins.NATIVE_ASSET_FULL) {

		if strings.ToUpper(asset.Name) == config_coins.NATIVE_ASSET_NAME {
			return errors.New("Asset can not contain same name")
		}
		if asset.Ticker == config_coins.NATIVE_ASSET_TICKER {
			return errors.New("Asset can not contain same ticker")
		}

		identification := asset.Ticker + "-" + hex.EncodeToString(asset.PublicKeyHash[:3])
		if asset.Identification != identification {
			return fmt.Errorf("Asset identification is not matching %s != %s", asset.Identification, identification)
		}

	} else {
		if asset.Identification != config_coins.NATIVE_ASSET_IDENTIFICATION {
			return errors.New("Asset native identification is not matching")
		}
		if asset.Ticker != config_coins.NATIVE_ASSET_TICKER {
			return errors.New("Asset ticker is not matching")
		}

	}

	return nil
}

func (asset *Asset) ConvertToUnits(amount float64) (uint64, error) {
	COIN_DENOMINATION := math.Pow10(int(asset.DecimalSeparator))
	if amount < float64(math.MaxUint64)/COIN_DENOMINATION {
		return uint64(amount * COIN_DENOMINATION), nil
	}
	return 0, errors.New("Error converting to units")
}

func (asset *Asset) ConvertToBase(amount uint64) float64 {
	COIN_DENOMINATION := math.Pow10(int(asset.DecimalSeparator))
	return float64(amount) / COIN_DENOMINATION
}

func (asset *Asset) AddNativeSupply(sign bool, amount uint64) error {
	if sign {
		if asset.MaxSupply-asset.Supply < amount {
			return errors.New("Supply exceeded max supply")
		}
		return helpers.SafeUint64Add(&asset.Supply, amount)
	}

	if asset.Supply < amount {
		return errors.New("Supply would become negative")
	}
	return helpers.SafeUint64Sub(&asset.Supply, amount)
}

func (asset *Asset) AddSupply(sign bool, amount uint64) error {

	if bytes.Equal(asset.SupplyPublicKey, config_coins.BURN_PUBLIC_KEY) {
		return errors.New("BURN PUBLIC KEY")
	}

	if sign {
		if !asset.CanMint {
			return errors.New("Can't mint")
		}
		if asset.MaxSupply-asset.Supply < amount {
			return errors.New("Supply exceeded max supply")
		}
		return helpers.SafeUint64Add(&asset.Supply, amount)
	}

	if !asset.CanBurn {
		return errors.New("Can't burn")
	}
	if asset.Supply < amount {
		return errors.New("Supply would become negative")
	}
	return helpers.SafeUint64Sub(&asset.Supply, amount)
}

func (asset *Asset) Serialize(w *advanced_buffers.BufferWriter) {

	w.WriteUvarint(asset.Version)

	w.WriteBool(asset.CanUpgrade)
	w.WriteBool(asset.CanMint)
	w.WriteBool(asset.CanBurn)
	w.WriteBool(asset.CanChangeUpdatePublicKey)
	w.WriteBool(asset.CanChangeSupplyPublicKey)
	w.WriteBool(asset.CanPause)
	w.WriteBool(asset.CanFreeze)
	w.WriteByte(asset.DecimalSeparator)

	w.WriteUvarint(asset.MaxSupply)
	w.WriteUvarint(asset.Supply)

	w.Write(asset.UpdatePublicKey)
	w.Write(asset.SupplyPublicKey)

	w.WriteString(asset.Name)
	w.WriteString(asset.Ticker)
	w.WriteString(asset.Description)
	w.WriteVariableBytes(asset.Data)
}

func (asset *Asset) setIdentification() {
	if bytes.Equal(asset.PublicKeyHash, config_coins.NATIVE_ASSET_FULL) {
		asset.Identification = config_coins.NATIVE_ASSET_IDENTIFICATION
	} else {
		asset.Identification = asset.Ticker + "-" + hex.EncodeToString(asset.PublicKeyHash[:3])
	}
}

func (asset *Asset) Deserialize(r *advanced_buffers.BufferReader) (err error) {

	if asset.Version, err = r.ReadUvarint(); err != nil {
		return
	}
	if asset.CanUpgrade, err = r.ReadBool(); err != nil {
		return
	}
	if asset.CanMint, err = r.ReadBool(); err != nil {
		return
	}
	if asset.CanBurn, err = r.ReadBool(); err != nil {
		return
	}
	if asset.CanChangeUpdatePublicKey, err = r.ReadBool(); err != nil {
		return
	}
	if asset.CanChangeSupplyPublicKey, err = r.ReadBool(); err != nil {
		return
	}
	if asset.CanPause, err = r.ReadBool(); err != nil {
		return
	}
	if asset.CanFreeze, err = r.ReadBool(); err != nil {
		return
	}
	if asset.DecimalSeparator, err = r.ReadByte(); err != nil {
		return
	}
	if asset.MaxSupply, err = r.ReadUvarint(); err != nil {
		return
	}
	if asset.Supply, err = r.ReadUvarint(); err != nil {
		return
	}
	if asset.UpdatePublicKey, err = r.ReadBytes(cryptography.PublicKeySize); err != nil {
		return
	}
	if asset.SupplyPublicKey, err = r.ReadBytes(cryptography.PublicKeySize); err != nil {
		return
	}
	if asset.Name, err = r.ReadString(15); err != nil {
		return
	}
	if asset.Ticker, err = r.ReadString(10); err != nil {
		return
	}
	if asset.Description, err = r.ReadString(1024); err != nil {
		return
	}
	if asset.Data, err = r.ReadVariableBytes(5120); err != nil {
		return
	}

	asset.setIdentification()

	return
}

func NewAsset(publicKeyHash []byte, index uint64) *Asset {
	return &Asset{
		PublicKeyHash: publicKeyHash,
		Index:         index,
	}
}
