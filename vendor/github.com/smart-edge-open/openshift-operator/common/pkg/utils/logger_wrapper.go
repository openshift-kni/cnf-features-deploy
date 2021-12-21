// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2020-2021 Intel Corporation

package utils

import (
	"github.com/go-logr/logr"
	"github.com/sirupsen/logrus"
)

type logrusWrapper struct {
	log       *logrus.Logger
	lastEntry *logrus.Entry
}

func (l *logrusWrapper) Enabled() bool {
	return true
}

func (l *logrusWrapper) Info(msg string, keysAndValues ...interface{}) {
	if l.lastEntry != nil {
		l.lastEntry.WithFields(l.parseFields(keysAndValues)).Info(msg)
		l.lastEntry = nil
	} else {
		logrus.WithFields(l.parseFields(keysAndValues)).Info(msg)
	}
}

func (l *logrusWrapper) Error(err error, msg string, keysAndValues ...interface{}) {
	if l.lastEntry != nil {
		l.lastEntry.WithError(err).WithFields(l.parseFields(keysAndValues)).Error(msg)
		l.lastEntry = nil
	} else {
		logrus.WithError(err).WithFields(l.parseFields(keysAndValues)).Error(msg)
	}
}

func (l *logrusWrapper) V(level int) logr.Logger {
	return l
}

func (l *logrusWrapper) WithValues(keysAndValues ...interface{}) logr.Logger {
	entry := l.getEntry()
	entry.WithFields(l.parseFields(keysAndValues))
	l.lastEntry = entry
	return l
}

func (l *logrusWrapper) parseFields(keysAndValues []interface{}) logrus.Fields {
	res := logrus.Fields{}
	for i := 0; i+1 < len(keysAndValues); i = i + 2 {
		key, ok := keysAndValues[i].(string)
		if ok {
			res[key] = keysAndValues[i+1]
		}
	}
	return res
}

func (l *logrusWrapper) getEntry() *logrus.Entry {
	if l.lastEntry != nil {
		return l.lastEntry
	}
	return logrus.NewEntry(l.log)
}

func (l *logrusWrapper) WithName(name string) logr.Logger {
	entry := l.getEntry()
	l.lastEntry = entry.WithField("name", name)
	return l
}

func NewLogger() *logrus.Logger {
	log := logrus.New()
	log.SetReportCaller(true)
	log.SetFormatter(&logrus.JSONFormatter{})
	return log
}

func NewLogWrapper() *logrusWrapper {
	return &logrusWrapper{
		log: NewLogger(),
	}
}
