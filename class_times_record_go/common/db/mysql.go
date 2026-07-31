// Package db MySQL 数据库连接与 ORM 封装
//
// 对齐 Java MyBatis-Plus 的功能：
//   - 连接池管理（对齐 Java HikariCP）
//   - 通用 CRUD（对齐 Java BaseMapper）
//   - 数据库配置来自 Nacos common-db.yaml
//
// 依赖：
//   - github.com/go-sql-driver/mysql：MySQL 驱动
//   - database/sql：标准库 SQL 操作
//
// 配置来源（与 Java common-db.yaml 一致）：
//   host: 121.196.229.10
//   port: 3306
//   database: class_times_record
//   username: class_times_record
//   password: 8BCnbZjTT8ZxmBj6
//   charset: utf8mb4
//   timezone: Asia/Shanghai
package db

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql" // MySQL 驱动，匿名导入
)

// Config 数据库配置（对齐 Java common-db.yaml 的 spring.datasource）
type Config struct {
	Host            string        `yaml:"host" json:"host"`                     // 主机地址
	Port            int           `yaml:"port" json:"port"`                     // 端口
	Database        string        `yaml:"database" json:"database"`             // 数据库名
	Username        string        `yaml:"username" json:"username"`             // 用户名
	Password        string        `yaml:"password" json:"password"`             // 密码
	Charset         string        `yaml:"charset" json:"charset"`               // 字符集
	MaxOpenConns    int           `yaml:"maxOpenConns" json:"maxOpenConns"`     // 最大连接数
	MaxIdleConns    int           `yaml:"maxIdleConns" json:"maxIdleConns"`     // 最大空闲连接数
	ConnMaxLifetime time.Duration `yaml:"connMaxLifetime" json:"connMaxLifetime"` // 连接最大存活时间
}

// DefaultConfig 返回默认数据库配置
//
// 与 Java Nacos common-db.yaml 完全一致
func DefaultConfig() *Config {
	return &Config{
		Host:            "121.196.229.10",
		Port:            3306,
		Database:        "class_times_record",
		Username:        "class_times_record",
		Password:        "8BCnbZjTT8ZxmBj6",
		Charset:         "utf8mb4",
		MaxOpenConns:    20,
		MaxIdleConns:    10,
		ConnMaxLifetime: 30 * time.Minute,
	}
}

// DSN 构建 MySQL DSN 字符串
//
// 格式：user:password@tcp(host:port)/database?charset&parseTime&loc
func (c *Config) DSN() string {
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=true&loc=Asia%%2FShanghai",
		c.Username, c.Password,
		c.Host, c.Port,
		c.Database,
		c.Charset,
	)
}

// NewMySQL 创建 MySQL 连接池
//
// 参数：
//   - cfg: 数据库配置
//
// 返回：
//   - *sql.DB 连接池
//   - 错误信息
func NewMySQL(cfg *Config) (*sql.DB, error) {
	// 打开数据库连接（此时未真正连接，Ping 时才连接）
	db, err := sql.Open("mysql", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("打开 MySQL 连接失败: %w", err)
	}

	// 连接池配置（对齐 Java HikariCP 的 maximum-pool-size / minimum-idle）
	db.SetMaxOpenConns(cfg.MaxOpenConns)     // 最大连接数
	db.SetMaxIdleConns(cfg.MaxIdleConns)     // 最大空闲连接数
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime) // 连接最大存活时间

	// 验证连接（对齐 Java HikariCP 启动时校验）
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("MySQL Ping 失败: %w", err)
	}

	log.Printf("✅ MySQL 连接成功: %s:%d/%s", cfg.Host, cfg.Port, cfg.Database)
	return db, nil
}
