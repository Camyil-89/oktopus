package view

type userResponse struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	Protected bool   `json:"protected"`
	Enabled   bool   `json:"enabled"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type listResponse struct {
	Count    int64          `json:"count"`
	Page     int            `json:"page"`
	PageSize int            `json:"page_size"`
	Results  []userResponse `json:"results"`
}

type createUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type changePasswordRequest struct {
	Password string `json:"password"`
}

type setEnabledRequest struct {
	Enabled bool `json:"enabled"`
}
