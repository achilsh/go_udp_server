package main

import (
	"errors"
	"fmt"
	"log"
)

type ILogicProcess interface {
	LogicProc(in []byte) []byte
	GetResponseCmd() uint16
	GetReqCmd() uint16
	//
	checkReqParams([]byte) error
	doInnerLogic() error
}

// this is demo for logic process, as login process.
const (
	LOGIN_CMD     uint16 = 1
	LOGIN_CMD_RSP uint16 = 2
)

// /////////////////////////////////
type LoginProcess struct {
	Req *LoginReq
	Rsp *LoginRsp
}

func NewLoginHandle() ILogicProcess {
	ret := &LoginProcess{
		Req: new(LoginReq),
		Rsp: new(LoginRsp),
	}

	//init response with default value.
	ret.Rsp.Code = 1000
	ret.Rsp.Message = "succc"
	ret.Rsp.Data = "ok"
	return ret
}

func (l *LoginProcess) GetReqCmd() uint16 {
	return LOGIN_CMD
}
func (l *LoginProcess) GetResponseCmd() uint16 {
	return LOGIN_CMD_RSP
}

func (l *LoginProcess) checkReqParams(data []byte) error {
	if len(data) <= 0 {
		l.Rsp.Code = 2000
		l.Rsp.Message = "input data is empty"
		return errors.New("input param is invalid")
	}
	return nil
}

func (l *LoginProcess) doInnerLogic() error {
	//TODO:
	return nil
}

func (l *LoginProcess) LogicProc(in []byte) []byte {
	codeHandle := GetCodecs(CODEC_JSON)
	for {
		if l.checkReqParams(in) != nil { //check input parameters.
			break
		}

		e := codeHandle.Decode(in, l.Req) // decode request message.
		if e != nil {
			l.Rsp.Code = 2001
			l.Rsp.Message = fmt.Sprintf("decode fail, e: %v", e)
			log.Printf("decode req msg fail, e: %v\n, msg: %v\n", e, string(in))
			break
		}

		l.doInnerLogic() // do business logic.

		break
	}

	rspData, e := codeHandle.Encode(l.Rsp)
	if e != nil {
		log.Printf("encode response to json fail, e: %v\n", e)
		return nil
	}
	//
	return rspData
}
