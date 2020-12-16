package models

import (
	"time"

	"github.com/dgrijalva/jwt-go"
	"gorm.io/gorm"
)

// User model
type User struct {
	ID uint						`gorm:"primaryKey"`
	Name string 				`json:"name"`
	Email string 				`json:"email" gorm:"unique_index"`
	Password string 			`json:"password"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt 	`gorm:"index"`
}

// Video model
type Viedo struct {
	ID uint						`gorm:"primaryKey"`

	UserID uint
	Title string

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt 	`gorm:"index"`
}

// ErrResponse - Error response
type ErrResponse struct {
	Error string `json:"error"`
}

// CustomClaims custom claims
type CustomClaims struct {
	UserID uint
	Name string
	Email string
	jwt.StandardClaims
}
