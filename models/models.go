package models

import (
	"time"

	"github.com/dgrijalva/jwt-go"
	"gopkg.in/guregu/null.v3"
	"gorm.io/gorm"
)

// User model
type User struct {
	ID uint						`gorm:"primaryKey" json:"id"`
	Name string 				`json:"name"`
	Email string 				`json:"email" gorm:"unique_index"`
	Password string 			`json:"password"`
	IsAdmin bool				`gorm:"default:false" json:"isAdmin,string"`

	CreatedAt time.Time			`json:"createdAt"`
	UpdatedAt time.Time			`json:"-"`
	DeletedAt gorm.DeletedAt 	`gorm:"index" json:"-"`
}

// UserStat user statistics
type UserStat struct {
	User
	TotalViews int				`json:"totalViews"`
	NumVideos int				`json:"numVideos"`
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

	Categories []VideoCategory 	`gorm:"foreignKey:VID" json:"categories"`

	CreatedAt time.Time			`json:"createdAt"`
	UpdatedAt time.Time			`json:"-"`
	DeletedAt gorm.DeletedAt 	`gorm:"index" json:"-"`
}

// VideoCategory model
type VideoCategory struct {
	ID uint						`gorm:"primaryKey" json:"id,string,omitempty"`
	VID uint					`json:"videoId,string"`
	CID uint					`json:"categoryId,string"`
	Category Category 			`gorm:"foreignKey:ID;references:CID" json:"category"`
}

// Category model
type Category struct {
	ID uint						`gorm:"primaryKey" json:"id,string,omitempty"`
	Title string 				`gorm:"unique" json:"title"`
}

// Like model
type Like struct {
	ID uint						`gorm:"primaryKey" json:"id,string,omitempty"`
	UID uint					`json:"userId,string"`
	VID uint					`json:"videoId,string"`
	IsDislike bool				`gorm:"default:false" json:"isDislike,string"`
}

// Comment model
type Comment struct {
	ID uint						`gorm:"primaryKey" json:"id,string,omitempty"`
	UserID uint					`json:"userId"`
	User User					`json:"user"`
	VideoID uint				`json:"videoId"`
	ReplyTo null.Int			`json:"replyTo,omitempty"`
	ReplyCount int 				`gorm:"-" json:"replyCount"`
	Replies []Comment 			`gorm:"foreignKey:ReplyTo" json:"replies"`
	Text string 				`json:"text"`

	CreatedAt time.Time			`json:"createdAt"`
	UpdatedAt time.Time			`json:"-"`
	DeletedAt gorm.DeletedAt 	`gorm:"index" json:"-"`
}

// ErrResponse - Error response
type ErrResponse struct {
	Error string `json:"error"`
}

// UserClaims custom claims
type UserClaims struct {
	UserID uint
	Name string
	Email string
	Password string
	IsAdmin bool
	jwt.StandardClaims
}
