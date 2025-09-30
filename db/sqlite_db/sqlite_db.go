package sqlite_db

import (
	"fmt"
	"github.com/jmoiron/sqlx"
	"log"
	//_ "github.com/mutecomm/go-sqlcipher/v4"
	_ "modernc.org/sqlite"
	"time"
)

var db *sqlx.DB

func Db() *sqlx.DB {
	return db
}
func InitDB(filepath string) error {
	var err error
	//key := url.QueryEscape("test")
	//dbname := fmt.Sprintf("%s?_pragma_key=%s&_pragma_cipher_page_size=4096", filepath, key)
	dbname := filepath
	db, err = sqlx.Connect("sqlite", dbname)
	if err != nil {
		log.Println("Error opening database:", err)
		return err
	}

	// 设置连接池
	db.SetConnMaxLifetime(4 * time.Hour)
	db.SetMaxOpenConns(200)
	db.SetMaxIdleConns(100)
	err = db.Ping()
	if err != nil {
		log.Fatalf("数据库连接失败ping:%v", err)
	}
	//_, err = db.Exec("PRAGMA journal_mode=WAL;")
	//if err != nil {
	//	log.Fatal(err)
	//}
	return nil
}
func CloseDB() {
	db.Close()
}

type Scanner interface {
	Scan(dest ...interface{}) error
}

func QueryAll[T any](query string, destSlice *[]T, scanFunc func(row Scanner) T) error {
	rows, err := db.Query(query)
	if err != nil {
		return fmt.Errorf("error executing query: %v", err)
	}
	defer rows.Close()
	//columns, err := rows.Columns()
	//if err != nil {
	//	log.Println(err, "Columns Error")
	//	return err
	//}
	var result []T
	for rows.Next() {
		tValue := scanFunc(rows)
		result = append(result, tValue)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating over rows: %v", err)
	}

	*destSlice = result

	return nil
}
