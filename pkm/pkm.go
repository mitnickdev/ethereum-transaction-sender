package pkm

import (
	"crypto/ecdsa"
	"fmt"
	"git.ddex.io/infrastructure/ethereum-launcher/config"
	"github.com/HydroProtocol/hydro-sdk-backend/sdk/crypto"
	"github.com/HydroProtocol/hydro-sdk-backend/sdk/signer"
	"github.com/HydroProtocol/hydro-sdk-backend/sdk/types"
	"github.com/HydroProtocol/hydro-sdk-backend/utils"
	"github.com/sirupsen/logrus"
	"strings"
)

type Pkm interface {
	Sign(from string, t*types.Transaction) (string, error)
}

type localPKM struct {
	KeyMap map[string]*ecdsa.PrivateKey
}

var LocalPKM *localPKM

func init() {
	privateKeys := config.Config.PrivateKeys
	privateKeyList := strings.Split(privateKeys, ",")

	keyMap := make(map[string]*ecdsa.PrivateKey)
	for idx, privateKeyHex := range privateKeyList {
		privateKey, err := crypto.NewPrivateKeyByHex(privateKeyHex)
		if err != nil {
			logrus.Errorf("parse private key fail, key is num %d", idx)
			continue
		}

		publicKey := crypto.PubKey2Address(privateKey.PublicKey)
		keyMap[publicKey] = privateKey
		logrus.Infof("parse private key success, public key is %s", publicKey)
	}
	LocalPKM = &localPKM{
		KeyMap: keyMap,
	}
}

func (l localPKM) Sign(from string, t *types.Transaction) (string, error) {
	privateKey, ok := l.KeyMap[from]
	if !ok {
		return "", fmt.Errorf("cannot sign by account %s", from)
	}

	signedTransaction, err := signer.SignTx(t, privateKey)

	if err != nil {
		utils.Errorf("sign transaction error: %v", err)
		panic(err)
	}

	return utils.Bytes2Hex(signedTransaction.Signature), nil
}