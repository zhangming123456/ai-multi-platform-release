package services

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/beego/beego/v2/client/orm"
	_ "github.com/mattn/go-sqlite3"

	"ai-multi-platform-release/backend-go/models"
)

var defaultOrm orm.Ormer
var defaultDBPath string

func GetOrm() orm.Ormer {
	return defaultOrm
}

func GetUploadDir() string {
	if defaultDBPath == "" {
		return "uploads"
	}
	return filepath.Join(filepath.Dir(defaultDBPath), "uploads")
}

func GetSQLDB() (*sql.DB, error) {
	if defaultOrm == nil {
		return nil, errors.New("数据库未初始化")
	}
	return orm.GetDB()
}

func InitDatabase(dbPath string) error {
	models.InitModels()
	if dbPath == "" {
		dbPath = filepath.Join("backend-go", "app.db")
	}
	abs, err := filepath.Abs(dbPath)
	if err != nil {
		return err
	}
	defaultDBPath = abs
	if dir := filepath.Dir(abs); dir != "" {
		os.MkdirAll(dir, 0o755)
	}
	if err := orm.RegisterDataBase("default", "sqlite3", abs); err != nil {
		return err
	}
	orm.DefaultTimeLoc = time.Local
	if err := orm.RunSyncdb("default", false, true); err != nil {
		return err
	}
	defaultOrm = orm.NewOrm()
	return migrateSchema()
}

func migrateSchema() error {
	db, err := GetSQLDB()
	if err != nil {
		return err
	}
	type migration struct {
		table   string
		column  string
		ddl     string
	}
	migrations := []migration{
		{"inspections", "template_id", "ALTER TABLE inspections ADD COLUMN template_id varchar(36) DEFAULT ''"},
		{"inspections", "template_name", "ALTER TABLE inspections ADD COLUMN template_name varchar(200) DEFAULT ''"},
		{"inspection_scores", "standard", "ALTER TABLE inspection_scores ADD COLUMN standard text"},
		{"inspection_scores", "standard_image", "ALTER TABLE inspection_scores ADD COLUMN standard_image varchar(500) DEFAULT ''"},
		{"inspection_scores", "score_type", "ALTER TABLE inspection_scores ADD COLUMN score_type varchar(20) DEFAULT 'score'"},
		{"inspection_scores", "score_options", "ALTER TABLE inspection_scores ADD COLUMN score_options text"},
		{"inspection_scores", "require_remark", "ALTER TABLE inspection_scores ADD COLUMN require_remark bool DEFAULT false"},
		{"inspection_scores", "require_photo", "ALTER TABLE inspection_scores ADD COLUMN require_photo bool DEFAULT false"},
		{"inspection_scores", "show_remark", "ALTER TABLE inspection_scores ADD COLUMN show_remark bool DEFAULT true"},
		{"inspection_scores", "show_photo", "ALTER TABLE inspection_scores ADD COLUMN show_photo bool DEFAULT true"},
		{"inspection_scores", "photos", "ALTER TABLE inspection_scores ADD COLUMN photos text"},
		{"inspection_scores", "ai_suggestion", "ALTER TABLE inspection_scores ADD COLUMN ai_suggestion text"},
		{"inspection_template_items", "score_options", "ALTER TABLE inspection_template_items ADD COLUMN score_options text"},
		{"inspection_template_items", "category", "ALTER TABLE inspection_template_items ADD COLUMN category varchar(100) DEFAULT ''"},
		{"inspection_template_items", "show_remark", "ALTER TABLE inspection_template_items ADD COLUMN show_remark bool DEFAULT true"},
		{"inspection_template_items", "show_photo", "ALTER TABLE inspection_template_items ADD COLUMN show_photo bool DEFAULT true"},
		{"inspection_templates", "scoring_mode", "ALTER TABLE inspection_templates ADD COLUMN scoring_mode varchar(20) DEFAULT 'additive'"},
		{"inspection_materials", "category", "ALTER TABLE inspection_materials ADD COLUMN category varchar(100) DEFAULT ''"},
		{"inspection_materials", "title", "ALTER TABLE inspection_materials ADD COLUMN title varchar(200) DEFAULT ''"},
		{"inspection_materials", "standard", "ALTER TABLE inspection_materials ADD COLUMN standard text"},
		{"inspection_materials", "standard_image", "ALTER TABLE inspection_materials ADD COLUMN standard_image varchar(500) DEFAULT ''"},
		{"inspection_materials", "score_type", "ALTER TABLE inspection_materials ADD COLUMN score_type varchar(20) DEFAULT 'score'"},
		{"inspection_materials", "max_score", "ALTER TABLE inspection_materials ADD COLUMN max_score int DEFAULT 0"},
		{"inspection_materials", "score_options", "ALTER TABLE inspection_materials ADD COLUMN score_options text"},
		{"inspection_materials", "is_active", "ALTER TABLE inspection_materials ADD COLUMN is_active bool DEFAULT true"},
		{"inspection_materials", "created_at", "ALTER TABLE inspection_materials ADD COLUMN created_at datetime"},
		{"inspection_materials", "updated_at", "ALTER TABLE inspection_materials ADD COLUMN updated_at datetime"},
		{"materials", "name", "ALTER TABLE materials ADD COLUMN name varchar(200) DEFAULT ''"},
		{"materials", "url", "ALTER TABLE materials ADD COLUMN url varchar(500) DEFAULT ''"},
		{"materials", "type", "ALTER TABLE materials ADD COLUMN type varchar(20) DEFAULT 'image'"},
		{"materials", "category", "ALTER TABLE materials ADD COLUMN category varchar(100) DEFAULT ''"},
		{"materials", "is_active", "ALTER TABLE materials ADD COLUMN is_active bool DEFAULT true"},
		{"materials", "created_at", "ALTER TABLE materials ADD COLUMN created_at datetime"},
		{"materials", "updated_at", "ALTER TABLE materials ADD COLUMN updated_at datetime"},
		{"stores", "manager_id", "ALTER TABLE stores ADD COLUMN manager_id varchar(36) DEFAULT ''"},
		{"notifications", "channel", "ALTER TABLE notifications ADD COLUMN channel varchar(20) DEFAULT 'internal'"},
		{"inspections", "ai_summary", "ALTER TABLE inspections ADD COLUMN ai_summary text"},
		{"inspection_task_items", "ai_problem_desc", "ALTER TABLE inspection_task_items ADD COLUMN ai_problem_desc text"},
	}
	for _, m := range migrations {
		exists, err := columnExists(db, m.table, m.column)
		if err != nil {
			return err
		}
		if exists {
			continue
		}
		if _, err := db.Exec(m.ddl); err != nil {
			return err
		}
	}
	if err := migrateScoreTenSystem(db); err != nil {
		return err
	}
	return nil
}

func migrateScoreTenSystem(db *sql.DB) error {
	statements := []string{
		"UPDATE inspection_template_items SET max_score = 5 WHERE max_score <> 5 AND max_score > 0",
		"UPDATE inspection_items SET max_score = 5 WHERE max_score <> 5 AND max_score > 0",
		"UPDATE inspection_template_items SET score_options = '[{\"score\":0,\"label\":\"0分\"},{\"score\":2,\"label\":\"2分\"},{\"score\":5,\"label\":\"5分\"}]' WHERE score_type = 'score' AND (score_options IS NULL OR score_options = '' OR score_options = 'null')",
		"UPDATE inspection_template_items SET score_options = '[{\"score\":0,\"label\":\"0分\"},{\"score\":2,\"label\":\"2分\"},{\"score\":5,\"label\":\"5分\"}]' WHERE score_type = 'score' AND score_options LIKE '%\"score\":10%'",
		"UPDATE inspection_template_items SET score_options = '[{\"score\":1,\"label\":\"合格\"},{\"score\":2,\"label\":\"不合格\"}]' WHERE score_type = 'pass_fail' AND (score_options IS NULL OR score_options = '' OR score_options = 'null')",
		"UPDATE inspection_template_items SET score_options = '[{\"score\":1,\"label\":\"合格\"},{\"score\":2,\"label\":\"不合格\"}]' WHERE score_type = 'pass_fail' AND score_options NOT LIKE '%\"score\":1%'",
		"UPDATE inspection_scores SET score_options = '[{\"score\":1,\"label\":\"合格\"},{\"score\":2,\"label\":\"不合格\"}]' WHERE score_type = 'pass_fail' AND (score_options IS NULL OR score_options = '' OR score_options = 'null')",
		"UPDATE inspection_scores SET score = 1 WHERE score_type = 'pass_fail' AND score = 10",
		"UPDATE inspection_scores SET score = 2 WHERE score_type = 'pass_fail' AND score = 0",
	}
	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

func columnExists(db *sql.DB, table, column string) (bool, error) {
	rows, err := db.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull int
		var dflt interface{}
		var pk int
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}
	return false, rows.Err()
}

func ParseTime(s string) (*time.Time, error) {
	if s == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t2, err2 := time.Parse("2006-01-02 15:04:05", s)
		if err2 != nil {
			return nil, err
		}
		return &t2, nil
	}
	return &t, nil
}

func FormatTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(time.RFC3339)
}
