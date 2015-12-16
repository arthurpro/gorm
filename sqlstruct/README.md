sqlstruct
=========

This is a modified version of http://go.pkgdoc.org/github.com/kisielk/sqlstruct.

It adds support for embedded structs and adds a slightly different type info cache.

## Basic use:
```go

// struct to represent a query from bar
type Rows struct {
	Name string `sql:"name_attr"`
	Id   int64  `sql:"@id"`
	...
}

// Create a new session - this will also maintain a type info cache
// so it's beneficial for complex structs
s := sqlstruct.NewSession()

r := Rows{}
db := sql.Open(...)
dbquery := "select * from bar"

rows := mustQuery(db, dbquery)
for rows.Next() {
	s.MustScan(&r, rows)
	// do something with the Rows data in r
}

```

sqlstruct provides some convenience functions for using structs with go's database/sql package

Documentation can be found via godoc or at http://go.pkgdoc.org/github.com/kisielk/sqlstruct
