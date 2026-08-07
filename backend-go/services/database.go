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

func GetOrm() orm.Ormer {
	return defaultOrm
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
	return nil
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
