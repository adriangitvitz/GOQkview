package repositories

import (
	"fmt"
	"os"

	"goqkview/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type PostgresConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Database string
	SSLMode  string
}

func ConfigFromEnv() PostgresConfig {
	sslMode := os.Getenv("POSTGRES_SSLMODE")
	if sslMode == "" {
		sslMode = "disable"
	}
	return PostgresConfig{
		Host:     os.Getenv("POSTGRES_HOST"),
		Port:     os.Getenv("POSTGRES_PORT"),
		User:     os.Getenv("POSTGRES_USER"),
		Password: os.Getenv("POSTGRES_PASSWORD"),
		Database: os.Getenv("POSTGRES_DB"),
		SSLMode:  sslMode,
	}
}

type PostgresDB struct {
	db *gorm.DB
}

func NewPostgresDB(cfg PostgresConfig) (*PostgresDB, error) {
	dsn := fmt.Sprintf("user=%s password=%s dbname=%s port=%s sslmode=%s host=%s",
		cfg.User, cfg.Password, cfg.Database, cfg.Port, cfg.SSLMode, cfg.Host)

	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN:                  dsn,
		PreferSimpleProtocol: true,
	}))
	if err != nil {
		return nil, fmt.Errorf("postgres: connection failed: %w", err)
	}

	return &PostgresDB{db: db}, nil
}

func (p *PostgresDB) FindUnprocessedUpload(tag, uploadType string) (*models.Upload, error) {
	var upload models.Upload
	result := p.db.Where(&models.Upload{Tag: tag, Type: uploadType, Processed: false}).First(&upload)
	if result.Error != nil {
		return nil, result.Error
	}
	return &upload, nil
}

func (p *PostgresDB) MarkProcessed(tag string) error {
	result := p.db.Model(&models.Upload{}).Where(&models.Upload{Tag: tag}).Update("processed", true)
	return result.Error
}

func (p *PostgresDB) Close() error {
	sqlDB, err := p.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
