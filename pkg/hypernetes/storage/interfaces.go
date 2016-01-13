package storage

import "k8s.io/kubernetes/pkg/hypernetes/auth"

type Interface interface {
	// Create adds a new entry for table of database
	Create(database, table string, auth auth.AuthItem) error
	// Get gets an existed item from table of database
	Get(database, table, accesskey string) (*auth.AuthItem, error)
	// Delete removes the specified accesskey
	Delete(database, table, accesskey string) error
	// Set
	Set(database, table, accesskey string, auth auth.AuthItem) error
}
