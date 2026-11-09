//go:build wasm

package d1docker

import (
	"testing"

	"webtyp.com/cloudflare/d1"
	"webtyp.com/ddl"
	ddlconf "webtyp.com/ddl/conformance"
	"webtyp.com/model"
	"webtyp.com/storage"
	dbconf "webtyp.com/storage/conformance"
)

const (
	bindingName = "DB"
	userTables  = "SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%' AND name NOT LIKE '_cf_%'"
)

// open returns the D1 binding as a connection with every user table dropped, so each clause starts from an empty database.
func open(t *testing.T) storage.Conn {
	t.Helper()
	db, err := d1.NewEdge(bindingName)
	if err != nil {
		t.Fatalf("NewEdge: %v (run these tests through run.sh: they need a Miniflare D1 binding)", err)
	}
	conn := db.RawConn()
	rows, err := conn.Query(userTables)
	if err != nil {
		t.Fatalf("list tables: %v", err)
	}
	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan table name: %v", err)
		}
		names = append(names, name)
	}
	rows.Close()
	for _, name := range names {
		if _, err := conn.Exec("DROP TABLE " + name); err != nil {
			t.Fatalf("drop %s: %v", name, err)
		}
	}
	return conn
}

func TestD1_StorageConformance(t *testing.T) {
	dbconf.Run(t, dbconf.Factory{
		Name: "d1",
		New: func(t *testing.T, models ...model.Model) storage.Conn {
			conn := open(t)
			dc, ok := d1.DDLCompiler(conn)
			if !ok {
				t.Fatalf("the D1 connection is not a ddl.Compiler")
			}
			schema := ddl.New(conn, dc)
			for _, m := range models {
				if err := schema.Sync(m); err != nil {
					t.Fatalf("Sync: %v", err)
				}
			}
			return conn
		},
	})
}

func TestD1_DDLConformance(t *testing.T) {
	ddlconf.Run(t, ddlconf.Factory{
		Name: "d1",
		New: func(t *testing.T) (*ddl.DB, storage.Conn, func(string) []string) {
			conn := open(t)
			dc, ok := d1.DDLCompiler(conn)
			if !ok {
				t.Fatalf("the D1 connection is not a ddl.Compiler")
			}
			cols := func(table string) []string {
				rows, err := conn.Query("PRAGMA table_info(" + table + ")")
				if err != nil {
					t.Fatalf("PRAGMA table_info: %v", err)
				}
				defer rows.Close()
				var names []string
				for rows.Next() {
					var cid, notnull, pk int
					var name, typeStr string
					var dflt any
					if err := rows.Scan(&cid, &name, &typeStr, &notnull, &dflt, &pk); err != nil {
						t.Fatalf("scan table_info: %v", err)
					}
					names = append(names, name)
				}
				return names
			}
			return ddl.New(conn, dc), conn, cols
		},
	})
}
