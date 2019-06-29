package main

import (
	"ztalk_server/internal/database"
	"ztalk_server/internal/handler"
	"ztalk_server/internal/server"
	"ztalk_server/internal/utils"

	_ "github.com/go-sql-driver/mysql"
)

var db = database.NewDB()
var redisConn = database.NewRedis()
var ser = server.NewMsf(redisConn)
var ut = utils.NewUtils()
var rootHandler = handler.NewHandler(ser, db, ut, redisConn)

func main() {
	ser.Listening()
}
