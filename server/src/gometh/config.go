package gometh

// Config is the server configurtion
type Config struct {
	DataPath       string
	KeystorePath   string
	KeystorePasswd string
	ContractsPath  string
	ParentWSUrl    string // like "ws://127.0.0.1:8546"
	ChildrenWSUrl  string // like "ws://127.0.0.1:8546"
}
