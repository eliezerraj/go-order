package erro

import (
	"errors"
)

var (
	ErrNotFound 		= errors.New("item not found")
	ErrInsert 			= errors.New("insert data error")
	ErrUpdate			= errors.New("update data error")
	ErrDelete 			= errors.New("delete data error")
	ErrUnmarshal 		= errors.New("unmarshal json error")
	ErrUnauthorized 	= errors.New("not authorized")
)