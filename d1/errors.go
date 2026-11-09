package d1

import . "webtyp.com/fmt"

const errPrefix = "d1: "

var ErrDatabaseNotFound = Err(errPrefix, "database", "not", "found")

// ErrNoChangesCount means the run result of D1 carried no meta.changes: the count of rows is never guessed.
var ErrNoChangesCount = Err(errPrefix, "run", "result", "has", "no", "meta.changes")
