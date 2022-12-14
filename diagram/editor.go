package diagram

type InitialMessage struct {
	Type     string   `json:"type"`
	Clients  []string `json:"clients"`
	ClientId string   `json:"clientId"`
}

type Vector3 struct {
	X        float32        `json:"x"`
	Y        float32        `json:"y"`
	Z        float32        `json:"z"`
}

type Message struct {
	ClientId string         `json:"clientId"`
	Type	 string			`json:"type"`
	Position Vector3		`json:"position"`
	For      string         `json:"for"`
}

type Editor struct {
	Id		string
}

func (editor *Editor) Process(msg *Message) []*Message {
	messages := []*Message{msg}

	switch msg.Type {
	case "move":

		break
	}

	return messages
}
