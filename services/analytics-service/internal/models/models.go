package models

import (
	"time"
	"gorm.io/gorm"
)

// Example model - replace with actual models
type Example struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	Name      string         `json:"name" gorm:"size:255;not null"`
	Status    string         `json:"status" gorm:"size:50;default:'active'"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (Example) TableName() string {
	return "analytics_service_examples"
}
