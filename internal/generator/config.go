package generator

type Config struct {
	InputPath  string
	OutputDir  string
	Lang       string // ts|js
	SDKName    string
	BaseURLVar string // exported base url var name
	AuthMode   string // none|bearer
	TokenFn    string // getToken
}
