package domain

import "github.com/sirupsen/logrus"

var Log = logrus.New()

func init() {
	Log.SetLevel(logrus.DebugLevel)
}
