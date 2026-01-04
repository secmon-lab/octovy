package repository

import "github.com/m-mizutani/goerr/v2"

var (
	ErrNotFound      = goerr.New("not found")
	ErrAlreadyExists = goerr.New("already exists")
	ErrInvalidInput  = goerr.New("invalid input")
)
