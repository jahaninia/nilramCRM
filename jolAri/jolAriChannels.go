package jolAri

type Channel struct {
	Id     string `json:"id"`
	Name   string `json:"name"`
	State  string `json:"state"`
	Caller struct {
		Name   string `json:"name"`
		Number string `json:"number"`
	} `json:"caller"`
	Connected struct {
		Name   string `json:"name"`
		Number string `json:"number"`
	} `json:"connected"`
	Accountcode string `json:"accountcode"`
	Dialplan    struct {
		Context  string `json:"context"`
		Exten    string `json:"exten"`
		Priority int    `json:"priority"`
	} `json:"dialplan"`
	Creationtime string `json:"creationtime"`
	Language     string `json:"language"`
}

type OnlineChannels struct {
	Channels []Channel
}
