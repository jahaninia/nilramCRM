package jolConfigurtion

import (
	"encoding/json"
	"os"
)

type Configurtion struct {
	Asterisk struct {
		HostAmi     string `json:"hostAmi"`
		PortAmi     string `json:"portAmi"`
		UsernameAmi string `json:"usernameAmi"`
		PasswordAmi string `json:"passwordAmi"`
	} `json:"asterisk"`
	Token      string `json:"token"`
	UrlCall    string `json:"urlCall"`
	UrlAnswer  string `json:"urlAnswer"`
	UrlReject  string `json:"urlReject"`
	UrlEndCall string `json:"urlEndCall"`
	Length     int    `json:"lengthExtension"`
	Debug      bool   `json:"debug"`
	Pool       int    `json:"pool"`
}

func LoadConfigFile(filename string) (Configurtion, error) {
	var config Configurtion
	Configurtion, err := os.Open(filename)

	if err != nil {
		return config, err
	}
	jsonpare := json.NewDecoder(Configurtion)
	err = jsonpare.Decode(&config)
	if err != nil {
		return config, err
	}
	return config, nil
}
