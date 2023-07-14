package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"strconv"

	"github.com/xtaci/kcptun/generic"
	"github.com/xtaci/smux"
)

func socks5(client *smux.Stream) {
	defer client.Close()

	b := make([]byte, 1024)
	n, err := client.Read(b[:])
	if err != nil {
		//log.Println(err)
		client.Close()
		return
	}
	if b[0] != 0x05 {
		client.Close()
		return
	}
	client.Write([]byte{0x05, 0x00}) //不需要验证
	n, err = client.Read(b[:])
	if err != nil {
		//log.Println(err)
		client.Close()
		return
	}
	//log.Printf("====>%v", b)
	var host, port string

	switch b[3] {
	case 0x01: //IP V4
		host = net.IPv4(b[4], b[5], b[6], b[7]).String()
	case 0x03: //域名
		host = string(b[5 : n-2]) //b[4]表示域名的长度
	case 0x04: //IP V6
		host = net.IP{b[4], b[5], b[6], b[7], b[8], b[9], b[10], b[11], b[12], b[13], b[14], b[15], b[16], b[17], b[18], b[19]}.String()
	}
	port = strconv.Itoa(int(b[n-2])<<8 | int(b[n-1]))
	objhost := net.JoinHostPort(host, port)

	p2, err := net.Dial("tcp", objhost)
	if err != nil {
		log.Println("no server available")
		client.Write([]byte{0x05, 0x03, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}) //响应客户端连接失败
		client.Close()
		return
	}
	client.Write([]byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}) //响应客户端连接成功
	streamCopy := func(dst io.Writer, src io.ReadCloser) {
		if _, err := generic.Copy(dst, src); err != nil {
			if err == smux.ErrInvalidProtocol {
				log.Println("smux", err, "in:", fmt.Sprint(client.RemoteAddr(), "(", client.ID(), ")"), "out:", p2.RemoteAddr())
			}
		}
		client.Close()
		p2.Close()
	}

	go streamCopy(p2, client)
	streamCopy(client, p2)
}
