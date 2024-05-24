// Copyright 2024 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"github.com/go-logr/logr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	ctrlZap "sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func Logger(development bool, level int) logr.Logger {
	ctrlZapOpts := ctrlZap.Options{
		Development: true,
		Level:       zap.NewAtomicLevelAt(zapcore.Level(level)),
	}
	if !development {
		ctrlZapOpts.Development = false
		ctrlZapOpts.EncoderConfigOptions = []ctrlZap.EncoderConfigOption{
			logEncoderOptionsProd(),
		}
	}
	return ctrlZap.New(ctrlZap.UseFlagOptions(&ctrlZapOpts))
}

// logEncoderOptionsProd
func logEncoderOptionsProd() ctrlZap.EncoderConfigOption {
	return func(ec *zapcore.EncoderConfig) {
		ec.LevelKey = "severity"
		ec.MessageKey = "message"
		ec.TimeKey = "time"
		ec.EncodeTime = zapcore.RFC3339TimeEncoder
		ec.EncodeLevel = func(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
			switch l {
			case zapcore.InfoLevel:
				enc.AppendString("info")
			case zapcore.WarnLevel:
				enc.AppendString("warning")
			case zapcore.ErrorLevel:
				enc.AppendString("error")
			case zapcore.DPanicLevel:
				enc.AppendString("critical")
			case zapcore.PanicLevel:
				enc.AppendString("alert")
			case zapcore.FatalLevel:
				enc.AppendString("emergency")
			default:
				enc.AppendString("debug")
			}
		}
	}
}
