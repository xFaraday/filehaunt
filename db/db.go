package db

import (
	"database/sql"
	"os"

	"go.uber.org/zap"
)

var (
	dbname string = "fh.db"
)

func DbConnect() *sql.DB {
	db, err := sql.Open("sqlite3", dbname)
	if err != nil {
		zap.S().Fatal("Database connection failed! Info: ", err)
	}
	return db
}

func DbInit() {
	_, err := os.Stat(dbname)
	if os.IsNotExist(err) {
		zap.S().Info("Database not found. Creating database...")
		os.Create(dbname)
	} else {
		zap.S().Info("Database exists...")
	}

	conn := DbConnect()
	defer conn.Close()

	sqlCmds := `
	create table if not exists fileindex (filepath text not null primary key, name text not null, backupfile text not null, backuptime text not null, hash text not null);
	create table if not exists filechanges (filepath text not null primary key, timeofchange text not null, change text not null, changehash text not null);
	create table if not exists dirindex (dirpath text not null, name text not null)
	`
	res, err := conn.Exec(sqlCmds)
	if err != nil {
		zap.S().Fatal("Unable to create database tables...")
	} else {
		zap.S().Info(res)
	}
}

func InsertIntoTable(tablename string) (stmt *sql.Stmt) {
	conn := DbConnect()

	switch tablename {
	case "fileindex":
		operation := "INSERT INTO fileindex (filepath, name, backupfile, backuptime, hash) VALUES (?, ?, ?, ?, ?)"
		stmt, _ = conn.Prepare(operation)
		return stmt
	case "filechanges":
		operation := "INSERT INTO filechanges (filepath, timeofchange, change, changehash) VALUES (?, ?, ?, ?)"
		stmt, _ = conn.Prepare(operation)
		return stmt
	case "dirindex":
		operation := "INSERT INTO dirindex (dirpath, name) VALUES (?, ?)"
		stmt, _ = conn.Prepare(operation)
		return stmt
	default:
		return stmt
	}
}

func CountRows(tablename string) (num int) {
	conn := DbConnect()

	switch tablename {
	case "fileindex":
		query := "SELECT COUNT(filepath) FROM fileindex"
		//err := conn.QueryRow(query).Scan(&num)
		if err := conn.QueryRow(query).Scan(&num); err != nil {
			zap.S().Warn("Unable to get count of table: fileindex")
		}
		return num
	case "filechanges":
		query := "SELECT COUNT(filepath) FROM filechanges"
		//err := conn.QueryRow(query).Scan(&num)
		if err := conn.QueryRow(query).Scan(&num); err != nil {
			zap.S().Warn("Unable to get count of table: filechanges")
		}
		return num
	case "dirindex":
		query := "SELECT COUNT(dirpath) FROM dirindex"
		//err := conn.QueryRow(query).Scan(&num)
		if err := conn.QueryRow(query).Scan(&num); err != nil {
			zap.S().Warn("Unable to get count of table: dirindex")
		}
		return num
	default:
		return num
	}
}

/*
func SelectFromDb(tablename string) (stmt *sql.Stmt) {
	conn := DbConnect()

	switch tablename {
	case "fileindex":

	}
}
*/
