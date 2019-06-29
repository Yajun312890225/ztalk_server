package server

import (
	"log"
	"net"
	"sync"
	"time"
)

//Session 一个session代表一个连接
type Session struct {
	CID   string
	Con   net.Conn
	times int64
	lock  sync.Mutex
	Phone string
}

//NewSession 新建连接
func NewSession(cid string, con net.Conn) *Session {
	return &Session{
		CID:   cid,
		Con:   con,
		times: time.Now().Unix(),
	}
}

func (s *Session) write(msg string) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	_, errs := s.Con.Write([]byte(msg))
	return errs
}

func (s *Session) close() {
	s.Con.Close()
}

func (s *Session) updateTime() {
	s.times = time.Now().Unix()
}

//SessionM SESSION管理类
type SessionM struct {
	ser         *Msf
	sessions    sync.Map
	onlinePhone sync.Map
	onlineCID   sync.Map
}

//NewSessonM 新建管理类
func NewSessonM(msf *Msf) *SessionM {
	if msf == nil {
		return nil
	}

	return &SessionM{
		ser: msf,
	}
}

//GetSessionByCID 获取session
func (s *SessionM) GetSessionByCID(cid string) *Session {
	tem, exit := s.sessions.Load(cid)
	if exit {
		if sess, ok := tem.(*Session); ok {
			return sess
		}
	}
	return nil
}

//SetPhoneByCID 设置手机号
func (s *SessionM) SetPhoneByCID(cid string, phone string) {
	s.onlinePhone.Store(phone, cid)
	s.onlineCID.Store(cid, phone)
}

//GetPhoneOnline  判断是否在线
func (s *SessionM) GetPhoneOnline(phone string) bool {
	_, ok := s.onlinePhone.Load(phone)
	return ok
}

//WriteByPhone 通过手机号进行转发
func (s *SessionM) WriteByPhone(phone string, msg []byte) {
	if tem, ok := s.onlinePhone.Load(phone); ok {
		if id, ok1 := tem.(string); ok1 {
			if se, ok2 := s.sessions.Load(id); ok2 {
				if sess, ok3 := se.(*Session); ok3 {
					if _, err := sess.Con.Write(msg); err != nil {
						s.DelSessionByCID(id)
						log.Println(err)
					}
				}
			} else {
				//找不到，说明已经下线，入离线库
				s.onlinePhone.Delete(phone)
			}
		}
	}
}

//SetSession 设置session
func (s *SessionM) SetSession(cid string, conn net.Conn) {
	sess := NewSession(cid, conn)
	s.sessions.Store(cid, sess)
}

//DelSessionByCID 关闭连接并删除
func (s *SessionM) DelSessionByCID(cid string) {
	tem, exit := s.sessions.Load(cid)
	if exit {
		if sess, ok := tem.(*Session); ok {
			sess.close()
		}
	}
	phone, exit := s.onlineCID.Load(cid)
	if exit {
		s.onlineCID.Delete(cid)
		s.onlinePhone.Delete(phone)

		_, err := (*s.ser.redisConn).Do("HMSET", "ZUE_"+phone.(string), "logouttime", time.Now().Unix(), "online", false)
		if err != nil {
			log.Println("redis HGET error:", err)
		}

	}
	s.sessions.Delete(cid)
}

//WriteToAll 向所有客户端发送消息
func (s *SessionM) WriteToAll(msg []byte) {
	// msg = s.ser.SocketType.Pack(msg)
	s.sessions.Range(func(key, val interface{}) bool {
		if val, ok := val.(*Session); ok {
			if err := val.write(string(msg)); err != nil {
				s.DelSessionByCID(key.(string))
				log.Println(err)
			}
		}
		return true
	})
}

//WriteByCID 向单个客户端发送信息
func (s *SessionM) WriteByCID(cid string, msg []byte) bool {
	tem, exit := s.sessions.Load(cid)
	if exit {
		if sess, ok := tem.(*Session); ok {
			if _, err := sess.Con.Write(msg); err == nil {
				return true
			}
		}
	}
	s.DelSessionByCID(cid)
	return false
}
