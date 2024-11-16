package models

import (
	"time"
)

type User struct {
	ID       uint      `gorm:"primaryKey;autoIncrement"`
	Name     string    `gorm:"unique;not null" json:"name"`
	Password string    `gorm:"not null" json:"-"`
	Created  time.Time `json:"created_at"`
	Updated  time.Time `json:"updated_at"`
}
