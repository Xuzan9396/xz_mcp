package mysql_db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/Xuzan9396/zmysql/smysql"
	_ "github.com/go-sql-driver/mysql"
)

var db *smysql.MySQLClient
var rawDB *sql.DB

// ConnectionConfig MySQL连接配置
type ConnectionConfig struct {
	Username        string        `json:"username"`
	Password        string        `json:"password"`
	Addr            string        `json:"addr"`
	DatabaseName    string        `json:"database_name"`
	Debug           bool          `json:"debug,omitempty"`
	MaxOpenConns    int           `json:"max_open_conns,omitempty"`
	MaxIdleConns    int           `json:"max_idle_conns,omitempty"`
	ConnMaxLifetime time.Duration `json:"conn_max_lifetime,omitempty"`
}

// InitDB 初始化数据库连接
func InitDB(config ConnectionConfig) error {
	// 设置默认值 - 针对MCP单次调用优化
	if config.MaxOpenConns == 0 {
		config.MaxOpenConns = 5 // 降低最大连接数
	}
	if config.MaxIdleConns == 0 {
		config.MaxIdleConns = 2 // 降低空闲连接数
	}
	if config.ConnMaxLifetime == 0 {
		config.ConnMaxLifetime = 4 * time.Hour
	}

	// 构建连接选项
	opts := []func(*smysql.MySQLClient){
		smysql.WithMaxOpenConns(config.MaxOpenConns),
		smysql.WithMaxIdleConns(config.MaxIdleConns),
		smysql.WithConnMaxLifetime(config.ConnMaxLifetime),
	}

	if config.Debug {
		opts = append(opts, smysql.WithDebug())
	}

	// 创建连接
	client, err := smysql.Conn(
		config.Username,
		config.Password,
		config.Addr,
		config.DatabaseName,
		opts...,
	)
	if err != nil {
		return fmt.Errorf("failed to connect to MySQL: %v", err)
	}

	// 同时创建原始数据库连接用于多结果集处理
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local&multiStatements=true",
		config.Username, config.Password, config.Addr, config.DatabaseName)
	rawDB, err = sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("failed to create raw MySQL connection: %v", err)
	}

	// 设置连接池参数
	rawDB.SetMaxOpenConns(config.MaxOpenConns)
	rawDB.SetMaxIdleConns(config.MaxIdleConns)
	rawDB.SetConnMaxLifetime(config.ConnMaxLifetime)

	db = client
	return nil
}

// CloseDB 关闭数据库连接
func CloseDB() error {
	if rawDB != nil {
		rawDB.Close()
	}
	if db != nil {
		return db.Close()
	}
	return nil
}

// GetDB 获取数据库客户端
func GetDB() *smysql.MySQLClient {
	return db
}

// IsConnected 检查是否已连接
func IsConnected() bool {
	return db != nil
}
