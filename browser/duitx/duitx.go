package duitx

import (
	"github.com/psilva261/opossum/logger"
)

var log *logger.Logger

func SetLogger(l *logger.Logger) {
	log = l
}