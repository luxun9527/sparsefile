# 远程拷贝稀疏文件

1、稀疏文件定义https://zh.wikipedia.org/wiki/%E7%A8%80%E7%96%8F%E6%96%87%E4%BB%B6

`dd if=/dev/null of=sparseFileTest count=0 bs=1MB seek=1024`  stat命令和du对比可以看出，这种文件实际的大小和stat显示的大小是不一样的。

```
[root@localhost gpush]# stat sparseFileTest 
  文件："sparseFileTest"
  大小：1024000000      块：0          IO 块：4096   普通文件
设备：fd00h/64768d      Inode：3374592     硬链接：1
权限：(0644/-rw-r--r--)  Uid：(    0/    root)   Gid：(    0/    root)
环境：unconfined_u:object_r:admin_home_t:s0
最近访问：2023-07-24 11:03:33.966130832 +0800
最近更改：2023-07-24 11:04:31.086918005 +0800
最近改动：2023-07-24 11:04:31.086918005 +0800

[root@localhost gpush]# du -sh sparseFileTest 
0       sparseFileTest
```

 使用scp或ftp传输该文件，会将复制1024000000个字节，我们可以使用SEEK_HOLE,SEEK_DATA的方式,取出稀疏文件的有效数据块然后传输。文件系统读取是按块读取一般是4k,所以传到远程的稀疏文件和真实的文件大小可能会有一点差异，一些无效的字节也被读取，不过一般不会很大。

服务端

```bash
nohup ./sparsefile-server -p 9992 
```

客户端

```bash
./sparsefile-client -path 稀疏文件的路径 -addr 远程的ip和端口 targetPath 远程文件的位置
```

