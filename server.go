package main

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"errors"
	"flag"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cast"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
)

func main() {
	var (
		port uint64
		v    bool
	)
	flag.Uint64Var(&port, "p", 9992, "端口")
	flag.BoolVar(&v, "v", false, "是否显示日志")
	flag.Parse()
	if v {
		logrus.SetLevel(logrus.DebugLevel)
	} else {
		logrus.SetLevel(logrus.ErrorLevel)
	}
	listen, err := net.Listen("tcp", "0.0.0.0:"+cast.ToString(port))
	if err != nil {
		logrus.Panic(err)
	}
	for {
		conn, err := listen.Accept()
		if err != nil {
			logrus.Errorf("accept connection failed err %v ", err)
			continue
		}
		buffer := &sparseFileServer{
			buf:  bufio.NewReaderSize(conn, 1024*1024*20),
			conn: conn,
		}
		go buffer.hande()
	}
}

type sparseFileServer struct {
	buf  *bufio.Reader
	conn net.Conn
}

func (c *sparseFileServer) Read(b []byte) (n int, err error) {
	return c.conn.Read(b)
}

type MetaData struct {
	Size int64
	Path string
}

func (c *sparseFileServer) hande() {
	defer c.conn.Close()
	//读取出路径的长度
	header := make([]byte, 8)
	_, err := io.ReadFull(c, header)
	if err != nil {
		return
	}
	//先读出元数据
	length := binary.BigEndian.Uint64(header)
	md := make([]byte, length)
	_, err = io.ReadFull(c, md)
	if err != nil {
		return
	}
	var meta MetaData
	if err := json.Unmarshal(md, &meta); err != nil {
		logrus.Errorf("unmarshal meta data failed  err %v", err)
		return
	}

	if err := os.MkdirAll(filepath.Dir(meta.Path), 0644); err != nil {
		logrus.Errorf("create dir failed  err %v", err)
	}

	fd, err := os.OpenFile(meta.Path, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0644)
	if err != nil {
		logrus.Errorf("open file failed path %v  err %v", meta.Path, err)
		return
	}
	//创建稀疏文件
	if err := fd.Truncate(meta.Size); err != nil {
		logrus.Errorf("truncate file failed path %v  err %v", meta.Path, err)
		return
	}
	logrus.Infof("receive metadata path=%v size=%v", meta.Path, meta.Size)

	//读完路径信息后
	//一段一段读，每段10MB
	sectionSize := uint64(1024 * 1024 * 10)
	header = make([]byte, 16)
	for {
		_, err = io.ReadFull(c, header)
		if err != nil {
			if !errors.Is(err, io.EOF) {
				logrus.Errorf("read msg head  failed err =%v", err)
			}
			return
		}
		offset := binary.BigEndian.Uint64(header[:8])
		size := binary.BigEndian.Uint64(header[8:16])
		logrus.Infof("receive data offset =%v size=%v", offset, size)
		sectionBuf := make([]byte, sectionSize)
		for size > sectionSize {
			if _, err := io.ReadFull(c, sectionBuf); err != nil {
				if !errors.Is(err, io.EOF) {
					logrus.Errorf("read data  failed err =%v", err)
				}
				return
			}
			if _, err := fd.WriteAt(sectionBuf, int64(offset)); err != nil {
				return
			}
			size -= sectionSize

		}
		if size != 0 {
			sectionBuf = make([]byte, size)
			if n, err := io.ReadFull(c, sectionBuf); err != nil {
				log.Println(n)
				return
			}
			if _, err := fd.WriteAt(sectionBuf, int64(offset)); err != nil {
				return
			}
		}

	}

}
