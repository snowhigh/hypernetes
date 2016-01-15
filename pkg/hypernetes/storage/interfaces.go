package storage

type Interface interface {
	// Create adds a new entry for table of database
	Create(database, table string, data interface{}) error
	// Get gets an existed item from table of database
	Get(database, table, key, value string, data interface{}) error
	// Delete removes the specified accesskey
	Delete(database, table, key, value string) error
	// Set
	Set(database, table, key, value string, data interface{}) error
}
