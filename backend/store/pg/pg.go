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
	if err := ensureUpgradedColumns(db); err != nil {
		return nil, err
	}

	return &DB{DB: db}, nil
}

// ensureUpgradedColumns 为不走增量迁移脚本的存量库幂等补齐新增关键列与索引，
// 避免升级后接口因缺列报错。
func ensureUpgradedColumns(db *gorm.DB) error {
	var hasAPITokens bool
	if err := db.Raw(`
SELECT EXISTS (
	SELECT 1 FROM information_schema.tables
	WHERE table_schema = 'public' AND table_name = 'api_tokens'
)`).Scan(&hasAPITokens).Error; err != nil {
		return fmt.Errorf("check api_tokens table failed: %w", err)
	}
	if !hasAPITokens {
		return nil
	}

	var hasTokenIndex bool
	if err := db.Raw(`
SELECT EXISTS (
	SELECT 1 FROM information_schema.columns
	WHERE table_schema = 'public' AND table_name = 'api_tokens' AND column_name = 'token_index'
)`).Scan(&hasTokenIndex).Error; err != nil {
		return fmt.Errorf("check api_tokens.token_index column failed: %w", err)
	}
	if hasTokenIndex {
		return nil
	}

	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`ALTER TABLE api_tokens ADD COLUMN IF NOT EXISTS token_index TEXT NOT NULL DEFAULT ''`).Error; err != nil {
			return fmt.Errorf("add api_tokens.token_index column failed: %w", err)
		}
		// 存量行 token 均为明文且唯一，回填后建唯一索引不冲突；
		// 若配置了 ENCRYPTION_KEY，后续 ap_token.go 懒迁移会将其升级为 HMAC 索引+密文。
		if err := tx.Exec(`UPDATE api_tokens SET token_index = token WHERE token_index = ''`).Error; err != nil {
			return fmt.Errorf("backfill api_tokens.token_index failed: %w", err)
		}
		if err := tx.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS uniq_api_tokens_token_index ON api_tokens(token_index)`).Error; err != nil {
			return fmt.Errorf("create uniq_api_tokens_token_index failed: %w", err)
		}
		return nil
	})
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
	// 老库可能只有旧表而缺 pgvector 相关表（rag_chunks/rag_jobs），
	// 只检查 knowledge_bases 会让缺失延迟到问答时才暴露
	for _, table := range []string{"rag_chunks", "rag_jobs"} {
		var ok bool
		if err := db.Raw(`
SELECT EXISTS (
	SELECT 1 FROM information_schema.tables
	WHERE table_schema = 'public' AND table_name = ?
)`, table).Scan(&ok).Error; err != nil {
			return fmt.Errorf("check database schema failed: %w", err)
		}
		if !ok {
			return fmt.Errorf("database schema is outdated: table %q is missing — the stack-slim schema requires a fresh deploy (import backend/store/pg/migration/full_fresh_deploy.sql)", table)
		}
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
