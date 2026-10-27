package sqltrace

import "testing"

func TestSummarize(t *testing.T) {
	tests := []struct{ query, op, summary string }{
		{"SELECT * FROM users WHERE id = $1", "SELECT", "SELECT users"},
		{"  -- note\n/* c */ select id from public.users", "SELECT", "SELECT public.users"},
		{`SELECT * FROM "Orders"`, "SELECT", "SELECT Orders"},
		{"SELECT 'x FROM secret' FROM t", "SELECT", "SELECT t"},
		{"SELECT $$ FROM secret $$ FROM t", "SELECT", "SELECT t"},
		{`SELECT E'a\' FROM secret'`, "SELECT", "SELECT"},
		{`SELECT e'a\\' FROM t`, "SELECT", "SELECT t"},
		{`SELECT E'it''s \' FROM secret' FROM t`, "SELECT", "SELECT t"},
		{`SELECT 'a\' FROM t`, "SELECT", "SELECT t"},
		{"SELECT $tag$ FROM secret $tag$", "SELECT", "SELECT"},
		{"SELECT 1", "SELECT", "SELECT"},
		{"SELECT * FROM (SELECT 1) s", "SELECT", "SELECT"},
		{"SELECT * FROM $1", "SELECT", "SELECT"},
		{"insert into items(a) values ($1)", "INSERT", "INSERT items"},
		{"UPDATE ONLY items SET a = 1", "UPDATE", "UPDATE items"},
		{"DELETE FROM items", "DELETE", "DELETE items"},
		{"TRUNCATE TABLE items", "TRUNCATE", "TRUNCATE items"},
		{"WITH x AS (SELECT 1) SELECT * FROM x", "WITH", "WITH"},
		{"(SELECT 1)", "SELECT", "SELECT"},
		{"secret-value", "_OTHER", "_OTHER"},
		{"", "_OTHER", "_OTHER"},
	}
	for _, tt := range tests {
		op, summary := summarize(tt.query)
		if op != tt.op || summary != tt.summary {
			t.Errorf("summarize(%q) = %q, %q; want %q, %q", tt.query, op, summary, tt.op, tt.summary)
		}
	}
}
