// Code generated from Pkl module `spike.contract.Fragments`. DO NOT EDIT.
package fragments

// A PostgreSQL connection. The URL carries no password: it names the environment variable that does.
type Postgres struct {
	// A connection URL without credentials, for example postgres://user@host:5432/dbname?sslmode=require.
	Url string `pkl:"url"`

	// The NAME of the environment variable holding the password. Unset means the connection needs none.
	PasswordEnv *string `pkl:"passwordEnv"`

	// Pool size for this instance. Sized against the server's limit divided by the number of instances, not guessed.
	MaxConnections int `pkl:"maxConnections"`
}
