package models

import (
	"github.com/dgrijalva/jwt-go"
	"gorm.io/gorm"
)

// User model
type User struct {
	gorm.Model
	Name string `json:"name"`
	Email string `json:"email" gorm:"type:varchar(100);unique_index"`
	Password string `json:"password"`
}

// Video model
type Viedo struct {
	gorm.Model
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
