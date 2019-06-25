package utils

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"math/rand"
	"time"
)

//Ut utils
type Ut struct {
	r *rand.Rand
}

//NewUtils utils
func NewUtils() *Ut {
	return &Ut{
		r: rand.New(rand.NewSource(time.Now().Unix())),
	}
}

//Md5 md5
func (u *Ut) Md5(str string) string {
	h := md5.New()
	h.Write([]byte(str))
	return hex.EncodeToString(h.Sum(nil))
}

//Base64encode  encode
func (u *Ut) Base64encode(input []byte) string {
	return base64.StdEncoding.EncodeToString(input)
}

//Base64decode decode
func (u *Ut) Base64decode(encodeString string) []byte {
	if decodeBytes, err := base64.StdEncoding.DecodeString(encodeString); err == nil {
		return decodeBytes
	}
	return nil
}

//GetNonce get
func (u *Ut) GetNonce() string {
	bytes := make([]byte, 6)
	for i := 0; i < 6; i++ {
		b := u.r.Intn(78) + 48
		bytes[i] = byte(b)
	}
	return string(bytes)
}

//GetPasswd get
func (u *Ut) GetPasswd() []byte {
	bytes := make([]byte, 12)
	for i := 0; i < 12; i++ {
		b := u.r.Intn(78) + 48
		bytes[i] = byte(b)
	}
	return bytes
}
