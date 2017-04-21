package main

import (
	"database/sql"
	"path"
	"log"
	"github.com/mattn/go-sqlite3"
)

func SetUpSchema() (dbInstance *sql.DB, error error) {
	db, err := sql.Open("sqlite3", path.Join("./data/urls.db"))
	if err != nil {
		log.Fatalf("Error Occurred in creating Schema: %s", err)
	}

	createTable := `create table ` + table_name + ` (id INTEGER PRIMARY KEY AUTOINCREMENT, hash varchar(14), longurl varchar(200))`
	_, err = db.Exec(createTable)

	if error, ok := err.(sqlite3.Error); ok {
		log.Println(error)
		return db, nil
	}
	return db, nil
}

func CountRecords() (urlCount int64) {
	return 65464654
}
