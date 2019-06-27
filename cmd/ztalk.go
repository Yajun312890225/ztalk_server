package main

import (
	"ztalk_server/internal/database"
	"ztalk_server/internal/handler"
	"ztalk_server/internal/server"
	"ztalk_server/internal/utils"

	_ "github.com/go-sql-driver/mysql"
)

var ser = server.NewMsf()
var db = database.NewDB()
var redisConn = database.NewRedis()
var ut = utils.NewUtils()
var rootHandler = handler.NewHandler(ser, db, ut, redisConn)

func main() {
	ser.Listening()

	// redisData := &rp.Friend{
	// 	UserID:   proto.Int64(8),
	// 	Contact:  proto.Bool(false),
	// 	Reg:      proto.Bool(true),
	// 	LastTime: proto.String("111231231231231231231231231231"),
	// }
	// data, _ := proto.Marshal(redisData)
	// _, err := (*redisConn).Do("HSET", "test", "proto", data)
	// if err != nil {
	// 	log.Println("redis HGET error:", err)
	// }

	// data, err := redis.ByteSlices((*redisConn).Do("HGETALL", "ZU_+8617600113331"))
	// if err != nil {
	// 	log.Println("redis HGET error:", err)
	// }
	// fmt.Println(string(data[0]))
	// fmt.Println(string(data[1]))

}
