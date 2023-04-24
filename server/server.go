package main

import (
	"bufio"
	"encoding/binary"
	"flag"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cast"
	"io"
	"net"
	"os"
)

func main() {
	var (
		port uint64
		v    bool
	)
	flag.Uint64Var(&port, "port", 9992, "端口")
	flag.BoolVar(&v, "v", false, "是否显示日志")
	flag.Parse()
	if v {
		logrus.SetLevel(logrus.DebugLevel)
	} else {
		logrus.SetLevel(logrus.ErrorLevel)
	}
	listen, err := net.Listen("tcp", "127.0.0.1:"+cast.ToString(port))
	if err != nil {
		logrus.Panic(err)
	}
	for {
		conn, err := listen.Accept()
		if err != nil {
			logrus.Errorf("accept connection failed err %v ", err)
			continue
		}
		buffer := Buffer{
			buf: bufio.NewReaderSize(conn, 1024*1024*20),
		}
		go buffer.hande()
	}
}

type Buffer struct {
	buf         *bufio.Reader
	hasReadPath bool
	fd          *os.File
	path        string
	conn        net.Conn
}

func (c *Buffer) Read(b []byte) (n int, err error) {
	return c.conn.Read(b)
}

//要考虑一次读不完一条河一次读出多条的情况。
func (c *Buffer) hande() {
	defer c.conn.Close()
	//读取出路径的长度
	header := make([]byte, 8)
	_, err := io.ReadFull(c, header)
	if err != nil {
		return
	}
	length := binary.BigEndian.Uint64(header)
	bodyBuf := make([]byte, length)
	_, err = io.ReadFull(c, bodyBuf)
	if err != nil {
		return
	}
	c.path = string(bodyBuf)
	fd, err := os.OpenFile(c.path, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0644)
	if err != nil {
		logrus.Errorf("open file failed path %v  err %v", c.path, err)
		return
	}
	c.fd = fd
	//读完路径信息后
	//一段一段读，每段10MB
	sectionSize := uint64(1024 * 1024 * 10)
	header = make([]byte, 16)
	for {
		_, err = io.ReadFull(c, header)
		if err != nil {
			return
		}
		offset := binary.BigEndian.Uint64(header[:8])
		size := binary.BigEndian.Uint64(header[8:16])
		sectionBuf := make([]byte, sectionSize)
		for {
			if _, err := io.ReadFull(c, sectionBuf); err != nil {
				return
			}
			if _, err := c.fd.WriteAt(sectionBuf, int64(offset)); err != nil {
				return
			}
			size -= sectionSize
			if size < sectionSize {
				if size != 0 {
					sectionBuf = make([]byte, size)
					continue
				}
				break
			}
		}

	}

}
