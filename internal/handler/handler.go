package handler

import (
	"fmt"
	"log"
	"strconv"
	"time"
	"ztalk_server/api/rp"
	"ztalk_server/internal/database"
	"ztalk_server/internal/server"
	"ztalk_server/internal/utils"

	"github.com/gogo/protobuf/proto"
	"github.com/gomodule/redigo/redis"
)

const (
	heart = int16(iota)
	auth
	challenge
	fail
	succ
	reqSyncContact
	rspSyncContact
	reqUserinfo
	rspUSerinfo
	reqUserState
	rspUserState
	reqSetUser
	rspSetUser
	reqMyprivacy
	rspMyprivacy
	reqSetprivacy
	rspSetprivacy
	reqC2Cmsg
	askC2Cmsg
	failC2Cmsg
	notifyMsg
	askNotify
	reqOffmsg
	rspOffmsg
	reqheart
	sidClose
	rptReadmsg
	notifySetuser
	reqMyblock
	rspMyblock
	reqSetblock
	rspSetblock
	reqCreategroup
	rspCreategroup
	reqAddmember
	rspAddmember
	reqExitgroup
	rspExitgroup
	reqKickgroup
	rspKickgroup
	reqSetGroupInfo
	rspSetGroupInfo
	reqSetGroupAdmin
	rspSetGroupAdmin
	reqGroupList
	rspGroupList
	reqGroup
	rspGroup
	reqGroupMember
	rspGroupMember
	notifyGroup
	notifyReLogin //异地登录通知
	reqC2Gmsg
	askC2Gmsg
	failC2Gmsg
	reqSelfInfo
	rspSelfInfo
	commTip
	reqDelAccount
	rspDelAccount
	gateChange
	reqBeginVoice
	reqUploadVoice
	reChallenge
)
const key = "006fef6cce9e2900d49f906bef179bf1"

//Handler sockethandler
type Handler struct {
	msf       *server.Msf
	db        *database.DB
	ut        *utils.Ut
	redisConn *redis.Conn
}

//NewHandler new
func NewHandler(msf *server.Msf, db *database.DB, ut *utils.Ut, red *redis.Conn) *Handler {
	handler := Handler{
		msf:       msf,
		db:        db,
		ut:        ut,
		redisConn: red,
	}
	msf.EventPool.Register(auth, handler.authMessage)
	msf.EventPool.Register(reqSyncContact, handler.syncContact)
	msf.EventPool.Register(reqUserinfo, handler.userInfo)
	msf.EventPool.Register(reqUserState, handler.userState)
	msf.EventPool.Register(reqSetUser, handler.setUser)
	msf.EventPool.Register(reqSelfInfo, handler.selfInfo)
	msf.EventPool.Register(reqC2Cmsg, handler.c2cMessage)
	return &handler
}

func (h *Handler) authMessage(cid string, reqData map[string]interface{}) bool {
	log.Println(reqData)
	var ok bool
	var ty int
	var phone string
	var password string
	var nonce string
	var source string
	var md5sign string

	result := make(map[string]interface{})
	md5sign, ok = reqData["sign"].(string)
	if ok == false {
		log.Println("sign error")
		return false
	}
	phone, ok = reqData["phone"].(string)
	if ok == false {
		log.Println("phone error")
		return false
	}
	ty, ok = reqData["type"].(int)
	if ok == false {
		log.Println("type error")
		return false
	}
	source, ok = reqData["source"].(string)
	if ok == false {
		log.Println("source error")
		return false
	}

	res, err := redis.ByteSlices((*h.redisConn).Do("HMGET", "ZU_"+phone, "passwd", "nonce"))
	if err != nil {
		fmt.Println("redis HGET error:", err)

		if ty != 1 {
			qus := fmt.Sprintf("SELECT fPassword , fNonce FROM tuser WHERE fPhone='%s'", phone)
			err := h.db.QueryOne(qus).Scan(&password, &nonce)
			if err != nil {
				log.Printf("scan failed, err:%v\n", err)
				return false
			}
		} else {
			qus := fmt.Sprintf("SELECT fPassword FROM tuser WHERE fPhone='%s'", phone)
			err := h.db.QueryOne(qus).Scan(&password)
			if err != nil {
				log.Printf("scan failed, err:%v\n", err)
				return false
			}
		}
	} else {
		password = string(res[0])
		nonce = string(res[1])
	}

	var checksign string
	if ty == 1 {
		checksign = fmt.Sprintf("%s%s%02x%s", phone, source, []byte(password), key)
	} else {
		checksign = fmt.Sprintf("%s%s%02x%s%s", phone, source, []byte(password), nonce, key)
	}

	if md5sign != h.ut.Md5(checksign) {
		result["desc"] = "sign error"
		h.msf.SessionMaster.WriteByCID(cid, h.msf.BsonData.Set(result, fail))
		return true
	}
	newNonce := h.ut.GetNonce()
	result["nonce"] = newNonce
	ex := fmt.Sprintf("UPDATE tuser SET fNonce='%s' WHERE fPhone='%s'", newNonce, phone)
	ok = h.db.UpdateData(ex)
	if ok == false {
		log.Println("update nonce error")
		return false
	}
	_, err = (*h.redisConn).Do("HMSET", "ZU_"+phone, "passwd", password, "nonce", newNonce)
	if err != nil {
		log.Println("redis HGET error:", err)
	}
	if ty == 1 {
		h.msf.SessionMaster.WriteByCID(cid, h.msf.BsonData.Set(result, challenge))
	} else if ty == 0 {
		result["time"] = int(time.Now().Unix())
		result["ext0"] = []map[string]interface{}{
			{"ext1": "192.168.0.98", "ext2": 9000},
		}
		var isFirstLogin int
		qus := fmt.Sprintf("SELECT fFirstLogin FROM tuser WHERE fPhone='%s'", phone)
		err := h.db.QueryOne(qus).Scan(&isFirstLogin)
		if err != nil {
			log.Printf("scan failed, err:%v\n", err)
			return false
		}

		result["ext3"] = isFirstLogin
		if isFirstLogin == 1 {
			e := fmt.Sprintf("UPDATE tuser SET fFirstLogin=0 WHERE fPhone='%s'", phone)
			ok = h.db.UpdateData(e)
			if ok == false {
				log.Println("update fFirstLogin error")
				return false
			}
		}
		//redis userEx
		var nickname, iconresid, sdesc string
		qus = fmt.Sprintf("SELECT fNickname,fIconresid,fSdesc FROM tuser WHERE fPhone='%s'", phone)
		err = h.db.QueryOne(qus).Scan(&nickname, &iconresid, &sdesc)
		if err != nil {
			log.Printf("scan failed, err:%v\n", err)
			return false
		}
		_, err = (*h.redisConn).Do("HMSET", "ZUE_"+phone, "nickname", nickname, "iconresid", iconresid, "sdesc", sdesc, "setonline", "0", "seticon", "0", "setsdesc", "0", "logintime", time.Now().Unix(), "online", true)
		if err != nil {
			log.Println("redis HSETZUR error:", err)
		}
		log.Println(result)
		h.msf.SessionMaster.SetPhoneByCID(cid, phone)
		h.msf.SessionMaster.WriteByCID(cid, h.msf.BsonData.Set(result, succ))
	}

	return true
}

func (h *Handler) syncContact(cid string, reqData map[string]interface{}) bool {
	log.Println(reqData)
	result := make(map[string]interface{})
	if phone, ok := reqData["phone"].(string); ok {
		var userID int
		s := fmt.Sprintf("SELECT fUserId FROM tuser WHERE fPhone='%s'", phone)
		err := h.db.QueryOne(s).Scan(&userID)
		if err != nil {
			log.Printf("scan failed, err:%v\n", err)
			return false
		}
		result["in"] = []map[string]interface{}{}
		if add, ok := reqData["add"].([]interface{}); ok {
			for _, arry := range add {
				if friendPhone, ok := arry.(string); ok {
					var friendUserID int64
					s := fmt.Sprintf("SELECT fUserId FROM tuser WHERE fPhone='%s'", friendPhone)
					err := h.db.QueryOne(s).Scan(&friendUserID)
					if err != nil {
						log.Printf("User not exist\n")
						//redis friend reg == false
						// friend := &rp.Friend{
						// 	UserID:   proto.Int64(-1),
						// 	Contact:  proto.Bool(true),
						// 	Reg:      proto.Bool(false),
						// 	LastTime: proto.Int64(time.Now().Unix()),
						// }
						// d, _ := proto.Marshal(friend)
						// _, err := (*h.redisConn).Do("HSET", "ZUC_"+phone, friendPhone, d)
						// if err != nil {
						// 	log.Println("redis HGET error:", err)
						// }
					} else {
						u := fmt.Sprintf("INSERT INTO tcontact(fUserId,fFriendUserId,fContactType ,fLastTime) VALUES(%d,%d,'%s',FROM_UNIXTIME(%d)) ON DUPLICATE KEY UPDATE fContactType = '%s',fLastTime = FROM_UNIXTIME(%d)", userID, friendUserID, "friend", time.Now().Unix(), "friend", time.Now().Unix())
						if ok = h.db.UpdateData(u); ok {
							log.Println("Contact update")
							result["in"] = append(result["in"].([]map[string]interface{}), map[string]interface{}{
								"phone": friendPhone,
							})
							friend := &rp.Friend{
								UserID:   proto.Int64(friendUserID),
								Contact:  proto.Bool(true),
								Reg:      proto.Bool(true),
								LastTime: proto.Int64(time.Now().Unix()),
							}
							d, _ := proto.Marshal(friend)
							_, err := (*h.redisConn).Do("HSET", "ZUC_"+phone, friendPhone, d)
							if err != nil {
								log.Println("redis HGET error:", err)
							}

						}
					}
				}
			}
		}
		if del, ok := reqData["del"].([]interface{}); ok {
			for _, arry := range del {
				if friendPhone, ok := arry.(string); ok {
					var friendUserID int64
					s := fmt.Sprintf("SELECT fUserId FROM tuser WHERE fPhone='%s'", friendPhone)
					err := h.db.QueryOne(s).Scan(&friendUserID)
					if err != nil {
						log.Printf("User not exist\n")
						// friend := &rp.Friend{
						// 	UserID:   proto.Int64(-1),
						// 	Contact:  proto.Bool(false),
						// 	Reg:      proto.Bool(false),
						// 	LastTime: proto.Int64(time.Now().Unix()),
						// }
						// d, _ := proto.Marshal(friend)
						// _, err := (*h.redisConn).Do("HSET", "ZUC_"+phone, friendPhone, d)
						// if err != nil {
						// 	log.Println("redis HSET error:", err)
						// }
					} else {
						u := fmt.Sprintf("INSERT INTO tcontact(fUserId,fFriendUserId,fContactType ,fLastTime) VALUES(%d,%d,'%s',FROM_UNIXTIME(%d)) ON DUPLICATE KEY UPDATE fContactType = '%s',fLastTime = FROM_UNIXTIME(%d)", userID, friendUserID, "deleted", time.Now().Unix(), "deleted", time.Now().Unix())
						if ok = h.db.UpdateData(u); ok {
							log.Println("Contact update")
							friend := &rp.Friend{
								UserID:   proto.Int64(friendUserID),
								Contact:  proto.Bool(false),
								Reg:      proto.Bool(true),
								LastTime: proto.Int64(time.Now().Unix()),
							}
							d, _ := proto.Marshal(friend)
							_, err := (*h.redisConn).Do("HSET", "ZUC_"+phone, friendPhone, d)
							if err != nil {
								log.Println("redis HGET error:", err)
							}
						}
					}
				}
			}
		}
		h.msf.SessionMaster.WriteByCID(cid, h.msf.BsonData.Set(result, rspSyncContact))
		return true
	}
	return false
}

func (h *Handler) userInfo(cid string, reqData map[string]interface{}) bool {
	log.Println(reqData)
	result := make(map[string]interface{})
	if phone, ok := reqData["phone"].(string); ok {

		lasttime, ok := reqData["lasttime"].(string)
		if ok == false {
			log.Println("get lasttime error")
			return false
		}

		data, err := redis.ByteSlices((*h.redisConn).Do("HGETALL", "ZUC_"+phone))
		if err != nil {
			log.Println("redis HGET error:", err)
			return false
		}
		result["userlist"] = []map[string]interface{}{}
		for i := 0; i < len(data); i++ {
			// for _, friendData := range data {
			friendPhone := string(data[i])
			i++
			friend := &rp.Friend{}
			err := proto.Unmarshal(data[i], friend)
			if err != nil {
				log.Println("proto Unmarshal error:", err)
			}
			if *friend.Contact && *friend.Reg {
				if timestamp, err := strconv.ParseInt(lasttime, 10, 64); err == nil && timestamp < *friend.LastTime {
					userProfile, err := redis.ByteSlices((*h.redisConn).Do("HMGET", "ZUE_"+friendPhone, "iconresid", "sdesc"))
					if err != nil {
						log.Println("redis HGET error:", err)
						return false
					}
					result["userlist"] = append(result["userlist"].([]map[string]interface{}), map[string]interface{}{
						"phone": friendPhone,
						"icon":  string(userProfile[0]),
						"sdesc": string(userProfile[1]),
					})
				}
			}

		}
		result["lasttime"] = strconv.FormatInt(time.Now().Unix(), 10)
		h.msf.SessionMaster.WriteByCID(cid, h.msf.BsonData.Set(result, rspUSerinfo))
		return true
	}

	return false
}
func (h *Handler) userState(cid string, reqData map[string]interface{}) bool {
	log.Println(reqData)
	result := make(map[string]interface{})
	if _, ok := reqData["phone"].(string); ok {
		result["userstate"] = []map[string]interface{}{}
		if contacts, ok := reqData["contacts"].([]string); ok {
			for _, contact := range contacts {
				user, err := redis.ByteSlices((*h.redisConn).Do("HMGET", "ZUE_"+contact, "setonline", "logouttime", "online"))
				if err != nil {
					log.Println("redis HGET error:", err)
					return false
				}
				if string(user[0]) == "0" {
					result["userstate"] = append(result["userstate"].([]map[string]interface{}), map[string]interface{}{
						"phone":  contact,
						"status": -1,
					})
				} else {
					if online, err := redis.Bool(user[3], err); err == nil && online {
						result["userstate"] = append(result["userstate"].([]map[string]interface{}), map[string]interface{}{
							"phone":  contact,
							"status": 0,
						})
					} else {
						if logouttime, err := redis.Int64(user[2], err); err == nil {
							result["userstate"] = append(result["userstate"].([]map[string]interface{}), map[string]interface{}{
								"phone":  contact,
								"status": int(logouttime),
							})
						}
					}
				}
			}
		}
		h.msf.SessionMaster.WriteByCID(cid, h.msf.BsonData.Set(result, rspUserState))
		return true
	}
	return false
}

func (h *Handler) setUser(cid string, reqData map[string]interface{}) bool {
	log.Println(reqData)
	if phone, ok := reqData["phone"].(string); ok {
		result := make(map[string]interface{})
		result["items"] = []map[string]interface{}{}
		if items, ok := reqData["items"].([]map[string]interface{}); ok {
			userData := map[string]string{}
			for _, item := range items {
				name, ok := item["name"].(string)
				if ok == false {
					log.Println("name error")
				}
				value, ok := item["value"].(string)
				if ok == false {
					log.Println("value error")
				}
				userData[name] = value
			}
			e := fmt.Sprintf("UPDATE tuser SET fNickname='%s', fIconresid='%s' , fSdesc='%s' WHERE fPhone='%s'", userData["nick"], userData["icon"], userData["sdesc"], phone)
			ok = h.db.UpdateData(e)
			if ok == false {
				log.Println("set user error")
				return false
			}
			_, err := (*h.redisConn).Do("HMSET", "ZUE_"+phone, "nickname", userData["nick"], "iconresid", userData["icon"], "sdesc", userData["sdesc"])
			if err != nil {
				log.Println("redis HMSET error:", err)
				return false
			}

			for k, v := range userData {
				result["items"] = append(result["items"].([]map[string]interface{}), map[string]interface{}{
					"name":  k,
					"value": v,
				})
			}
		}
		result["phone"] = phone
		h.msf.SessionMaster.WriteByCID(cid, h.msf.BsonData.Set(result, rspSetUser))

		//notify_setUser
		q := fmt.Sprintf("SELECT t2.fUserId, t2.fPhone FROM tcontact t1,tuser t2 WHERE t1.fUserId = t2.fUserId  AND t1.fContactType ='friend' AND t1.fFriendUserId IN (SELECT fUserId FROM tuser WHERE fPhone = '%s' )", phone)
		rows, err := h.db.Query(q)
		if err != nil {
			log.Printf("Query failed,err:%v", err)
			return false
		}
		phoneList := []string{}
		userIDList := []int64{}
		for rows.Next() {
			var friendPhone string
			var friendUserID int64
			err = rows.Scan(&friendUserID, &friendPhone)
			if err != nil {
				fmt.Printf("Scan failed,err:%v", err)
				return false
			}
			userIDList = append(userIDList, friendUserID)
			phoneList = append(phoneList, friendPhone)
		}
		//fmt.Println(phoneList)
		for index, distphone := range phoneList {
			friend := &rp.Friend{
				UserID:   proto.Int64(userIDList[index]),
				Contact:  proto.Bool(true),
				Reg:      proto.Bool(true),
				LastTime: proto.Int64(time.Now().Unix()),
			}
			d, _ := proto.Marshal(friend)
			_, err := (*h.redisConn).Do("HSET", "ZUC_"+distphone, phone, d)
			if err != nil {
				log.Println("redis HGET error:", err)
				return false
			}

			distMsg := map[string]interface{}{
				"lasttime": strconv.FormatInt(time.Now().Unix(), 10),
				"phone":    phone,
				"items":    result["items"],
			}
			if ok := h.msf.SessionMaster.GetPhoneOnline(distphone); ok {
				h.msf.SessionMaster.WriteByPhone(distphone, h.msf.BsonData.Set(distMsg, notifySetuser))
			}
		}
		return true
	}
	return false
}

func (h *Handler) selfInfo(cid string, reqData map[string]interface{}) bool {
	log.Println(reqData)
	if phone, ok := reqData["phone"].(string); ok {
		result := make(map[string]interface{})
		result["items"] = []map[string]interface{}{}
		if items, ok := reqData["items"].([]map[string]interface{}); ok {
			userProfile, err := redis.ByteSlices((*h.redisConn).Do("HMGET", "ZUE_"+phone, "nickname", "seticon", "iconresid", "setsdesc", "sdesc"))
			if err != nil {
				log.Println("redis HGET error:", err)
				return false
			}
			for _, item := range items {
				name, ok := item["name"].(string)
				if ok == false {
					log.Println("name error")
				}
				switch name {
				case "nick":
					result["items"] = append(result["items"].([]map[string]interface{}), map[string]interface{}{
						"name":  name,
						"value": string(userProfile[0]),
					})

				case "icon":
					if string(userProfile[1]) == "0" {
						result["items"] = append(result["items"].([]map[string]interface{}), map[string]interface{}{
							"name":  name,
							"value": string(userProfile[2]),
						})
					} else {
						result["items"] = append(result["items"].([]map[string]interface{}), map[string]interface{}{
							"name":  name,
							"value": "",
						})
					}

				case "sdesc":
					if string(userProfile[3]) == "0" {
						result["items"] = append(result["items"].([]map[string]interface{}), map[string]interface{}{
							"name":  name,
							"value": string(userProfile[4]),
						})
					} else {
						result["items"] = append(result["items"].([]map[string]interface{}), map[string]interface{}{
							"name":  name,
							"value": "",
						})
					}
				default:
					log.Println("items name error")
				}
			}
		}
		result["phone"] = phone
		h.msf.SessionMaster.WriteByCID(cid, h.msf.BsonData.Set(result, rspSelfInfo))
		return true
	}
	return false
}
func (h *Handler) c2cMessage(cid string, reqData map[string]interface{}) bool {
	log.Println(reqData)

	result := make(map[string]interface{})

	from, ok := reqData["from"].(string)
	if ok == false {
		log.Println("from phone not found")
		return false
	}
	to, ok := reqData["to"].(string)
	if ok == false {
		log.Println("to phone not found")
		return false
	}
	seqid, ok := reqData["seqid"].(string)
	if ok == false {
		log.Println("seqid not found")
		return false
	}
	//校验好友关系
	// friend := &rp.Friend{}
	// fData, err := redis.Bytes((*h.redisConn).Do("HGET", "ZUC_"+from, to))
	// if err != nil {
	// 	log.Println("redis HGET error:", err)
	// 	return false
	// }
	// err = proto.Unmarshal(fData, friend)
	// if err != nil {
	// 	log.Println("proto Unmarshal error:", err)
	// 	return false
	// }
	// if *friend.Contact == false || *friend.Reg == false {
	// 	result["to"] = to
	// 	result["seqid"] = seqid
	// 	h.msf.SessionMaster.WriteByCID(cid, h.msf.BsonData.Set(result, failC2Cmsg))
	// 	return true
	// }

	contentType, ok := reqData["type"].(int)
	if ok == false {
		log.Println("type not found")
		return false
	}
	content, ok := reqData["content"].([]byte)
	if ok == false {
		log.Println("content not found")
		return false
	}
	result["to"] = to
	result["seqid"] = seqid
	result["msgid"] = strconv.FormatInt(time.Now().Unix(), 10) + cid
	h.msf.SessionMaster.WriteByCID(cid, h.msf.BsonData.Set(result, askC2Cmsg))

	msginfo := map[string]interface{}{
		"msgid":   result["msgid"],
		"msgtime": time.Now().Unix(),
		"from":    from,
		"to":      to,
		"type":    contentType,
		"content": content,
	}
	binaryData := h.msf.BsonData.TransFromMap(&msginfo)
	notify := map[string]interface{}{}
	notify["msglist"] = []map[string]interface{}{}
	notify["msglist"] = append(notify["msglist"].([]map[string]interface{}), map[string]interface{}{
		"msgtype": 0x0001,
		"msginfo": binaryData.Bytes(),
	})

	if ok := h.msf.SessionMaster.GetPhoneOnline(to); ok {
		//在线
		log.Println(to, "is online")
		h.msf.SessionMaster.WriteByPhone(to, h.msf.BsonData.Set(notify, notifyMsg))
	} else {
		//离线库
		log.Println(to, "is not online")
		u := fmt.Sprintf("INSERT INTO toffmsg(fPhone,fMsgId,fMsgType,fMsgInfo,fCreateTime) VALUES('%s','%s',%d,'%s',FROM_UNIXTIME(%d))", to, result["msgid"], 0x0001, binaryData.Bytes(), time.Now().Unix())
		// fmt.Println(u)
		if ok = h.db.UpdateData(u); ok {
			log.Println("offmsg insert success")
		}
	}
	return true

}

func (h *Handler) askNotifyMessage(cid string, reqData map[string]interface{}) bool {
	log.Println(reqData)
	if phone, ok := reqData["phone"].(string); ok {
		if msgs, ok := reqData["msglist"].([]map[string]interface{}); ok {
			for _, msg := range msgs {
				notify := map[string]interface{}{}
				notify["msglist"] = []map[string]interface{}{}
				msgid, ok := msg["msgid"].(string)
				if ok == false {
					log.Println("msgid not exist")
					return false
				}
				to, ok := msg["to"].(string)
				if ok == false {
					log.Println("to not exist")
					return false
				}
				msgtype, ok := msg["msgid"].(int)
				if ok == false {
					log.Println("msgtype not exist")
					return false
				}
				switch msgtype {
				case 0x0001:
					//C2C
					msginfo := map[string]interface{}{
						"msgid": strconv.FormatInt(time.Now().Unix(), 10) + cid,
					}
					msginfo["recylist"] = []map[string]interface{}{}

					msginfo["recylist"] = append(msginfo["recylist"].([]map[string]interface{}), map[string]interface{}{
						"msgid": msgid,
						"to":    phone,
					})
					binaryData := h.msf.BsonData.TransFromMap(&msginfo)

					notify["msglist"] = append(notify["msglist"].([]map[string]interface{}), map[string]interface{}{
						"msgtype": 0x000A,
						"msginfo": binaryData.Bytes(),
					})

					if ok := h.msf.SessionMaster.GetPhoneOnline(to); ok {
						//在线
						log.Println(to, "is online")
						h.msf.SessionMaster.WriteByPhone(to, h.msf.BsonData.Set(notify, notifyMsg))
					} else {
						//离线库
						log.Println(to, "is not online")
						u := fmt.Sprintf("INSERT INTO toffmsg(fPhone,fMsgId,fMsgType,fMsgInfo,fCreateTime) VALUES('%s','%s',%d,'%s',FROM_UNIXTIME(%d))", to, msginfo["msgid"], 0x000A, binaryData.Bytes(), time.Now().Unix())
						fmt.Println(u)
						if ok = h.db.UpdateData(u); ok {
							log.Println("offmsg insert success")
						}
					}
				default:
					log.Println("msgtype error")
				}
			}
		}
	}

	return false
}

func (h *Handler) offMessage(cid string, reqData map[string]interface{}) bool {
	log.Println(reqData)
	if phone, ok := reqData["phone"].(string); ok {
		msglist, ok := reqData["msglist"].([]map[string]interface{})
		if ok == false {
			log.Println("msglist not exist")
			return false
		}
		notify := map[string]interface{}{}
		notify["msglist"] = []map[string]interface{}{}
		if len(msglist) != 0 {
			for _, msg := range msglist {
				if msgid, ok := msg["msgid"].(string); ok {
					del := fmt.Sprintf("DELETE FROM toffmsg WHERE fphone='%s' AND fMsgId= '%s'", phone, msgid)
					if ok = h.db.UpdateData(del); ok {
						log.Println("offmsg delete success")
					}
				}
			}
		}
		q := fmt.Sprintf("SELECT fMsgType ,fMsgInfo FROM toffmsg WHERE fPhone = '%s' LIMIT 0,5", phone)
		rows, err := h.db.Query(q)
		if err != nil {
			log.Printf("Query failed,err:%v", err)
			return false
		}
		for rows.Next() {
			var msgtype int
			var msginfo []byte
			err = rows.Scan(&msgtype, &msginfo)
			if err != nil {
				fmt.Printf("Scan failed,err:%v", err)
				return false
			}
			notify["msglist"] = append(notify["msglist"].([]map[string]interface{}), map[string]interface{}{
				"msgtype": msgtype,
				"msginfo": msginfo,
			})
		}
		h.msf.SessionMaster.WriteByCID(cid, h.msf.BsonData.Set(notify, rspOffmsg))
		return true
	}
	return false
}
