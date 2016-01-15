package auth

const (
	Database    string = "hypernetes"
	AuthTable   string = "auth"
	TenantTable string = "tenant"
)

// AuthItem is the credentials value for individual credential fields
type AuthItem struct {
	AccessKey string
	SecretKey string
	UserID    string
	TenantID  string
}

type TenantItem struct {
	TenantID   string
	Namespaces []string
	// Containers
	Containers map[string]string
	// Image Names
	Images map[string]string
	// Volume Names
	Volumes map[string]string
	// Network IDs
	Networks map[string]string
}
