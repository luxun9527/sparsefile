package main

import (
	"context"
	"github.com/sirupsen/logrus"
	"io"
	"log"
	"os"
	"os/exec"
	"testing"
)

//go test -v client_test.go client.go -test.run TestCopyLocal
func TestCopyLocal(t *testing.T) {
	src, err := os.OpenFile("test.txt", os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0644)
	if err != nil {
		logrus.Errorf("open file failed path %v  err %v", src, err)
		return
	}
	for i := 0; i < 100; i++ {
		if _, err := src.WriteAt([]byte("a"), int64(i*1000000)); err != nil {
			log.Println(err)
			return
		}
	}
	defer src.Close()
	src.Seek(0, io.SeekCurrent)
	logrus.SetLevel(logrus.DebugLevel)
	sfc := sparseFileClient{srcFs: src}
	target, err := os.OpenFile("local.txt", os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0644)
	if err != nil {
		logrus.Errorf("open file failed path %v  err %v", target, err)
		return
	}
	if err := sfc.Copy(context.Background(), target); err != nil {
		logrus.Errorf("copy file failed  err %v", err)
		return
	}
	result, err := exec.Command("/bin/bash", "-c", `md5sum test.txt &&  md5sum local.txt`).Output()
	if err != nil {
		logrus.Errorf("err %v", err)
		return
	}
	logrus.Info(string(result))
}
