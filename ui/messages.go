package ui

type Message struct {
	Message string
	timeout float64
}

func NewMessage(message string, timeoutInSeconds float64) Message {
	return Message{Message: message, timeout: timeoutInSeconds}
}
