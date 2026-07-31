// Package integration MySQL 辅助工具
package integration

import (
	"database/sql"

	_ "github.com/go-sql-driver/mysql"
)

// openMySQL 打开 MySQL 连接
// 使用 go-sql-driver/mysql 驱动，DSN 格式兼容 Java 侧 jdbc URL
//
// 参数：
//   - dsn: Go 格式的 DSN，如 user:pass@tcp(host:port)/db?charset=utf8mb4
//
// 返回：*sql.DB 实例
func openMySQL(dsn string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	// 验证连接
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	// 连接池配置
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	return db, nil
}
