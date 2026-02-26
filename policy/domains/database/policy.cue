// Database domain policy — constraints for SQL / database agents.

package database

#DatabaseAction: {
	action_type: "select" | "insert" | "update" | "delete" | "create_table" | "drop_table"
	target:      string & !=""  // table name
	parameters:  #DatabaseParams
	context: {
		domain: "database"
		...
	}
	...
}

#DatabaseParams: {
	where_clause?: string
	columns?:      [...string]
	values?:       _
	limit?:        int & >0
	...
}

// ── Policy rules ──

// Tables that cannot be modified (read-only)
#ReadOnlyTables: [...string]

// Tables where DELETE is never allowed
#NoDeleteTables: [...string]

// Sensitive columns that require special access
#SensitiveColumns: [...string]

policy: {
	read_only_tables:  #ReadOnlyTables | *[]
	no_delete_tables:  #NoDeleteTables | *[]
	sensitive_columns: #SensitiveColumns | *[]

	// WHERE clause is required for UPDATE and DELETE
	require_where_for_mutations: bool | *true

	// DROP TABLE is globally forbidden by default
	allow_drop_table: bool | *false

	// Maximum JOIN depth
	max_join_depth: int | *3
}
