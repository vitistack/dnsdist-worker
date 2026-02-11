package model

type DNSDISTServerReady struct {
	Name string `json:"name"`
	Host string `json:"host,omitempty"`
	Port string `json:"port"`
	Key  string `json:"key"`
}

type Rule struct {
	ID      string
	Name    string
	Matches string
	Rule    string
	Action  string
}
