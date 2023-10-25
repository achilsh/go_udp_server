package main

import (
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

var (
	SvrIP   = flag.String("ip", "localhost", "udp server ip")
	SvrPort = flag.Int("port", 8081, "udp server port")
)

const (
	PKG_LEN_MAX  = 4
	CMD_TYPE_LEN = 2
)

const (
	LOGIN_CMD     uint16 = 1
	LOGIN_CMD_RSP uint16 = 2
)

type LoginReq struct {
	Name  string  `json:"name"`
	Age   int32   `json:"age"`
	Score float32 `json:"score"`
}
type LoginRsp struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data"`
}

type IUdpConnClient interface {
	Read([]byte) (int, error)
	Write(data []byte) (int, error)
	SetDeadLine(t time.Time)
	SetReadDeadline(t time.Time)
	SetWriteDeadline(t time.Time)
	Close()
}

type DialUdpConn struct {
	sAddr *net.UDPAddr
	c     *net.UDPConn
}

func (d *DialUdpConn) Read(data []byte) (int, error) {
	n, _, e := d.c.ReadFromUDP(data)
	return n, e
}
func (d *DialUdpConn) Write(data []byte) (int, error) {
	return d.c.Write(data)
}
func (d *DialUdpConn) SetDeadLine(t time.Time) {
	d.c.SetDeadline(t)
}
func (d *DialUdpConn) SetReadDeadline(t time.Time) {
	d.c.SetReadDeadline(t)
}
func (d *DialUdpConn) SetWriteDeadline(t time.Time) {
	d.c.SetWriteDeadline(t)
}
func (d *DialUdpConn) Close() {
	d.c.Close()
}

type PkgUdpConn struct {
	sAddr *net.UDPAddr
	c     net.PacketConn
}

func (d *PkgUdpConn) Read(data []byte) (int, error) {
	n, _, e := d.c.ReadFrom(data)
	return n, e
}
func (d *PkgUdpConn) Write(data []byte) (int, error) {
	return d.c.WriteTo(data, d.sAddr)
}
func (d *PkgUdpConn) SetDeadLine(t time.Time) {}
func (d *PkgUdpConn) SetReadDeadline(t time.Time) {
	d.c.SetReadDeadline(t)
}
func (d *PkgUdpConn) SetWriteDeadline(t time.Time) {
	d.c.SetWriteDeadline(t)
}
func (d *PkgUdpConn) Close() {

}

const (
	UDP_TYPE_CONN = 1
	UDP_TYPE_PKG  = 2
)

func NewClientConn(svrAddr *net.UDPAddr, cType int) IUdpConnClient {
	var ret IUdpConnClient
	var e error
	switch cType {
	case UDP_TYPE_CONN: //udp conn client
		du := &DialUdpConn{
			sAddr: svrAddr,
		}
		du.c, e = net.DialUDP("udp", nil, du.sAddr)
		if e != nil {
			return nil
		}
		ret = du

	case UDP_TYPE_PKG:
		pu := &PkgUdpConn{
			sAddr: svrAddr,
		}
		pu.c, e = net.ListenPacket("udp", ":0")
		if e != nil {
			return nil
		}
		ret = pu
	}

	return ret
}

func call_server() {
	udpAddr, e := net.ResolveUDPAddr("udp", fmt.Sprintf("%v:%d", *SvrIP, *SvrPort))
	if e != nil {
		log.Printf("resolve udp addr fail, err: %v\n", e)
		return
	}

	udpConn := NewClientConn(udpAddr, UDP_TYPE_PKG) //UDP_TYPE_PKG, UDP_TYPE_CONN
	if udpConn == nil {
		return
	}
	defer udpConn.Close()

	pkgSendPKg := func(i int) []byte {
		dataJson := &LoginReq{
			Name:  "name", //fmt.Sprintf("name: %v", i),
			Age:   int32(30 + i),
			Score: float32(11111.0 + i),
		}
		//
		data, e := json.Marshal(dataJson)
		if e != nil {
			log.Printf("marshal send data fail, e: %v\n", e)
			return nil
		}
		sendBufLen := PKG_LEN_MAX + CMD_TYPE_LEN + len(data)
		sendBuf := make([]byte, sendBufLen)
		binary.BigEndian.PutUint32(sendBuf[:PKG_LEN_MAX], uint32(sendBufLen))
		binary.BigEndian.PutUint16(sendBuf[PKG_LEN_MAX:PKG_LEN_MAX+CMD_TYPE_LEN], LOGIN_CMD)
		copy(sendBuf[PKG_LEN_MAX+CMD_TYPE_LEN:], data)

		return sendBuf
	}

	recvMsg := func() {
		recvBuf := make([]byte, 1024*4)
		udpConn.SetReadDeadline(time.Now().Add(2 * time.Second))
		rLen, e := udpConn.Read(recvBuf)
		if e != nil {
			log.Printf("read data from peer node fail, e: %v\n", e)
			return
		}
		if rLen <= 0 {
			return
		}
		recvBuf = recvBuf[:rLen]
		//
		recvLen := binary.BigEndian.Uint32(recvBuf[:PKG_LEN_MAX])
		if recvLen > 1024*4 {
			log.Printf("parse recv pkg len fail, len: %v\n", recvLen)
			return
		}
		//
		rspCmdTypeVal := binary.BigEndian.Uint16(recvBuf[PKG_LEN_MAX : PKG_LEN_MAX+CMD_TYPE_LEN])
		if rspCmdTypeVal != LOGIN_CMD_RSP {
			log.Printf("recv msg response cmd type not correct, v: %v\n", rspCmdTypeVal)
			return
		}

		data := recvBuf[PKG_LEN_MAX+CMD_TYPE_LEN:]
		rspMsg := &LoginRsp{}
		e = json.Unmarshal(data, rspMsg)
		if e != nil {
			log.Printf("unmarshal response data fail, e: %v\n", e)
			return
		}
		fmt.Printf("recv busi msg: %#v\n", rspMsg)
	}

	//...
	n := 3
	for i := 0; i < n; i++ {
		//send message
		sBuf := pkgSendPKg(i)
		udpConn.SetWriteDeadline(time.Now().Add(1 * time.Second))
		_, e := udpConn.Write(sBuf)
		if e != nil {
			log.Printf("send udp msg fail, e: %v\n", e)
			break
		}
		// recv message
		recvMsg()
	}

}
func parallel_call_server() {
	testNums := 10000
	var wg sync.WaitGroup
	for i := 0; i < testNums; i++ {
		wg.Add(1)
		time.Sleep(10 * time.Microsecond)

		go func() {
			defer wg.Done()
			call_server()
		}()
	}
	wg.Wait()
}

func main() {
	flag.Parse() //
	//call_server()
	parallel_call_server()
}
