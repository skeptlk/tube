package models

import (
	"time"

	"github.com/dgrijalva/jwt-go"
	"gorm.io/gorm"
)

// User model
type User struct {
	ID uint						`gorm:"primaryKey" json:"id"`
	Name string 				`json:"name"`
	Email string 				`json:"email" gorm:"unique_index"`
	Password string 			`json:"password"`

	CreatedAt time.Time			`json:"-"`
	UpdatedAt time.Time			`json:"-"`
	DeletedAt gorm.DeletedAt 	`gorm:"index" json:"-"`
}

// Video model
type Video struct {
	ID uint						`gorm:"primaryKey" json:"id"`

	UserID uint					`json:"userId"`
	User User					`json:"user"`
	Title string				`json:"title"`
	Description string			`json:"description"`
	Duration int				`json:"duration"`
	Views int					`json:"views"`
	Likes int					`json:"likes"`
	Dislikes int				`json:"dislikes"`
	URL string					`json:"url"`
	ThumbnailURL string			`json:"thumbnail"`

	CreatedAt time.Time			`json:"-"`
	UpdatedAt time.Time			`json:"-"`
	DeletedAt gorm.DeletedAt 	`gorm:"index" json:"-"`
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
