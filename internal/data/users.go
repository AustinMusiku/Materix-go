package data

type User struct {
	Email      string `json:"email"`
	FirstName  string `json:"first_name"`
	LastName   string `json:"last_name"`
	Activated  bool   `json:"activated"`
	Avatar_url string `json:"avatar"`
	Provider   string `json:"provider"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}
