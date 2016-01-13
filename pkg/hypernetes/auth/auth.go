package auth

const (
	Database string = "hypernetes"
	Table    string = "auth"
)

// AuthItem is the credentials value for individual credential fields
type AuthItem struct {
	AccessKey string
	SecretKey string
	TenantID  string
}
