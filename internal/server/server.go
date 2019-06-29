package server

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"sync/atomic"
	"time"

	"github.com/gomodule/redigo/redis"
)

//HEADERLENGTH 包头长度
const HEADERLENGTH = 11

//Msf server
type Msf struct {
	EventPool     *RouterMap
	BsonData      *Bson
	SessionMaster *SessionM
	redisConn     *redis.Conn
}

// NewMsf new
func NewMsf(red *redis.Conn) *Msf {

	msf := &Msf{
		EventPool: NewRouterMap(),
		BsonData:  NewBson(),
		redisConn: red,
	}
	msf.SessionMaster = NewSessonM(msf)
	return msf
}

// Listening 监听
func (m *Msf) Listening() {
	listener, err := net.Listen("tcp", "0.0.0.0:8000")
	if err != nil {
		log.Fatal(err)
	}
	fd := uint32(0)
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Print(err)
			continue
		}
		if fd > 1000 {
			atomic.AddUint32(&fd, ^uint32(1000))
		}
		newfd := atomic.AddUint32(&fd, uint32(1))
		cid := fmt.Sprintf("%d%d", time.Now().Unix(), newfd)
		m.SessionMaster.SetSession(cid, conn)
		go m.ConnHandle(m, m.SessionMaster.GetSessionByCID(cid))
	}
}

// ConnHandle 消息处理
func (m *Msf) ConnHandle(msf *Msf, sess *Session) {
	defer func() {
		log.Printf("cid = %s closed\n", sess.CID)
		msf.SessionMaster.DelSessionByCID(sess.CID)
	}()
	headBuff := make([]byte, 1024)
	tempBuff := make([]byte, 0)
	data := make([]byte, 20)
	var cmdid int16
	log.Printf("cid = %s , Address = %s\n", sess.CID, sess.Con.RemoteAddr().String())
	for {
		n, err := sess.Con.Read(headBuff)
		if err != nil {
			return
		}
		tempBuff = append(tempBuff, headBuff[:n]...)
		tempBuff, data, cmdid, err = m.decode(tempBuff)
		sess.updateTime()
		if err != nil {
			continue
		}
		if len(data) == 0 {
			//heart
			if time.Now().Unix()-sess.times > 60000 {
				msf.SessionMaster.DelSessionByCID(sess.CID)
			}
			continue
		}
		v, _, err := m.BsonData.Get(data)
		if err != nil {
			log.Printf("get data err:%v\n", err)
			continue
		}
		if ok := m.hook(sess.CID, cmdid, v); !ok {
			log.Println("hook error cmdid ", cmdid)
		}
	}

}

func (m *Msf) hook(udi string, cmdid int16, requestData map[string]interface{}) bool {
	if action, ok := m.EventPool.pools[cmdid]; ok {
		return action(udi, requestData)
	}
	return false
}
func (m *Msf) decode(buff []byte) ([]byte, []byte, int16, error) {
	length := len(buff)
	if length < HEADERLENGTH {
		return buff, nil, 0, nil
	}
	cmdid, bodyLen, ok := m.parseHead(buff)
	if ok != true {
		return buff, nil, 0, nil
	}
	data := buff[HEADERLENGTH : HEADERLENGTH+bodyLen]
	buffs := buff[HEADERLENGTH+bodyLen:]
	return buffs, data, cmdid, nil
}
func (m *Msf) parseHead(data []byte) (int16, int, bool) {
	var length int32
	var cmdid int16
	var bodyLen int

	by := bytes.NewBuffer(data[:4])
	binary.Read(by, binary.LittleEndian, &length)
	if length != int32(len(data)) {
		return cmdid, bodyLen, false
	}
	id := bytes.NewBuffer(data[4:6])
	binary.Read(id, binary.LittleEndian, &cmdid)
	bodyLen = int(length) - HEADERLENGTH
	return cmdid, bodyLen, true
}
