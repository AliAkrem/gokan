package message

type MessageResponse struct {
	MsgID          string                 `json:"msg_id"`
	ClientMsgID    string                 `json:"client_msg_id"`
	RoomID         string                 `json:"room_id"`
	AuthorID       string                 `json:"author_id"`
	Text           *string                `json:"text,omitempty"`
	Type           string                 `json:"type"`
	Status         string                 `json:"status"`
	RepliedMessage map[string]interface{} `json:"replied_message,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt      int64                  `json:"created_at"`
	UpdatedAt      int64                  `json:"updated_at"`
	DeletedAt      *int64                 `json:"deleted_at,omitempty"`
}

type MessageListResponse struct {
	Messages []MessageResponse `json:"messages"`
}

type DeleteMessageResponse struct {
	Success bool `json:"success"`
}

type MessageDeliveredEvent struct {
	MsgID  string `json:"msg_id"`
	RoomID string `json:"room_id"`
}

type MessageReadEvent struct {
	MsgID  string `json:"msg_id"`
	RoomID string `json:"room_id"`
}
