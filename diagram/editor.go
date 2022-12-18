package diagram

type InitialMessage struct {
	Type     string   `json:"type"`
	Clients  []string `json:"clients"`
	ClientId string   `json:"clientId"`
}

type Vector3 struct {
	X float32 `json:"x"`
	Y float32 `json:"y"`
	Z float32 `json:"z"`
}

type Message struct {
	ClientId string  `json:"clientId"`
	Type     string  `json:"type"`
	Position Vector3 `json:"position"`
}
