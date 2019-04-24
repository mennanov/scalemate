package models

// TableNames is a list of all the registered table names that is used in tests to clean up database.
var TableNames = []string{"container_labels", "containers", "node_labels", "nodes", "processed_events"}

// mapKeys returns a slice of map string keys.
func mapKeys(m map[string]interface{}) []string {
	keys := make([]string, len(m))

	i := 0
	for k := range m {
		keys[i] = k
		i++
	}
	return keys
}
