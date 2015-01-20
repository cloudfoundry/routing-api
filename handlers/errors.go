package handlers

type Error struct {
	Type    string `json:"name"`
	Message string `json:"message"`
}

func (err Error) Error() string {
	return err.Message
}

const (
	RouteInvalidError    = "RouteInvalidError"
	DBCommunicationError = "DBCommunicationError"
)
