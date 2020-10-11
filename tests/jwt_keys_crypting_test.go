package tests

import (
	"testing"
	"gitlab.com/stihi/stihi-backend/app/config"
	"gitlab.com/stihi/stihi-backend/app/jwt"
	"gitlab.com/stihi/stihi-backend/app/random"
	"github.com/stretchr/testify/assert"
	"github.com/btcsuite/btcutil/base58"
	"log"
)

func init() {
	cfg := config.JWTConfig{
		PrivateKeyPath: "../configs/priv_key_test.pem",
		PublicKeyPath: "../configs/pub_key_test.pem",
	}

	err := jwt.Init(&cfg)
	if err != nil {
		log.Fatalln(err)
	}
}

func TestEncryptDecryptPrivKeyForJWT(t *testing.T) {
	originalBin, err := random.GetBytes(36)
	assert.Nil(t, err)

	// Encript
	originalBase58 := base58.Encode(originalBin)
	encBase58, err := jwt.EncryptPrivKey(originalBase58)
	assert.Nil(t, err)

	// Decript
	decBase58, err := jwt.DecryptPrivKey(encBase58)
	assert.Nil(t, err)

	decBin := base58.Decode(decBase58)

	assert.Equal(t, originalBin, decBin, "Encrypt/decrypt for private key is equals")
}