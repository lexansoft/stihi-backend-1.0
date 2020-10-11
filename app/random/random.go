package random

import (
	"math/rand"
	"sync"
	"time"
)

const (
	CharsetAl 		= "abcdefghijklmnopqrstuvwxyz"
	CharsetAlLen 	= 26
	CharsetAlD 		= "abcdefghijklmnopqrstuvwxyz0123456789"
	CharsetAlDLen 	= 36
	CharsetAu 		= "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	CharsetAuLen 	= 26
	CharsetA 		= "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	CharsetALen 	= 52
	CharsetAD 		= "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	CharsetADLen	= 62
	CharsetADS 		= "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789@$#%^&*()-+{}\\/.,;:`'"
	CharsetADSLen	= 82
)

var (
	seededRand *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	randMutex *sync.Mutex = &sync.Mutex{}
)

func StringWithCharset(length int, charset string, charsetLen int) string {
	b := make([]byte, length)
	randMutex.Lock()
	for i := range b {
		idx := int(seededRand.Float32()*float32(charsetLen))
		b[i] = charset[idx]
	}
	randMutex.Unlock()
	return string(b)
}

func String(length int) string {
	return StringWithCharset(length, CharsetAD, CharsetADLen)
}
