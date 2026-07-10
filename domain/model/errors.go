package model

import "errors"

var ErrNotFound = errors.New("model: not found")
var ErrIntegrityFailed = errors.New("model: integrity check failed")
