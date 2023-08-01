package internal

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
)

var Db *sql.DB

func InitDb() {
	var err error

	Db, err = sql.Open("sqlite3", "./cube.db")
	if err != nil {
		panic(err)
	}

	_, err = Db.Exec(`
		create table if not exists source (
			name varchar(64) not null,
			type varchar(16) not null,
			lang varchar(16) not null,
			content text not null,
			compiled text not null default '',
			active boolean not null default false,
			method varchar(8) not null default '',
			url varchar(64) not null default '',
			cron varchar(16) not null default '',
			primary key(name, type)
		);
	`)
	if err != nil {
		panic(err)
	}
}
