package models

type Database struct {
	Db_name string `yaml:"db_name"`
    User_name string `yaml:"user_name"`
    Password string `yaml:"password"`
    Host string `yaml:"host"`
}

type Account struct {
	Id             int
	Account_name   string
	Account_key    string
	Proxy_ip       string
	Proxy_login    string
	Proxy_password string
}

type Task struct {
	Date                  int64
	Last_processed_key_id int
	Primary_check         int
	Completed             int
}

type Keywords struct {
	Keyword_id    int
	Keyword_name  string
	Host_id       int
	Host_name     string
	In_statistics int
}

type Group struct {
	Host_name string `xml:"doc>domain"`
	Url       string `xml:"doc>url"`
}

type Grouping struct {
	Page  int     `xml:"page"`
	Group []Group `xml:"group"`
}

type Results struct {
	Grouping Grouping `xml:"grouping"`
}
type Error struct {
	Error_code int `xml:"code,attr"`
}

type Response struct {
	Error   Error   `xml:"error"`
	Results Results `xml:"results"`
}

type Result struct {
	Response Response `xml:"response"`
}