package cfg

type RouterDescr struct {
	Name      string
	Community string
	Vendor    string
}
type GenericCfg struct {
	Tasks      []string
	Pollers    int
	Timeout    int
	Retries    int
	GraphiteIP string
	RedisAddr  string
}
