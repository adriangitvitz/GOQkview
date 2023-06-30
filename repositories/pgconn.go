package repositories

import (
	"fmt"
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var POSTGRES *gorm.DB

func Connectpostgres()  {
	pgconnstr := fmt.Sprintf("user=%s password=%s dbname=%s port=%s sslmode=disable host=%s", os.Getenv("POSTGRES_USER"), os.Getenv("POSTGRES_PASSWORD"), os.Getenv("POSTGRES_DB"), os.Getenv("POSTGRES_PORT"), os.Getenv("POSTGRES_HOST"))
	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN:                  pgconnstr,
		PreferSimpleProtocol: true,
	}))
	if err != nil {
        log.Fatalf("Error connection: %s", err.Error())
	}
    POSTGRES = db
}
