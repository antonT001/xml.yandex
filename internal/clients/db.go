package clients

import (
	"database/sql"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type DBClient interface {
	Db() *sql.DB
}

type dBClient struct {
	db *sql.DB
}

func NewDb(dataSourceName string) (DBClient, error) {
	db, err := sql.Open("mysql", dataSourceName)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(10)
	db.SetConnMaxIdleTime(10 * time.Second)
	db.SetConnMaxLifetime(10 * time.Second)
	return &dBClient{
		db: db,
	}, nil
}

func (d *dBClient) Db() *sql.DB {
	return d.db
}
