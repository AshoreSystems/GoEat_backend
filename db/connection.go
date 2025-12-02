package db

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

var DB *sql.DB

func Connect() {
	var err error

	// 👇 No username or password — just connect to local MySQL
	dsn := "root:@tcp(127.0.0.1:3306)/osm_db"

	DB, err = sql.Open("mysql", dsn)
	if err != nil {
		panic(err)
	}

	err = DB.Ping()
	if err != nil {
		panic(err)
	}

	fmt.Println("✅ Connected to MySQL Database (osm_db)")
}
