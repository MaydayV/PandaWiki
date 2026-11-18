package pg

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/golang-migrate/migrate/v4"
	migratePG "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/chaitin/panda-wiki/config"
)

type DB struct {
	*gorm.DB
}

func NewDB(config *config.Config) (*DB, error) {
	dsn := config.PG.DSN
	// same as gorm logger.Default, but without colorful output and ignore record not found error
	newLogger := logger.New(log.New(os.Stdout, "\r\n", log.LstdFlags), logger.Config{
		SlowThreshold:             200 * time.Millisecond,
		LogLevel:                  logger.Warn,
		IgnoreRecordNotFoundError: true,
		Colorful:                  false,
	})
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		TranslateError: true,
		Logger:         newLogger,
	})
	if err != nil {
		return nil, err
	}

	// 连接池必须显式配置：database/sql 默认 MaxOpenConns 无上限、连接永不过期，
	// API 与内嵌 worker 共用本池，SSE 并发突发时会打满 PG max_connections
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql db failed: %w", err)
	}
	sqlDB.SetMaxOpenConns(config.PG.MaxOpenConns)
	sqlDB.SetMaxIdleConns(config.PG.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(config.PG.ConnMaxLifetime) * time.Second)
	sqlDB.SetConnMaxIdleTime(10 * time.Minute)

	if err := doMigrate(dsn); err != nil {
		return nil, err
	}
	if err := ensureSchemaInitialized(db); err != nil {
		return nil, err
	}

	return &DB{DB: db}, nil
}

func ensureSchemaInitialized(db *gorm.DB) error {
	migrationFiles, err := filepath.Glob("migration/*.up.sql")
	if err != nil {
		return fmt.Errorf("scan migration files failed: %w", err)
	}
	if len(migrationFiles) > 0 {
		return nil
	}
	var exists bool
	if err := db.Raw(`
SELECT EXISTS (
	SELECT 1 FROM information_schema.tables
	WHERE table_schema = 'public' AND table_name = 'knowledge_bases'
)`).Scan(&exists).Error; err != nil {
		return fmt.Errorf("check database schema failed: %w", err)
	}
	if !exists {
		return fmt.Errorf("database not initialized: import backend/store/pg/migration/full_fresh_deploy.sql (or run docs/deploy/quickstart.sh)")
	}
	return nil
}

func doMigrate(dsn string) error {
	migrationFiles, err := filepath.Glob("migration/*.up.sql")
	if err != nil {
		return fmt.Errorf("scan migration files failed: %w", err)
	}
	// 支持仅保留完整部署 SQL 的交付方式：无增量迁移文件时跳过自动迁移。
	if len(migrationFiles) == 0 {
		return nil
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("open db failed: %w", err)
	}
	defer db.Close()
	driver, err := migratePG.WithInstance(db, &migratePG.Config{})
	if err != nil {
		return fmt.Errorf("with instance failed: %w", err)
	}
	m, err := migrate.NewWithDatabaseInstance(
		"file://migration",
		"postgres", driver)
	if err != nil {
		return fmt.Errorf("new with database instance failed: %w", err)
	}
	if err := m.Up(); err != nil {
		if err == migrate.ErrNoChange {
			return nil
		}
		return fmt.Errorf("migrate db failed: %w", err)
	}

	return nil
}
