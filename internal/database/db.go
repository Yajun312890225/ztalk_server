package database

import (
	"database/sql"
	"fmt"
	"log"
)

const (
	user     = "root"
	password = "123456"
	net      = "tcp"
	server   = "192.168.0.234"
	port     = 3306
	data     = "ztalk_reg"
)

//DB db
type DB struct {
	db *sql.DB
}

//NewDB init
func NewDB() *DB {
	dsn := fmt.Sprintf("%s:%s@%s(%s:%d)/%s", user, password, net, server, port, data)
	d := &DB{}
	var err error
	d.db, err = sql.Open("mysql", dsn)
	if err != nil {
		fmt.Printf("Open mysql failed,err:%v\n", err)
		return nil
	}
	return d
}

//QueryOne query
func (d *DB) QueryOne(quertString string) *sql.Row {
	return d.db.QueryRow(quertString)
}

//UpdateData update
func (d *DB) UpdateData(execString string) bool {
	result, err := d.db.Exec(execString)
	if err != nil {
		log.Printf("Insert failed,err:%v", err)
		return false
	}
	rowsaffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("Get RowsAffected failed,err:%v", err)
		return false
	}
	log.Println("RowsAffected:", rowsaffected)
	return true
}
