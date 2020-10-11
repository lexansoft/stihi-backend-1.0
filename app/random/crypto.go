package random

import(
	"crypto/rand"
	"encoding/base64"
)

func GetBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}

	return b, nil
}

func GetString(s int) (string, error) {
	b, err := GetBytes(s)
	return base64.URLEncoding.EncodeToString(b), err
}
