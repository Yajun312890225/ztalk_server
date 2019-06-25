package handler

import (
	"fmt"
	"log"
	"time"
	"ztalk_server/internal/database"
	"ztalk_server/internal/server"
	"ztalk_server/internal/utils"
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
	notifymsg
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
	msf *server.Msf
	db  *database.DB
	ut  *utils.Ut
}

//NewHandler new
func NewHandler(msf *server.Msf, db *database.DB, ut *utils.Ut) *Handler {
	handler := Handler{
		msf: msf,
		db:  db,
		ut:  ut,
	}
	msf.EventPool.Register(auth, handler.authMessage)
	msf.EventPool.Register(reqC2Cmsg, handler.c2cMessage)
	msf.EventPool.Register(reqSyncContact,handler.syncContact)
	return &handler
}

func (h *Handler) authMessage(fd uint32, reqData map[string]interface{}) bool {
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

	var checksign string
	if ty == 1 {
		checksign = fmt.Sprintf("%s%s%02x%s", phone, source, []byte(password), key)
	} else {
		checksign = fmt.Sprintf("%s%s%02x%s%s", phone, source, []byte(password), nonce, key)
	}

	if md5sign != h.ut.Md5(checksign) {
		result["desc"] = "sign error"
		h.msf.SessionMaster.WriteByID(fd, h.msf.BsonData.Set(result, fail))
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
	if ty == 1 {
		h.msf.SessionMaster.WriteByID(fd, h.msf.BsonData.Set(result, challenge))
	} else if ty == 0 {
		result["time"] = int(time.Now().Unix())
		type resSt struct {
			ip   string
			port int
		}
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
		fmt.Println(result)
		h.msf.SessionMaster.SetPhoneByID(fd, phone)
		h.msf.SessionMaster.WriteByID(fd, h.msf.BsonData.Set(result, succ))
	}

	return true
}

func (h *Handler) syncContact(fd uint32, reqData map[string]interface{}) bool {
	log.Println(reqData)

	return false
}
func (h *Handler) c2cMessage(fd uint32, reqData map[string]interface{}) bool {
	log.Println(reqData)
	if phone, ok := reqData["phone"].(string); ok {
		result := make(map[string]interface{})
		result["nonce"] = "Wa hahaha"
		h.msf.SessionMaster.WriteByPhone(phone, h.msf.BsonData.Set(result, notifyMsg))
		return true
	}
	return false
}
