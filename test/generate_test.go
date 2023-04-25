package test

import (
	"log"
	"os"
	"testing"
)

func TestGenerateSparseFile(t *testing.T) {
	fd, err := os.OpenFile("/smb/sparsefile/test/test.txt", os.O_RDWR|os.O_CREATE|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		log.Panic("open failed", err)
	}
	for i := 0; i < 10000; i++ {
		if _, err := fd.WriteAt([]byte{97}, int64(i*10)); err != nil {
			log.Panic("write file failed", err)
		}
	}
}
