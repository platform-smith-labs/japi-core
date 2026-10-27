package sqltrace

import "strings"

// otherOperation is the db.operation.name used when the first keyword is not
// a known SQL command. It bounds the metric label's cardinality.
const otherOperation = "_OTHER"

// knownOperations are the leading SQL keywords reported as-is.
var knownOperations = map[string]struct{}{
	"ABORT": {}, "ALTER": {}, "ANALYZE": {}, "BEGIN": {}, "CALL": {}, "CLOSE": {},
	"CLUSTER": {}, "COMMENT": {}, "COMMIT": {}, "COPY": {}, "CREATE": {}, "DEALLOCATE": {},
	"DECLARE": {}, "DELETE": {}, "DISCARD": {}, "DO": {}, "DROP": {}, "END": {},
	"EXECUTE": {}, "EXPLAIN": {}, "FETCH": {}, "GRANT": {}, "INSERT": {}, "LISTEN": {},
	"LOCK": {}, "MERGE": {}, "MOVE": {}, "NOTIFY": {}, "PREPARE": {}, "REFRESH": {},
	"REINDEX": {}, "RELEASE": {}, "RESET": {}, "REVOKE": {}, "ROLLBACK": {}, "SAVEPOINT": {},
	"SELECT": {}, "SET": {}, "SHOW": {}, "START": {}, "TABLE": {}, "TRUNCATE": {},
	"UNLISTEN": {}, "UPDATE": {}, "VACUUM": {}, "VALUES": {}, "WITH": {},
}

// maxTableLen caps the table name copied into db.query.summary.
const maxTableLen = 63

// summarize returns the operation name (first SQL keyword, upper-cased, or
// otherOperation) and a query summary: the operation followed by the target
// table when it is cheap to find (SELECT/DELETE ... FROM t, INSERT INTO t,
// UPDATE t, MERGE INTO t, TRUNCATE t). Only keywords and identifiers are ever
// returned; literals, parameters and the rest of the statement are not.
func summarize(query string) (operation, summary string) {
	s := scanner{src: query}
	first := strings.ToUpper(s.word())
	if _, ok := knownOperations[first]; !ok {
		return otherOperation, otherOperation
	}
	var table string
	switch first {
	case "SELECT", "DELETE":
		table = s.identAfter("FROM")
	case "INSERT", "MERGE":
		table = s.identAfter("INTO")
	case "UPDATE":
		if w := s.word(); strings.EqualFold(w, "ONLY") {
			table = identifier(s.word())
		} else {
			table = identifier(w)
		}
	case "TRUNCATE":
		w := s.word()
		if strings.EqualFold(w, "TABLE") {
			w = s.word()
		}
		if strings.EqualFold(w, "ONLY") {
			w = s.word()
		}
		table = identifier(w)
	}
	if table == "" {
		return first, first
	}
	return first, first + " " + table
}

// scanner yields whitespace-separated words, skipping comments, opening
// parentheses and quoted literals.
type scanner struct {
	src string
	pos int
	// paren reports whether the last word followed an opening parenthesis
	// (a subquery or a column list, never a table name).
	paren bool
}

// word returns the next bare word, or "" at end of input. A quoted literal
// ('...') is skipped whole and never returned. A double-quoted identifier is
// returned with its quotes.
func (s *scanner) word() string {
	s.paren = false
	for s.pos < len(s.src) {
		c := s.src[s.pos]
		switch {
		case c == '(':
			s.paren = true
			s.pos++
		case c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == ')' || c == ',' || c == ';':
			s.pos++
		case c == '-' && strings.HasPrefix(s.src[s.pos:], "--"):
			if i := strings.IndexByte(s.src[s.pos:], '\n'); i >= 0 {
				s.pos += i + 1
			} else {
				s.pos = len(s.src)
			}
		case c == '/' && strings.HasPrefix(s.src[s.pos:], "/*"):
			if i := strings.Index(s.src[s.pos+2:], "*/"); i >= 0 {
				s.pos += i + 4
			} else {
				s.pos = len(s.src)
			}
		case c == '\'':
			s.skipQuoted()
		case c == '$' && s.skipDollarQuoted():
		default:
			start := s.pos
			for s.pos < len(s.src) && !strings.ContainsRune(" \t\n\r(),;'", rune(s.src[s.pos])) {
				s.pos++
			}
			return s.src[start:s.pos]
		}
	}
	return ""
}

// skipQuoted skips a single-quoted literal starting at s.pos. A doubled quote
// (”) inside it is part of the literal. In an escape string (E'...', the E directly before the
// quote and not part of a longer word) a backslash escapes the next byte, so
// \' does not end the literal. In a standard string a backslash is an
// ordinary character.
func (s *scanner) skipQuoted() {
	escapes := s.pos > 0 && (s.src[s.pos-1] == 'E' || s.src[s.pos-1] == 'e') &&
		(s.pos == 1 || !isWordByte(s.src[s.pos-2]))
	s.pos++
	for s.pos < len(s.src) {
		switch c := s.src[s.pos]; {
		case escapes && c == '\\':
			s.pos += 2
		case c == '\'' && s.pos+1 < len(s.src) && s.src[s.pos+1] == '\'':
			s.pos += 2 // doubled quote: part of the literal
		case c == '\'':
			s.pos++
			return
		default:
			s.pos++
		}
	}
	s.pos = len(s.src)
}

// isWordByte reports whether c can be part of an identifier or keyword.
func isWordByte(c byte) bool {
	return c == '_' || c == '$' || c >= '0' && c <= '9' || (c|0x20) >= 'a' && (c|0x20) <= 'z'
}

// skipDollarQuoted skips a PostgreSQL dollar-quoted literal ($$...$$ or
// $tag$...$tag$) starting at s.pos and reports whether there was one.
// A positional parameter ($1) is not a dollar quote.
func (s *scanner) skipDollarQuoted() bool {
	rest := s.src[s.pos+1:]
	end := strings.IndexByte(rest, '$')
	if end < 0 {
		return false
	}
	tag := rest[:end]
	for i := 0; i < len(tag); i++ {
		c := tag[i]
		if !(c == '_' || (c|0x20) >= 'a' && (c|0x20) <= 'z' || i > 0 && c >= '0' && c <= '9') {
			return false
		}
	}
	delim := "$" + tag + "$"
	body := s.pos + len(delim)
	if i := strings.Index(s.src[body:], delim); i >= 0 {
		s.pos = body + i + len(delim)
	} else {
		s.pos = len(s.src)
	}
	return true
}

// identAfter scans forward to the first occurrence of keyword at the top
// level of the scan and returns the identifier that follows it.
func (s *scanner) identAfter(keyword string) string {
	for w := s.word(); w != ""; w = s.word() {
		if strings.EqualFold(w, keyword) {
			if t := s.word(); !s.paren {
				return identifier(t)
			}
			return ""
		}
	}
	return ""
}

// identifier returns w if it is a plain or schema-qualified SQL identifier
// (letters, digits, underscore, dot, optionally double-quoted), else "".
// Anything else (a parameter, a literal, a function call) is not a table
// name and is not reported.
func identifier(w string) string {
	w = strings.ReplaceAll(w, `"`, "")
	if w == "" || len(w) > maxTableLen {
		return ""
	}
	if c := w[0]; !(c == '_' || (c|0x20) >= 'a' && (c|0x20) <= 'z') {
		return ""
	}
	for i := 0; i < len(w); i++ {
		c := w[i]
		if !(c == '_' || c == '.' || c >= '0' && c <= '9' || (c|0x20) >= 'a' && (c|0x20) <= 'z') {
			return ""
		}
	}
	return w
}
