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
	State    string  `json:"state"`
	Data     string  `json:"data"`
}

type Build struct {
	Position Vector3 `json:"position"`
	Shape    int     `json:"shape"`
	Material int     `json:"material"`
}
