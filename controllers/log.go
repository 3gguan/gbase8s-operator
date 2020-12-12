package controllers

import (
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/sirupsen/logrus"
	"io"
	"os"
	"time"
)

var log = logrus.New()

func init() {
	logFile := "./logs/gbase8s-operator.log"
	log.SetOutput(os.Stdout)
	log.SetFormatter(&logrus.TextFormatter{})

	writer, err := rotatelogs.New(
		logFile+"-%Y-%m-%d-%H:%M",
		rotatelogs.WithLinkName(logFile),
		rotatelogs.WithMaxAge(time.Duration(7)*time.Hour*24),
		rotatelogs.WithRotationTime(time.Duration(1)*time.Second*24),
	)
	writers := []io.Writer{writer, os.Stdout}
	foWriter := io.MultiWriter(writers...)
	if err == nil {
		log.SetOutput(foWriter)
	} else {
		log.Errorf("failed to log to file, err: %s", err.Error())
	}

	log.SetLevel(logrus.InfoLevel)
}
