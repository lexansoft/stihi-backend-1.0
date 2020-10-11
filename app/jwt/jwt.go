package jwt

import (
	"crypto/rsa"
	"github.com/pkg/errors"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/SermoDigital/jose/crypto"
	"github.com/SermoDigital/jose/jws"
	"gitlab.com/stihi/stihi-backend/app/config"
		"gitlab.com/stihi/stihi-backend/app/random"
	"crypto/rand"
	"github.com/btcsuite/btcutil/base58"
	"gitlab.com/stihi/stihi-backend/cache_level1"
)

var (
	RSAPublicKey *rsa.PublicKey
	RSAPrivateKey *rsa.PrivateKey
	Config	*config.JWTConfig
)

func Init(config *config.JWTConfig) error {
	Config = config
	bytes, err := ioutil.ReadFile(config.PrivateKeyPath)
	if err != nil {
		return errors.New("Error read private key file '"+config.PrivateKeyPath+"': "+err.Error())
	}

	RSAPrivateKey, err = crypto.ParseRSAPrivateKeyFromPEM(bytes)
	if err != nil {
		return errors.New("Error parse private key file '"+config.PrivateKeyPath+"': "+err.Error())
	}

	bytes, err = ioutil.ReadFile(config.PublicKeyPath)
	if err != nil {
		return errors.New("Error read public key file '"+config.PublicKeyPath+"': "+err.Error())
	}

	RSAPublicKey, err = crypto.ParseRSAPublicKeyFromPEM(bytes)
	if err != nil {
		return errors.New("Error parse public key file '"+config.PublicKeyPath+"': "+err.Error())
	}

	return nil
}

// TODO: При каждой выдаче токена на сервере (в Redis) сохраняется время его обновления (время истечения токена).
// Если при обновлении токена время обновления на сервере NULL или равно время истечения токена - токен обновляется
// (отправляется вместе с данными). Если время обновления токена на сервере не равно времени истечения токена,
// тогда требуется полная авторизация.

func New(userId int64, userName string, role string, privateKey, keyType string) ([]byte, *jws.JWS, error) {
	expiredAt := (time.Now().Add(time.Duration(Config.Expire) * time.Minute)).Unix()
	jwtId, err := getJWTId()
	payload := jws.Claims{
		"iss": Config.Issuer, // Имя создателя токена
		"sub": userId,        // ID юзера
		"n":   userName,      // Имя юзера
		"r":   role,          // Роль юзера
		"kp":  privateKey,    // Зашифрованный приватный posting ключ пользователя
		"kpt": EncodeKeyType(keyType),
		"exp": expiredAt,
		"jti": jwtId,
	}

	token := jws.New(payload, crypto.SigningMethodRS512)

	jwtBytes, err := token.Compact(RSAPrivateKey)
	if err != nil {
		return nil, nil, err
	}

	cache_level1.DB.RedisConn.Set("jwt_expired_"+jwtId, strconv.FormatInt(expiredAt, 10))

	return jwtBytes, &token, nil
}

func EncodeKeyType(keyType string) string {
	switch keyType {
	case "posting":
		return "p"
	case "active":
		return "a"
	case "owner":
		return "o"
	case "memo":
		return "m"
	}
	return keyType
}

func DecodeKeyType(kt string) string {
	switch kt {
	case "p":
		return "posting"
	case "a":
		return "active"
	case "o":
		return "owner"
	case "m":
		return "memo"
	}
	return kt
}

func Check(r *http.Request) (*map[string]interface{}, *jws.JWS, error) {
	jwToken, err := jws.ParseFromHeader(r, jws.Compact)
	if err != nil {
		return nil, nil, err
	}

	err = jwToken.Verify(RSAPublicKey, crypto.SigningMethodRS512)
	if err != nil {
		return nil, nil, err
	}

	claims, err := parsePayload(&jwToken)
	if err != nil {
		return nil, &jwToken, err
	}
	return claims, &jwToken, nil
}

// Если нет необходимости в обновлении - возвращаем везде nil
// Если невозможно обновить токен - возвращаем err
func Refresh(token *jws.JWS) ([]byte, *jws.JWS, error) {
	claims, err := parsePayload(token)
	if err != nil {
		return nil, nil, err
	}

	// TODO: Сделать алгоритм обновления JWT - учитывать: обновление при истечении времени, обновление по флагу в redis (обновление при изменении данных пользователя)

	// Сначала проверяем что токен надо обновлять по его сроку
	jwtExpired, ok := (*claims)["exp"].(int64)
	if !ok {
		err = errors.New("jwt expired time not found")
		return nil, nil, err
	}

	if jwtExpired - time.Now().Add(time.Duration(Config.RenewTime) * time.Minute).Unix() > 0 {
		// Не пришло время - возвращаем nil
		return nil, nil, err
	}

	jwtBytes, token, err := New(
		(*claims)["sub"].(int64),
		(*claims)["n"].(string),
		(*claims)["r"].(string),
		(*claims)["kp"].(string),
		(*claims)["kpt"].(string),
	)
	if err != nil {
		return nil, nil, err
	}

	return jwtBytes, token, nil
}

func getJWTId() (string, error) {
	id, err := random.GetString(32)
	if err != nil {
		return "", err
	}

	return id, nil
}

func parsePayload(token *jws.JWS) (*map[string]interface{}, error) {
	payload := (*token).Payload()

	switch payload.(type) {
	case map[string]interface{}:
		claims := payload.(map[string]interface{})
		return &claims, nil
	default:
		err := errors.New("unknown payload for JWT")
		return nil, err
	}
}

//
func EncryptPrivKey(privKey string) (string, error) {
	key := base58.Decode(privKey)

	rng := rand.Reader
	enc, err := rsa.EncryptPKCS1v15(rng, RSAPublicKey, key)
	if err != nil {
		return "", err
	}
	encKey := base58.Encode(enc)

	return encKey, nil
}

func DecryptPrivKey(encKey string) (string, error) {
	enc := base58.Decode(encKey)

	rng := rand.Reader
	dec, err := rsa.DecryptPKCS1v15(rng, RSAPrivateKey, enc)
	if err != nil {
		return "", err
	}
	key := base58.Encode(dec)

	return key, nil
}
