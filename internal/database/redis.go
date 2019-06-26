package database

import (
	"fmt"

	"github.com/gomodule/redigo/redis"
)

//Redis redis
type Redis struct {
	redisConn *redis.Conn
}

//NewRedis init
func NewRedis() *redis.Conn {
	conn, err := redis.Dial("tcp", "0.0.0.0:6379")
	if err != nil {
		fmt.Println("Connect redis error :", err)
		return nil
	}
	return &conn
}
