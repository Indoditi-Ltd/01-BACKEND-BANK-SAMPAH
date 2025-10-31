package helpers

import (
	"crypto/md5"
	"encoding/hex"
	"os"
)

func MakeSignPricelist(UniqueCode string) string {
	toSign := os.Getenv("IDENTITY") + os.Getenv("APIKEY") + UniqueCode
	h := md5.New()
	h.Write([]byte(toSign))
	return hex.EncodeToString(h.Sum(nil))
}