package db

import (
	"errors"

	"github.com/go-sql-driver/mysql"
)

func isDuplicate(err error) bool {
	mysqlErr, ok := errors.AsType[*mysql.MySQLError](err)
	return ok && mysqlErr.Number == 1062
}
