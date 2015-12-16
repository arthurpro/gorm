// Copyright 2012 Kamil Kisiel. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package sqlstruct provides some convenience functions for using structs with
the Go standard library's database/sql package.

The package works with structs that are tagged with a "sql" tag that identifies
which column of a SQL query the field corresponds to.

For example:

	type T struct {
		F1 string `sql:"f1"`
		F2 string `sql:"f2"`
	}

	rows, err := db.Query(fmt.Sprintf("SELECT %s FROM tablename", sqlstruct.Columns(T)))
	...

	for rows.Next() {
		var t T
		err = sqlstruct.Scan(&t, rows)
		...
	}

	err = rows.Err() // get any errors encountered during iteration


*/
package sqlstruct

import (
	"database/sql"
	"fmt"
	"reflect"
)

// Modified version of sqlstruct (http://go.pkgdoc.org/github.com/kisielk/sqlstruct)
// Added support for anonymous fields/structs

// Rows defines the interface of types that are scannable with the Scan function.
// It is implemented by the sql.Rows type from the standard library
type Rows interface {
	Scan(...interface{}) error
	Columns() ([]string, error)
}

type Session struct {
	finfos map[reflect.Type][]field
}

func NewSession() *Session {
	return &Session{
		finfos: make(map[reflect.Type][]field),
	}
}

func (s *Session) Scan(dest interface{}, rows Rows) error {
	destv := reflect.ValueOf(dest)
	typ := destv.Type()

	if typ.Kind() != reflect.Ptr || typ.Elem().Kind() != reflect.Struct {
		panic(fmt.Errorf("dest must be pointer to struct; got %T", destv))
	}

	valtyp := typ.Elem()
	fields, ok := s.finfos[valtyp]
	if !ok {
		fields = typeFields(valtyp)
		s.finfos[valtyp] = fields
	}

	return scan(destv, fields, rows)
}

func (s *Session) Columns(d interface{}) (names []string) {
	v := reflect.ValueOf(d)
	valtyp := v.Type()
	fields, ok := s.finfos[valtyp]
	if !ok {
		fields = typeFields(valtyp)
		s.finfos[valtyp] = fields
	}
	return columns(v, fields)
}

func (s *Session) MustScan(dest interface{}, rows Rows) {
	if err := s.Scan(dest, rows); err != nil {
		panic(err)
	}
}

// Scan scans the next row from rows in to a struct pointed to by dest. The struct type
// should have exported fields tagged with the "sql" tag. Columns from row which are not
// mapped to any struct fields are ignored. Struct fields which have no matching column
// in the result set are left unchanged.
func scan(destv reflect.Value, fields []field, rows Rows) error {
	finfos := make(map[string]field)
	for _, f := range fields {
		finfos[f.name] = f
	}

	elem := destv.Elem()
	var values []interface{}

	cols, err := rows.Columns()
	if err != nil {
		return err
	}

	for _, name := range cols {
		fi, ok := finfos[name]
		var v interface{}
		if !ok {
			fmt.Println("sqlstruct: no field for", name)
			// There is no field mapped to this column so we discard it
			v = &sql.RawBytes{}
		} else {
			v = elem.FieldByIndex(fi.index).Addr().Interface()
		}
		values = append(values, v)
	}

	if err := rows.Scan(values...); err != nil {
		return err
	}

	return nil
}

func columns(v reflect.Value, fields []field) (names []string) {
	names = make([]string, 0, len(fields))
	for _, f := range fields {
		names = append(names, f.ColName())
	}

	return
}

func Scan(dest interface{}, rows Rows) error {
	destv := reflect.ValueOf(dest)
	typ := destv.Type()

	if typ.Kind() != reflect.Ptr || typ.Elem().Kind() != reflect.Struct {
		panic(fmt.Errorf("dest must be pointer to struct; got %T", destv))
	}

	return scan(destv, typeFields(typ.Elem()), rows)
}

func Columns(s interface{}) (names []string) {
	v := reflect.ValueOf(s)
	fields := typeFields(v.Type())
	return columns(v, fields)
}

func MustScan(dest interface{}, rows Rows) {
	if err := Scan(dest, rows); err != nil {
		panic(err)
	}
}
