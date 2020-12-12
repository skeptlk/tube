
// User test
type User struct {
	gorm.Model
	ID int `json:"id" gorm:"unique_index"`
	Name string `json:"name"`
	Email string `json:"email" gorm:"type:varchar(100);unique_index"`
	Password string `json:"password"`
}
