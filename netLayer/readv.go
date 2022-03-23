package netLayer

import (
	"io"
	"log"
	"net"
	"syscall"

	"github.com/hahahrfool/v2ray_simple/utils"
)

//v2ray里还使用了动态分配的方式，我们为了简便先啥也不做
// 实测16个buf已经完全够用，平时也就偶尔遇到5个buf的情况, 极速测速时会占用更多；
// 16个1500那就是 24000, 23.4375 KB, 不算小了;
var readv_buffer_allocLen = 16

func GetRawConn(reader io.Reader) syscall.RawConn {
	if sc, ok := reader.(syscall.Conn); ok {
		rawConn, err := sc.SyscallConn()
		if err != nil {
			if utils.CanLogDebug() {
				log.Println("can't convert syscall.Conn to syscall.RawConn", reader, err)
			}
			return nil
		} else {
			return rawConn
		}
	}

	return nil
}

/* ReadFromMultiReader 用于读端实现了 readv但是写端的情况，比如 从socks5读取 数据, 等裸协议的情况。
返回错误时，依然返回原 buffer或者新分配的buffer. 本函数不负责 释放分配的内存. 这样有时可以重复利用缓存。
若allocedBuffers未给出，会使用 utils.AllocMTUBuffers 来初始化 缓存。

小贴士：将该 net.Buffers 写入io.Writer的话，只需使用 其WriteTo方法, 即可自动适配writev。

TryCopy函数使用到了本函数。
*/
func ReadFromMultiReader(rawReadConn syscall.RawConn, mr utils.MultiReader, allocedBuffers net.Buffers) (net.Buffers, error) {

	if allocedBuffers == nil {
		allocedBuffers = utils.AllocMTUBuffers(mr, readv_buffer_allocLen)
	}

	var nBytes int32
	err := rawReadConn.Read(func(fd uintptr) bool {
		n := mr.Read(fd)
		if n < 0 {
			return false
		}

		nBytes = n
		return true
	})
	if err != nil {

		return allocedBuffers, err
	}
	if nBytes == 0 {
		err = io.EOF
		return allocedBuffers, io.EOF
	}

	nBuf := utils.ShrinkBuffers(allocedBuffers, int(nBytes))
	/*
		if utils.CanLogDebug() {
			// 可用于查看到底用了几个buf, 便于我们调整buf最大长度
			log.Println("release buf", len(bs)-nBuf)
		}
	*/

	return allocedBuffers[:nBuf], nil
}
