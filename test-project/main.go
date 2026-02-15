package main

import (
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sys/unix"
)

func main() {
	fmt.Println("GolangZakhireh test project starting...")

	id := uuid.New()
	fmt.Println("UUID:", id)

	err := errors.New("test error")
	fmt.Println("Wrapped error:", errors.Wrap(err, "context"))

	logrus.Info("Logging with logrus")

	var stat unix.Stat_t
	if err := unix.Stat("/", &stat); err != nil {
		log.Println("unix.Stat error:", err)
	} else {
		fmt.Println("unix.Stat OK, device:", stat.Dev)
	}

	assert.Equal(nil, 1, 1, "test assert example")
	fmt.Println("Assert passed")

	fmt.Println("GolangZakhireh test project finished")
}
