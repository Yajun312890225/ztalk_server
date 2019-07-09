package main

import (
	"ztalk_server/internal/database"
	"ztalk_server/internal/handler"
	"ztalk_server/internal/server"
	"ztalk_server/internal/utils"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	db := database.NewDB()
	redisConn := database.NewRedis()
	ser := server.NewMsf(redisConn)
	ut := utils.NewUtils()
	handler.NewHandler(ser, db, ut, redisConn)
	ser.Listening()
}
