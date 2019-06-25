package server

import (
	"log"
	"net"
	"sync"
	"time"
)

//Session 一个session代表一个连接
type Session struct {
	ID    uint32
	Con   net.Conn
	times int64
	lock  sync.Mutex
	Phone string
}

//NewSession 新建连接
func NewSession(id uint32, con net.Conn) *Session {
	return &Session{
		ID:    id,
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

//GetSessionByID 获取session
func (s *SessionM) GetSessionByID(id uint32) *Session {
	tem, exit := s.sessions.Load(id)
	if exit {
		if sess, ok := tem.(*Session); ok {
			return sess
		}
	}
	return nil
}

//SetPhoneByID 设置手机号
func (s *SessionM) SetPhoneByID(id uint32, phone string) {
	s.onlinePhone.Store(phone, id)
}

//WriteByPhone 通过手机号进行转发
func (s *SessionM) WriteByPhone(phone string, msg []byte) {
	// if tem, ok := s.onlinePhone.Load(phone); ok {
	// 	if id, ok1 := tem.(uint32); ok1 {
	// 		if se, ok2 := s.sessions.Load(id); ok2 {
	// 			if sess, ok3 := se.(*Session); ok3 {
	// 				if _, err := sess.Con.Write(msg); err != nil {
	// 					s.DelSessionByID(id)
	// 					log.Println(err)
	// 				}
	// 			}
	// 		} else {
	// 			//找不到，说明已经下线，入离线库
	// 			s.onlinePhone.Delete(phone)
	// 		}
	// 	}
	// }
}

//SetSession 设置session
func (s *SessionM) SetSession(fd uint32, conn net.Conn) {
	sess := NewSession(fd, conn)
	s.sessions.Store(fd, sess)
}

//DelSessionByID 关闭连接并删除
func (s *SessionM) DelSessionByID(id uint32) {
	tem, exit := s.sessions.Load(id)
	if exit {
		if sess, ok := tem.(*Session); ok {
			sess.close()
		}
	}
	s.sessions.Delete(id)
}

//WriteToAll 向所有客户端发送消息
func (s *SessionM) WriteToAll(msg []byte) {
	// msg = s.ser.SocketType.Pack(msg)
	s.sessions.Range(func(key, val interface{}) bool {
		if val, ok := val.(*Session); ok {
			if err := val.write(string(msg)); err != nil {
				s.DelSessionByID(key.(uint32))
				log.Println(err)
			}
		}
		return true
	})
}

//WriteByID 向单个客户端发送信息
func (s *SessionM) WriteByID(id uint32, msg []byte) bool {
	//把消息打包
	// msg = s.ser.SocketType.Pack(msg)

	tem, exit := s.sessions.Load(id)
	if exit {
		if sess, ok := tem.(*Session); ok {
			if _, err := sess.Con.Write(msg); err == nil {
				return true
			}
		}
	}
	s.DelSessionByID(id)
	return false
}

// HeartBeat 心跳检测   每秒遍历一次 查看所有sess 上次接收消息时间  如果超过 num 就删除该 sess
func (s *SessionM) HeartBeat(num int64) {
	for {
		time.Sleep(time.Second)
		s.sessions.Range(func(key, val interface{}) bool {
			tem, ok := val.(*Session)
			if !ok {
				return true
			}

			if time.Now().Unix()-tem.times > num {
				s.DelSessionByID(key.(uint32))
			}
			return true
		})

	}
}
