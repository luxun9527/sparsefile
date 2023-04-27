# 远程拷贝稀疏文件

实现的方式通过 SEEK_HOLE,SEEK_DATA,取出稀疏文件的有效数据快。文件系统读取是按块读取一般是4k,所以传到远程的稀疏文件和真实的文件大小可能会有一点差异，一些无效的字节也被读取，不过一般不会很大。

服务端

```bash
nohup ./sparsefile-server -p 9992 
```

客户端

```bash
./sparsefile-client -path 稀疏文件的路径 -addr 远程的ip和端口 targetPath 远程文件的位置
```

