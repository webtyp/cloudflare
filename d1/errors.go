package d1

import . "webtyp.com/fmt"

const errPrefix = "d1: "

var ErrDatabaseNotFound = Err(errPrefix, "database", "not", "found")
