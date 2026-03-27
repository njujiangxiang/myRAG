package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

// DB 封装 sql.DB，提供应用级辅助方法
type DB struct {
	*sql.DB
	log *zap.Logger
}

// New 创建一个新的数据库连接并执行迁移
func New(postgresURL string, log *zap.Logger) (*DB, error) {
	// 首先执行迁移
	if err := runMigrations(postgresURL); err != nil {
		return nil, fmt.Errorf("执行迁移失败：%w", err)
	}

	// 创建连接池
	db, err := sql.Open("postgres", postgresURL)
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败：%w", err)
	}

	// 配置连接池
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("连接数据库失败：%w", err)
	}

	log.Info("数据库连接已建立")

	return &DB{
		DB:  db,
		log: log,
	}, nil
}

// runMigrations 通过读取 migrations 目录的 SQL 文件执行数据库迁移
func runMigrations(postgresURL string) error {
	// 打开数据库连接
	db, err := sql.Open("postgres", postgresURL)
	if err != nil {
		return fmt.Errorf("打开数据库执行迁移失败：%w", err)
	}
	defer db.Close()

	// 查找 migrations 目录（尝试多个路径以提高灵活性）
	migrationsDir := ""
	possiblePaths := []string{"migrations", "/app/migrations", "./migrations"}
	for _, path := range possiblePaths {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			migrationsDir = path
			break
		}
	}
	if migrationsDir == "" {
		return fmt.Errorf("未找到 migrations 目录")
	}

	// 读取迁移文件
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("读取 migrations 目录失败：%w", err)
	}

	// 过滤并排序 SQL 文件
	var sqlFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			sqlFiles = append(sqlFiles, entry.Name())
		}
	}
	sort.Strings(sqlFiles)

	// 跟踪已应用的迁移（使用简单方法）
	// 检查 migrations 表是否存在
	_, err = db.Exec("SELECT 1 FROM schema_migrations LIMIT 1")
	migrationsTableExists := err == nil

	// 如果不存在，创建 migrations 表
	if !migrationsTableExists {
		_, err = db.Exec(`
			CREATE TABLE IF NOT EXISTS schema_migrations (
				version VARCHAR(255) PRIMARY KEY
			)
		`)
		if err != nil {
			return fmt.Errorf("创建 migrations 表失败：%w", err)
		}
	}

	// 应用每个迁移
	for _, sqlFile := range sqlFiles {
		// 从文件名提取版本号（例如："001_init.sql" -> "001"）
		version := strings.TrimSuffix(sqlFile, ".sql")
		if idx := strings.Index(version, "_"); idx != -1 {
			version = version[:idx]
		}

		// 检查是否已应用
		var exists bool
		err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)", version).Scan(&exists)
		if err != nil {
			return fmt.Errorf("检查迁移状态失败：%w", err)
		}
		if exists {
			continue // 已应用
		}

		// 读取并执行迁移
		migrationPath := filepath.Join(migrationsDir, sqlFile)
		migrationSQL, err := os.ReadFile(migrationPath)
		if err != nil {
			return fmt.Errorf("读取迁移文件 %s 失败：%w", sqlFile, err)
		}

		// 在事务中执行迁移
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("开始事务失败：%w", err)
		}

		_, err = tx.Exec(string(migrationSQL))
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("执行迁移 %s 失败：%w", sqlFile, err)
		}

		// 记录迁移
		_, err = tx.Exec("INSERT INTO schema_migrations (version) VALUES ($1)", version)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("记录迁移 %s 失败：%w", sqlFile, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("提交迁移 %s 失败：%w", sqlFile, err)
		}
	}

	return nil
}

// WithTenant 为行级安全设置租户上下文
func (db *DB) WithTenant(ctx context.Context, tenantID string) context.Context {
	// 这将设置 app.current_tenant 用于 RLS
	// 实现取决于如何处理租户上下文
	return ctx
}

// Close 优雅地关闭数据库连接
func (db *DB) Close() error {
	db.log.Info("关闭数据库连接")
	return db.DB.Close()
}
