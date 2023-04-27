package test

import (
	"errors"
	"golang.org/x/sys/unix"
	"io"
	"io/fs"
	"log"
	"os"
	"syscall"
	"testing"
)

func GetSize(srcFs *os.File) (int64, error) {
	old, _ := srcFs.Seek(0, io.SeekCurrent)
	_, _ = srcFs.Seek(0, io.SeekStart)
	var hole, size int64
	for {
		data, err := srcFs.Seek(hole, unix.SEEK_DATA)
		if err != nil {

			if errors.Is(syscall.ENXIO, err.(*fs.PathError).Unwrap()) {
				log.Println(err)
			}
			log.Println(err)
		}
		if data >= hole {
			hole, _ = srcFs.Seek(data, unix.SEEK_HOLE)
			if hole > data {
				dataSize := hole - data
				size += dataSize
				continue
			}
		}
		break
	}
	_, _ = srcFs.Seek(old, io.SeekStart)

	return size, nil
}

//go test -v generate_test.go -test.run TestGenerateSparseFile
func TestGenerateSparseFile(t *testing.T) {
	fd, err := os.OpenFile("/smb/sparsefile/test/test.txt", os.O_RDWR|os.O_CREATE|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		log.Panic("open failed", err)
	}

	fd.Seek(0, io.SeekStart)
	for i := 0; i < 1000; i++ {
		_, err := fd.WriteAt([]byte{97}, int64(10000*i))
		if err != nil {
			log.Println(err)
		}
	}
	size, err := GetSize(fd)
	if err != nil {
		log.Println(err)
		return
	}
	log.Println(size)
}
