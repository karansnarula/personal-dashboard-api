package response

type Registered struct {
	ID    int64  `json:"id"`
	Email string `json:"email"`
}

type Token struct {
	Token     string `json:"token"`
	TokenType string `json:"token_type"`
	ExpiresIn int64  `json:"expires_in"` // seconds
}

type Message struct {
	Message string `json:"message"`
}
