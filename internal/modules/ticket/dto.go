package ticket

type TicketResponse struct {
	Ticket string `json:"ticket"`
}

type TicketPayload struct {
	UserID string `json:"user_id"`
	JWT    string `json:"jwt"`
}
