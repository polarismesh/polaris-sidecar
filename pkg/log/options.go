/**
 * Tencent is pleased to support the open source community by making Polaris available.
 *
 * Copyright (C) 2019 THL A29 Limited, a Tencent company. All rights reserved.
 *
 * Licensed under the BSD 3-Clause License (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * https://opensource.org/licenses/BSD-3-Clause
 *
 * Unless required by applicable law or agreed to in writing, software distributed
 * under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR
 * CONDITIONS OF ANY KIND, either express or implied. See the License for the
 * specific language governing permissions and limitations under the License.
 */

package log

import (
	"errors"
)

const (
	DefaultLoggerName         = "default"
	defaultOutputLevel        = InfoLevel
	defaultStacktraceLevel    = NoneLevel
	defaultOutputPath         = "stdout"
	defaultErrorOutputPath    = "stderr"
	defaultRotationMaxAge     = 7
	defaultRotationMaxSize    = 100
	defaultRotationMaxBackups = 10
)

// Level is an enumeration of all supported log levels.
type Level int

const (
	// NoneLevel disables logging
	NoneLevel Level = iota
	// FatalLevel enables fatal level logging
	FatalLevel
	// ErrorLevel enables error level logging
	ErrorLevel
	// WarnLevel enables warn level logging
	WarnLevel
	// InfoLevel enables info level logging
	InfoLevel
	// DebugLevel enables debug level logging
	DebugLevel
)

var levelToString = map[Level]string{
	DebugLevel: "debug",
	InfoLevel:  "info",
	WarnLevel:  "warn",
	ErrorLevel: "error",
	FatalLevel: "fatal",
	NoneLevel:  "none",
}

var stringToLevel = map[string]Level{
	"debug": DebugLevel,
	"info":  InfoLevel,
	"warn":  WarnLevel,
	"error": ErrorLevel,
	"fatal": FatalLevel,
	"none":  NoneLevel,
}

// Options defines the set of options supported by logging package.
type Options struct {
	// OutputPaths is a list of file system paths to write the log data to.
	// The special values stdout and stderr can be used to output to the
	// standard I/O streams. This defaults to stdout.
	OutputPaths []string `yaml:"output_paths"`

	// ErrorOutputPaths is a list of file system paths to write logger errors to.
	// The special values stdout and stderr can be used to output to the
	// standard I/O streams. This defaults to stderr.
	ErrorOutputPaths []string `yaml:"error_output_paths"`

	// RotateOutputPath is the path to a rotating log file. This file should
	// be automatically rotated over time, based on the rotation parameters such
	// as RotationMaxSize and RotationMaxAge. The default is to not rotate.
	//
	// This path is used as a foundational path. This is where log output is normally
	// saved. When a rotation needs to take place because the file got too big or too
	// old, then the file is renamed by appending a timestamp to the name. Such renamed
	// files are called backups. Once a backup has been created,
	// output resumes to this path.
	RotateOutputPath string `yaml:"rotate_output_path"`

	// RotateOutputPath is the path to a rotating error log file. This file should
	// be automatically rotated over time, based on the rotation parameters such
	// as RotationMaxSize and RotationMaxAge. The default is to not rotate.
	//
	// This path is used as a foundational path. This is where log output is normally
	// saved. When a rotation needs to take place because the file got too big or too
	// old, then the file is renamed by appending a timestamp to the name. Such renamed
	// files are called backups. Once a backup has been created,
	// output resumes to this path.
	ErrorRotateOutputPath string `yaml:"error_rotate_output_path"`

	// RotationMaxSize is the maximum size in megabytes of a log file before it gets
	// rotated. It defaults to 100 megabytes.
	RotationMaxSize int `yaml:"rotation_max_size"`

	// RotationMaxAge is the maximum number of days to retain old log files based on the
	// timestamp encoded in their filename. Note that a day is defined as 24
	// hours and may not exactly correspond to calendar days due to daylight
	// savings, leap seconds, etc. The default is to remove log files
	// older than 30 days.
	RotationMaxAge int `yaml:"rotation_max_age"`

	// RotationMaxBackups is the maximum number of old log files to retain.  The default
	// is to retain at most 1000 logs.
	RotationMaxBackups int `yaml:"rotation_max_backups"`

	// JSONEncoding controls whether the log is formatted as JSON.
	JSONEncoding bool `yaml:"json_encoding"`

	OutputLevel     string `yaml:"output_level"`
	StacktraceLevel string `yaml:"stacktrace_level"`
	LogCaller       bool   `yaml:"log_caller"`
}

// DefaultOptions returns a new set of options, initialized to the defaults
func DefaultOptions() *Options {
	return &Options{
		OutputPaths:        []string{defaultOutputPath},
		ErrorOutputPaths:   []string{defaultErrorOutputPath},
		RotationMaxSize:    defaultRotationMaxSize,
		RotationMaxAge:     defaultRotationMaxAge,
		RotationMaxBackups: defaultRotationMaxBackups,
		OutputLevel:        levelToString[defaultOutputLevel],
		StacktraceLevel:    levelToString[defaultStacktraceLevel],
	}
}

// SetOutputLevel sets the minimum log output level for a given scope.
func (o *Options) SetOutputLevel(level string) error {
	_, exist := stringToLevel[level]
	if !exist {
		return errors.New("invalid log level")
	}
	o.OutputLevel = level
	return nil
}

// GetOutputLevel returns the minimum log output level for a given scope.
func (o *Options) GetOutputLevel() Level {
	return stringToLevel[o.OutputLevel]
}

// SetStackTraceLevel sets the minimum stack tracing level for a given scope.
func (o *Options) SetStacktraceLevel(level string) error {
	_, exist := stringToLevel[level]
	if !exist {
		return errors.New("invalid stack trace level")
	}
	o.StacktraceLevel = level
	return nil
}

// GetStackTraceLevel returns the minimum stack tracing level for a given scope.
func (o *Options) GetStacktraceLevel() Level {
	return stringToLevel[o.StacktraceLevel]
}
