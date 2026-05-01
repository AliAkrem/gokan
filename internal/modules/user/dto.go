package user

type CreateUserRequest struct {
	UserID    string                 `json:"user_id" validate:"required"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	PublicKey *string                `json:"publicKey,omitempty"`
}

type UpdateUserRequest struct {
	PublicKey string `json:"publicKey" validate:"required"`
}

type UserResponse struct {
	UserID     string                 `json:"user_id"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	PublicKey  *string                `json:"publicKey,omitempty"`
	SyncedAt   *int64                 `json:"synced_at,omitempty"`
	CreatedAt  int64                  `json:"created_at"`
	UpdatedAt  int64                  `json:"updated_at"`
	LastSeenAt int64                  `json:"last_seen_at"`
}

type PublicKeyResponse struct {
	PublicKey *string `json:"publicKey"`
}

type UserListResponse struct {
	Users []UserResponse `json:"users"`
}
