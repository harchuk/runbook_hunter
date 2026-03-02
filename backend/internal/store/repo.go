package store

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Repository struct {
	db     *gorm.DB
	driver string
}

func Open(driver, dsn string) (*Repository, error) {
	var (
		db  *gorm.DB
		err error
	)

	gormCfg := &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)}
	switch driver {
	case "postgres":
		db, err = gorm.Open(postgres.Open(dsn), gormCfg)
	case "sqlite":
		db, err = gorm.Open(sqlite.Open(dsn), gormCfg)
	default:
		return nil, fmt.Errorf("unsupported driver: %s", driver)
	}
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetConnMaxIdleTime(5 * time.Minute)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(30)

	return &Repository{db: db, driver: driver}, nil
}

func RunMigrations(driver, dsn, migrationsPath string) error {
	if driver != "postgres" {
		// Security notes: non-production sqlite mode relies on explicit schema bootstrapping only for local runs.
		return nil
	}

	absPath, err := resolveMigrationsPath(migrationsPath)
	if err != nil {
		return err
	}

	m, err := migrate.New("file://"+absPath, dsn)
	if err != nil {
		return err
	}
	defer func() {
		_, _ = m.Close()
	}()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}

func resolveMigrationsPath(path string) (string, error) {
	if path == "" {
		path = "internal/store/migrations"
	}
	if strings.HasPrefix(path, "/") {
		return path, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/%s", cwd, strings.TrimPrefix(path, "./")), nil
}

func (r *Repository) DB() *gorm.DB {
	return r.db
}

func (r *Repository) Ping(ctx context.Context) error {
	sqlDB, err := r.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

func (r *Repository) EnsureLocalSchema() error {
	return AutoMigrate(r.db)
}

func (r *Repository) Close() error {
	sqlDB, err := r.db.DB()
	if err != nil {
		return err
	}
	if sqlDB == nil {
		return fmt.Errorf("sql db unavailable")
	}
	return sqlDB.Close()
}
