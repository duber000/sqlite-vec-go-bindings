//go:build darwin

package vec

// #cgo CFLAGS: -DSQLITE_CORE -Wno-deprecated-declarations
// #include "sqlite-vec.h"
// #include <stdlib.h>
//
import "C"
import (
	"bytes"
	"encoding/binary"
	"fmt"
	"reflect"
	"unsafe"
)

// Error represents a sqlite-vec extension error.
type Error struct {
	Code         int
	ExtendedCode int
	msg          string
}

func (e *Error) Error() string {
	return fmt.Sprintf("sqlite-vec error %d: %s", e.Code, e.msg)
}

// LoadConnection loads the sqlite-vec extension into the provided SQLite connection.
// This is the recommended way to load sqlite-vec on macOS and other platforms.
//
// The db parameter should be an unsafe.Pointer to a sqlite3* connection handle.
// Returns an error if the extension fails to load.
//
// Example usage with mattn/go-sqlite3:
//
//	import (
//	    "database/sql"
//	    sqlite_vec "github.com/asg017/sqlite-vec-go-bindings/cgo"
//	    "github.com/mattn/go-sqlite3"
//	)
//
//	sql.Register("sqlite3_with_vec", &sqlite3.SQLiteDriver{
//	    ConnectHook: func(conn *sqlite3.SQLiteConn) error {
//	        return sqlite_vec.LoadConnectionGo(conn)
//	    },
//	})
func LoadConnection(db unsafe.Pointer) error {
	var errMsg *C.char
	rc := C.sqlite3_vec_init((*C.sqlite3)(db), &errMsg, nil)
	if rc != C.SQLITE_OK {
		defer C.free(unsafe.Pointer(errMsg))
		if errMsg != nil {
			return &Error{
				Code:         int(rc),
				ExtendedCode: int(rc),
				msg:          C.GoString(errMsg),
			}
		}
		return &Error{
			Code:         int(rc),
			ExtendedCode: int(rc),
			msg:          "failed to load sqlite-vec extension",
		}
	}
	return nil
}

// LoadConnectionGo loads the sqlite-vec extension into a mattn/go-sqlite3 connection.
// This function extracts the underlying sqlite3* pointer using reflection.
//
// Example usage:
//
//	sql.Register("sqlite3_with_vec", &sqlite3.SQLiteDriver{
//	    ConnectHook: func(conn *sqlite3.SQLiteConn) error {
//	        return sqlite_vec.LoadConnectionGo(conn)
//	    },
//	})
func LoadConnectionGo(conn interface{}) error {
	// Use reflection to access the unexported 'db' field in SQLiteConn
	v := reflect.ValueOf(conn)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	dbField := v.FieldByName("db")
	if !dbField.IsValid() {
		return &Error{
			Code:         1,
			ExtendedCode: 1,
			msg:          "invalid connection: could not find db field",
		}
	}

	// Extract the pointer using unsafe to access unexported field
	dbPtr := unsafe.Pointer(dbField.UnsafeAddr())
	sqliteDB := *(**C.sqlite3)(dbPtr)

	if sqliteDB == nil {
		return &Error{
			Code:         1,
			ExtendedCode: 1,
			msg:          "invalid connection: db pointer is nil",
		}
	}

	return LoadConnection(unsafe.Pointer(sqliteDB))
}

// Deprecated: Use [LoadConnection] or [LoadConnectionGo] instead.
//
// Auto enables process-global auto-extension loading. This function uses
// sqlite3_auto_extension() which is deprecated on macOS and may not work
// on future Apple platforms.
//
// Prefer loading the extension per-connection using LoadConnection or
// LoadConnectionGo with a ConnectHook instead.
func Auto() {
	C.sqlite3_auto_extension((*[0]byte)((C.sqlite3_vec_init)))
}

// Deprecated: Use per-connection loading instead.
//
// Cancel disables process-global auto-extension loading. This function uses
// sqlite3_cancel_auto_extension() which is deprecated on macOS.
func Cancel() {
	C.sqlite3_cancel_auto_extension((*[0]byte)(C.sqlite3_vec_init))
}

// Serializes a float32 list into a vector BLOB that sqlite-vec accepts.
func SerializeFloat32(vector []float32) ([]byte, error) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, vector)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
