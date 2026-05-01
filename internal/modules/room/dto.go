package room

type CreateRoomRequest struct {
	Participants []string               `json:"participants" validate:"required,len=2"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

type RoomResponse struct {
	RoomID       string                 `json:"room_id"`
	Participants []string               `json:"participants"`
	Type         string                 `json:"type"`
	LastMessages map[string]interface{} `json:"last_messages,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt    int64                  `json:"created_at"`
	UpdatedAt    int64                  `json:"updated_at"`
	DeletedAt    *int64                 `json:"deleted_at,omitempty"`
}

type RoomListResponse struct {
	Rooms []RoomResponse `json:"rooms"`
}

type DeleteRoomResponse struct {
	Success bool `json:"success"`
}
