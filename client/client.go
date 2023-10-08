package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"flag"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
	"io"
	"io/fs"
	"net"
	"os"
	"time"
)

func main() {
	var (
		path       string
		addr       string
		targetPath string
		v          bool
	)
	flag.StringVar(&path, "path", "", "文件路径")
	flag.StringVar(&addr, "addr", "", "目的地的ip和端口")
	flag.StringVar(&targetPath, "targetPath", "", "目的地的路径")
	flag.BoolVar(&v, "v", false, "是否显示日志")
	flag.Parse()
	if path == "" || addr == "" {
		logrus.Panic("path and addr must have a value")
	}
	if v {
		logrus.SetLevel(logrus.DebugLevel)
	} else {
		logrus.SetLevel(logrus.ErrorLevel)
	}

	fd, err := os.Open(path)
	if err != nil {
		logrus.Panic("path valid", err)
	}
	defer fd.Close()
	conn, err := net.DialTimeout("tcp", addr, time.Second*5)
	if err != nil {
		logrus.Panic("conn addr failed", err)
	}
	defer conn.Close()
	target := remote{conn: conn}
	stat, _ := fd.Stat()
	if err := target.writeMetaData(targetPath, stat.Size()); err != nil {
		logrus.Errorf("write path to remote failed %v", err)
		return
	}
	sfc := sparseFileClient{srcFs: fd}
	if err := sfc.Copy(context.Background(), target); err != nil {
		logrus.Printf("copy to remote failed %v", err)
	}
	//time.Sleep(time.Second * 90)
}

type sparseFileClient struct {
	srcFs *os.File
}
type remote struct {
	conn net.Conn
}

//元数据
type MetaData struct {
	Size int64
	Path string
}

var totalBytes int64

//写文件的相关信息。
func (r remote) writeMetaData(path string, size int64) error {
	logrus.Infof("write meta data %v len %v ", path, size)
	buf := make([]byte, 8)
	m := &MetaData{
		Size: size,
		Path: path,
	}
	data, _ := json.Marshal(m)
	binary.BigEndian.PutUint64(buf, uint64(len(data)))
	if _, err := r.conn.Write(buf); err != nil {
		return err
	}

	if _, err := r.conn.Write(data); err != nil {
		return err
	}
	return nil
}

// WriteAt 写数据
func (r remote) WriteAt(p []byte, off int64) (n int, err error) {
	totalBytes += int64(len(p))
	logrus.Infof("write offset %v len %v totaBytes %v\n", off, len(p), totalBytes)
	buf := bytes.NewBuffer(make([]byte, 0, 16))
	//偏移 使用大端编码的方式发送偏移
	if err := binary.Write(buf, binary.BigEndian, uint64(off)); err != nil {
		return 0, err
	}

	//长度 使用大端编码的方式发送数据的长度
	if err := binary.Write(buf, binary.BigEndian, uint64(len(p))); err != nil {
		return 0, err
	}
	if _, err := r.conn.Write(buf.Bytes()); err != nil {
		return 0, err
	}
	//数据
	if _, err := r.conn.Write(p); err != nil {
		return 0, err
	}
	return 0, nil
}

// Copy  将稀疏文件有效的块拷贝到目的地
func (sc sparseFileClient) Copy(ctx context.Context, writer io.WriterAt) error {
	curOffset := int64(0)
	//当前hole的offset 上一个hole的offset
	curHole, lastHole := int64(0), int64(0)
	stat, _ := sc.srcFs.Stat()
	end := stat.Size()
	sec := 1024 * 1024 * 10
	buf := make([]byte, sec)
	for {
		//重置buf
		if len(buf) < sec {
			buf = make([]byte, sec)
		}
		//如果跳到文件的结尾表示结束
		if curOffset == end {
			return nil
		}

		//https://www.zhihu.com/question/407305048
		//SEEK_DATA的意思很明确，就是从指定的offset开始往后找，找到在大于等于offset的第一个不是Hole的地址。如果offset正好指在一个DATA区域的中间，那就返回offset。
		//不要去处理这个错误，当文件为空或一些异常情况这个地方会报错
		data, err := sc.srcFs.Seek(curOffset, unix.SEEK_DATA)
		if err != nil {
			if errors.Is(unix.ENXIO, err.(*fs.PathError).Unwrap()) {
				return nil
			}
			return err
		}
		//logrus.Debug("data position ", data)
		//有时出现hole不是结尾，当data变成0的时候,data会小于上个hole的位置。
		if data < lastHole {
			return nil
		}
		//SEEK_HOLE的意思就是从offset开始找，找到大于等于offset的第一个Hole开始的地址。如果offset指在一个Hole的中间，那就返回offset。如果offset后面再没有更多的hole了，那就返回文件结尾。
		hole, _ := sc.srcFs.Seek(data, unix.SEEK_HOLE)
		//logrus.Debug("hole position ", hole)

		//空文件直接返回
		if hole == 0 && data == 0 {
			return nil
		}
		if hole != curHole {
			lastHole = curHole
			curHole = hole
		}
		//跳到数据的区的开始位置。
		curOffset, _ = sc.srcFs.Seek(data, io.SeekStart)

		dataZoneSize := hole - data
		//如果dataZoneSize 小于我们定义的buf,就将buf修改到到dataZoneSize的长度，不修改长度会读出非数据区的数据。
		if dataZoneSize < int64(len(buf)) {
			buf = buf[:dataZoneSize]
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			n, err := sc.srcFs.Read(buf)
			if err == io.EOF {
				return nil
			}
			if err != nil {
				return err
			}

			if _, err := writer.WriteAt(buf[:n], curOffset); err != nil {
				return err
			}
			curOffset += int64(n)
		}
	}

}
