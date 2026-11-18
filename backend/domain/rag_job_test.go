package domain_test

import (
	"strings"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/chaitin/panda-wiki/domain"
)

// rag_jobs.id 是 UUID 列（DEFAULT gen_random_uuid()）。
// 若 GORM 把零值 string 主键显式插入为 ”，Postgres 会报
// "invalid input syntax for type uuid: \"\""，导致 RAG 任务入队全量失败。
// 通过 DryRun 生成 SQL 验证零值 ID 列被跳过（由 default tag 保护）。
func TestRagJobZeroIDSkippedOnInsert(t *testing.T) {
	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN:                  "host=localhost sslmode=disable",
		PreferSimpleProtocol: true,
	}), &gorm.Config{
		Logger:               logger.Default.LogMode(logger.Silent),
		DryRun:               true,
		DisableAutomaticPing: true,
	})
	if err != nil {
		t.Fatalf("open dry-run db: %v", err)
	}

	sql := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return tx.Create(&domain.RagJob{
			Payload:     []byte(`{}`),
			Status:      domain.RagJobStatusPending,
			MaxAttempts: 5,
		})
	})

	// 检查 INSERT 列清单：零值 id 必须被跳过（RETURNING "id" 是 default tag 的合法回填）
	start := strings.Index(sql, `INSERT INTO "rag_jobs" (`)
	if start < 0 {
		t.Fatalf("unexpected sql: %s", sql)
	}
	rest := sql[start+len(`INSERT INTO "rag_jobs" (`):]
	cols := rest[:strings.Index(rest, `)`)]
	if strings.Contains(cols, `"id"`) {
		t.Fatalf("zero-value id must be omitted from the insert column list so the DB default applies, columns: %s", cols)
	}
}
