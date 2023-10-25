package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

var (
	IpPtr   = flag.String("ip", "", "udp server ip")
	PortPtr = flag.Uint("Port", 8081, "udp server port")
)

type ConfigInfo struct {
	Ip                 string
	Port               uint16
	ReadTimeoutSecond  time.Duration
	WriteTimeoutSecond time.Duration
}
type UdpServer struct {
	Cfg          ConfigInfo
	addr         *net.UDPAddr
	udpConn      *net.UDPConn
	CloseSign    chan struct{}
	LogicHandles map[uint16]ILogicProcess
	//
	chSig chan os.Signal
	fire  int32
}

func NewUdpServerNode() *UdpServer {
	flag.Parse()
	s := &UdpServer{
		Cfg: ConfigInfo{
			Ip:                 *IpPtr,
			Port:               uint16(*PortPtr),
			ReadTimeoutSecond:  2 * time.Second,
			WriteTimeoutSecond: 2 * time.Second,
		},
		LogicHandles: make(map[uint16]ILogicProcess),
		CloseSign:    make(chan struct{}),
	}
	//
	s.RegisterLogicHandle(NewLoginHandle())
	return s
}
func (s *UdpServer) RegisterLogicHandle(busi ILogicProcess) {
	s.LogicHandles[busi.GetReqCmd()] = busi
}
func (s *UdpServer) GetLogicHandle(cmd uint16) ILogicProcess {
	ret, ok := s.LogicHandles[cmd]
	if !ok {
		return nil
	}
	return ret
}

// .....
func (s *UdpServer) Run() {
	s.chSig = make(chan os.Signal)
	signal.Notify(s.chSig, syscall.SIGINT, syscall.SIGTERM)
	var wg *sync.WaitGroup = new(sync.WaitGroup)

	go func() {
		for {
			//if stop, then return .
			if atomic.LoadInt32(&s.fire) == 1 {
				break
			}

			wg.Add(1)

			pkgDataProc := NewPackageDataProcess(s.udpConn, s.Cfg.ReadTimeoutSecond, s.Cfg.WriteTimeoutSecond)
			_, err := pkgDataProc.ReadData()

			if err != nil {
				wg.Done()
				continue
			}

			go func(pkgNode *PackageDataProcess) {
				defer wg.Done()

				if e := pkgNode.ParseHead(); e != nil { // parse, do busi, send msg.
					log.Println("parse pkg head fail. e: ", e)
					return
				}

				logicHandle := s.GetLogicHandle(pkgNode.Header.CmdType)
				if logicHandle == nil {
					log.Println("cmd: ", pkgNode.Header.CmdType, ", not implement logic process.")
					return
				}

				responseData := logicHandle.LogicProc(pkgNode.PayLoad.body)
				if len(responseData) <= 0 {
					return
				}
				pkgNode.SendData(responseData, logicHandle.GetResponseCmd())
			}(pkgDataProc)
		}
	}()

	fmt.Println("Signal: ", <-s.chSig)
	atomic.StoreInt32(&s.fire, 1)
	wg.Wait()
	s.udpConn.Close()

	log.Println("end of udp server.")
}

func (s *UdpServer) Start() {
	var e error
	s.addr, e = net.ResolveUDPAddr("udp", fmt.Sprintf("%v:%v", s.Cfg.Ip, s.Cfg.Port))
	if e != nil {
		log.Printf("get udp server addr fail, e: %v\n", e)
		return
	}

	net.ListenPacket("udp", fmt.Sprintf("%v:%v", s.Cfg.Ip, s.Cfg.Port))
	s.udpConn, e = net.ListenUDP("udp", s.addr)
	if e != nil {
		log.Printf("listen udp server fail, e: %v\n", e)
		return
	}
	return
}

func main() {
	s := NewUdpServerNode()
	s.Start()
	s.Run()
}
