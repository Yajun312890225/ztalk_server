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
var ut = utils.NewUtils()
var rootHandler = handler.NewHandler(ser, db, ut)

func main() {
	ser.Listening()

}