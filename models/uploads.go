package models

import "time"

type Upload struct {
	ID         int64     `gorm:"primarykey"`
	Filename   string    `gorm:"column:filename"`
	Bucket     string    `gorm:"column:bucket"`
	Size       int64     `gorm:"column:size"`
	Processed  bool      `gorm:"column:processed"`
	Uploadtime time.Time `gorm:"column:uploadtime"`
	Tag        string    `gorm:"column:uuidtag"`
	Type       string    `gorm:"column:type"`
}

func (Upload) TableName() string {
	return "uploads"
}
