package r2

import . "webtyp.com/fmt"

const errPrefix = "r2: "

var ErrBucketNotFound = Err(errPrefix + "bucket not found")
