/*
 *   Copyright (c) 2026 ForgeZero-cli
 *
 *   This program is free software: you can redistribute it and/or modify
 *   it under the terms of the GNU General Public License as published by
 *   the Free Software Foundation, either version 3 of the License, or
 *   (at your option) any later version.
 *
 *   This program is distributed in the hope that it will be useful,
 *   but WITHOUT ANY WARRANTY; without even the implied warranty of
 *   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *   GNU General Public License for more details.
 *
 *   You should have received a copy of the GNU General Public License
 *   along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package config

type ErrorKind uint8

const (
	ErrorNone ErrorKind = iota
	ErrorFileStat
	ErrorFileRead
	ErrorParseTOML
	ErrorParseYAML
	ErrorCyclicInclude
	ErrorIncludeRead
	ErrorInvalidSourceConfig
	ErrorInvalidMode
	ErrorInvalidProfile
	ErrorInvalidToolchain
	ErrorInvalidIsolation
	ErrorInvalidCacheMode
	ErrorMissingSource
	ErrorBuildRuleActionRequired
	ErrorBuildRuleOutputsRequired
	ErrorDuplicateBuildRuleOutput
	ErrorInvalidOverride
	ErrorUnsupportedOverride
	ErrorInvalidConfig
)

var kindText = [...]string{
	ErrorFileStat:                 "cannot stat config file",
	ErrorFileRead:                 "cannot read config file",
	ErrorParseTOML:                "cannot parse TOML",
	ErrorParseYAML:                "cannot parse YAML",
	ErrorCyclicInclude:            "cyclic config include",
	ErrorIncludeRead:              "cannot read included config file",
	ErrorInvalidSourceConfig:      "conflicting source configuration",
	ErrorInvalidMode:              "invalid mode",
	ErrorInvalidProfile:           "invalid profile",
	ErrorInvalidToolchain:         "invalid toolchain",
	ErrorInvalidIsolation:         "invalid isolation",
	ErrorInvalidCacheMode:         "invalid cache_mode",
	ErrorMissingSource:            "missing source",
	ErrorBuildRuleActionRequired:  "build rule action is required",
	ErrorBuildRuleOutputsRequired: "build rule outputs are required",
	ErrorDuplicateBuildRuleOutput: "duplicate build rule output",
	ErrorInvalidOverride:          "invalid override",
	ErrorUnsupportedOverride:      "unsupported config override",
	ErrorInvalidConfig:            "invalid config",
}

type Error struct {
	Kind   ErrorKind
	Detail string
	Cause  error
}

func (e Error) Error() string {
	if e.Kind == ErrorNone {
		return ""
	}
	text := kindText[e.Kind]
	if text == "" {
		text = "config error"
	}
	if e.Detail == "" {
		if e.Cause == nil {
			return text
		}
		return text + ": " + e.Cause.Error()
	}
	if e.Cause == nil {
		return text + ": " + e.Detail
	}
	return text + ": " + e.Detail + ": " + e.Cause.Error()
}

func (e Error) Is(target error) bool {
	switch t := target.(type) {
	case Error:
		return e.Kind == t.Kind
	case *Error:
		return e.Kind == t.Kind
	default:
		return false
	}
}

func (e Error) Unwrap() error {
	return e.Cause
}

func NewError(kind ErrorKind) error {
	return Error{Kind: kind}
}

func NewErrorDetail(kind ErrorKind, detail string) error {
	return Error{Kind: kind, Detail: detail}
}

func NewErrorCause(kind ErrorKind, detail string, cause error) error {
	return Error{Kind: kind, Detail: detail, Cause: cause}
}
