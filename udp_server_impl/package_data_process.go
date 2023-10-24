package main

import (
	"encoding/binary"
	"errors"
	"log"
	"net"
	"time"
)

type PackageHead struct {
	PkgLen  uint32
	CmdType uint16
}

func (ph *PackageHead) ParseHead(data []byte) error {
	if len(data) < PackageLenFieldLen+CmdTypeFieldLen {
		return errors.New("package len less sum of pkgLenFieldLen  and cmdTypeFieldLen")
	}

	ph.PkgLen = binary.BigEndian.Uint32(data[:PackageLenFieldLen])
	if ph.PkgLen > PACKAGE_LEN_MAX {
		return errors.New("parse package len more than max len")
	}

	ph.CmdType = binary.BigEndian.Uint16(data[PackageLenFieldLen : PackageLenFieldLen+CmdTypeFieldLen])
	return nil

}

type PackagePayLoad struct {
	body []byte
}

type PackageDataProcess struct {
	conn     *net.UDPConn
	buf      []byte
	peerAddr *net.UDPAddr
	//
	readTimeout  time.Duration
	writeTimeout time.Duration
	//
	Header  *PackageHead
	PayLoad *PackagePayLoad
}

func NewPackageDataProcess(c *net.UDPConn, rTimeout, wTimeout time.Duration) *PackageDataProcess {
	return &PackageDataProcess{
		conn:         c,
		readTimeout:  rTimeout,
		writeTimeout: wTimeout,
		buf:          make([]byte, 1024*4), // read buf. TODO
		Header:       &PackageHead{},
		PayLoad:      &PackagePayLoad{},
	}
}
func (p *PackageDataProcess) ReadData() ([]byte, error) {
	p.conn.SetReadDeadline(time.Now().Add(p.readTimeout))

	retRead, clientAddr, e := p.conn.ReadFromUDP(p.buf)
	if e != nil {
		return nil, e
	}
	if retRead <= 0 {
		return nil, nil
	}

	p.peerAddr = clientAddr
	return p.buf[:retRead], nil
}
func (p *PackageDataProcess) ParseHead() error {
	e := p.Header.ParseHead(p.buf)
	if e != nil {
		return nil
	}
	//
	p.PayLoad.body = p.buf[PackageLenFieldLen+CmdTypeFieldLen:]
	return nil
}
func (p *PackageDataProcess) SendData(data []byte, rspCmdType uint16) error {
	sendBuf := make([]byte, PackageLenFieldLen+CmdTypeFieldLen+int32(len(data)))
	binary.BigEndian.PutUint32(sendBuf[0:PackageLenFieldLen], uint32(len(sendBuf)))
	binary.BigEndian.PutUint16(sendBuf[PackageLenFieldLen:PackageLenFieldLen+CmdTypeFieldLen], rspCmdType)
	copy(sendBuf[PackageLenFieldLen+CmdTypeFieldLen:], data)

	p.conn.SetWriteDeadline(time.Now().Add(p.writeTimeout))
	_, e := p.conn.WriteToUDP(sendBuf, p.peerAddr)
	if e != nil {
		log.Println("write data to peer node fail, e: ", e)
	}
	return e
}
