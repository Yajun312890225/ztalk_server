package main

import (
	"fmt"
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
	// ser.Listening()

	res, err := (*redisConn).Do("HMGET", "ZU_+8617600113331", "passwd", "nonce")
	if err != nil {
		fmt.Println("redis HGET error:", err)
	} else {
		for _, v := range res.([]interface{}) {
			fmt.Printf("%s\n", v)
		}
	}

}
