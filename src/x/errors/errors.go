// Copyright (c) 2016 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

// Package errors provides utilities for working with different types errors.
package errors

import (
	"bytes"
	"errors"
	"fmt"
)

// FirstError returns the first non nil error.
func FirstError(errs ...error) error {
	for i := range errs {
		if errs[i] != nil {
			return errs[i]
		}
	}
	return nil
}

type containedError struct {
	inner error
}

func (e containedError) Error() string {
	return e.inner.Error()
}

func (e containedError) InnerError() error {
	return e.inner
}

// ContainedError is an error with a contained error.
type ContainedError interface {
	InnerError() error
}

// InnerError returns the packaged inner error if this is an error that
// contains another.
func InnerError(err error) error {
	contained, ok := err.(ContainedError)
	if !ok {
		return nil
	}
	return contained.InnerError()
}

type renamedError struct {
	containedError
	renamed error
}

// NewRenamedError returns a new error that packages an inner error with
// a renamed error.
func NewRenamedError(inner, renamed error) error {
	return renamedError{containedError{inner}, renamed}
}

func (e renamedError) Error() string {
	return e.renamed.Error()
}

func (e renamedError) InnerError() error {
	return e.inner
}

type invalidParamsError struct {
	containedError
}

// Wrap wraps an error with a message but preserves the type of the error.
func Wrap(err error, msg string) error {
	renamed := errors.New(msg + ": " + err.Error())
	return NewRenamedError(err, renamed)
}

// Wrapf formats according to a format specifier and uses that string to
// wrap an error while still preserving the type of the error.
func Wrapf(err error, format string, args ...interface{}) error {
	msg := fmt.Sprintf(format, args...)
	return Wrap(err, msg)
}

// NewInvalidParamsError creates a new invalid params error
func NewInvalidParamsError(inner error) error {
	return invalidParamsError{containedError{inner}}
}

func (e invalidParamsError) Error() string {
	return e.inner.Error()
}

func (e invalidParamsError) InnerError() error {
	return e.inner
}

// IsInvalidParams returns true if this is an invalid params error.
func IsInvalidParams(err error) bool {
	return GetInnerInvalidParamsError(err) != nil
}

// GetInnerInvalidParamsError returns an inner invalid params error
// if contained by this error, nil otherwise.
func GetInnerInvalidParamsError(err error) error {
	for err != nil {
		if _, ok := err.(invalidParamsError); ok {
			return InnerError(err)
		}
		// nolint:errorlint
		if multiErr, ok := err.(MultiError); ok {
			for _, e := range multiErr.Errors() {
				if inner := GetInnerInvalidParamsError(e); err != nil {
					return inner
				}
			}
		}
		err = InnerError(err)
	}
	return nil
}

type resourceExhaustedError struct {
	containedError
}

// NewResourceExhaustedError creates a new resource exhausted error
func NewResourceExhaustedError(inner error) error {
	return resourceExhaustedError{containedError{inner}}
}

func (e resourceExhaustedError) Error() string {
	return e.inner.Error()
}

func (e resourceExhaustedError) InnerError() error {
	return e.inner
}

// IsResourceExhausted returns true if this is a resource exhausted error.
func IsResourceExhausted(err error) bool {
	return GetInnerResourceExhaustedError(err) != nil
}

// GetInnerResourceExhaustedError returns an inner resource exhausted error
// if contained by this error, nil otherwise.
func GetInnerResourceExhaustedError(err error) error {
	for err != nil {
		// nolint:errorlint
		if _, ok := err.(resourceExhaustedError); ok {
			return InnerError(err)
		}
		// nolint:errorlint
		if multiErr, ok := err.(MultiError); ok {
			for _, e := range multiErr.Errors() {
				if inner := GetInnerResourceExhaustedError(e); err != nil {
					return inner
				}
			}
		}
		err = InnerError(err)
	}
	return nil
}

// Is checks if the error is or contains the corresponding target error.
// It's intended to mimic the errors.Is functionality, but also consider xerrors' MultiError / InnerError
// wrapping functionality.
func Is(err, target error) bool {
	for err != nil {
		if errors.Is(err, target) {
			return true
		}

		// nolint:errorlint
		if multiErr, ok := err.(MultiError); ok {
			for _, e := range multiErr.Errors() {
				if Is(e, target) {
					return true
				}
			}
		}

		err = InnerError(err)
	}
	return false
}

type retryableError struct {
	containedError
}

// NewRetryableError creates a new retryable error.
func NewRetryableError(inner error) error {
	return retryableError{containedError{inner}}
}

func (e retryableError) Error() string {
	return e.inner.Error()
}

func (e retryableError) InnerError() error {
	return e.inner
}

// IsRetryableError returns true if this is a retryable error.
func IsRetryableError(err error) bool {
	return GetInnerRetryableError(err) != nil
}

// GetInnerRetryableError returns an inner retryable error
// if contained by this error, nil otherwise.
func GetInnerRetryableError(err error) error {
	for err != nil {
		if _, ok := err.(retryableError); ok {
			return InnerError(err)
		}
		// nolint:errorlint
		if multiErr, ok := err.(MultiError); ok {
			for _, e := range multiErr.Errors() {
				if inner := GetInnerRetryableError(e); err != nil {
					return inner
				}
			}
		}
		err = InnerError(err)
	}
	return nil
}

type nonRetryableError struct {
	containedError
}

// NewNonRetryableError creates a new non-retryable error.
func NewNonRetryableError(inner error) error {
	return nonRetryableError{containedError{inner}}
}

func (e nonRetryableError) Error() string {
	return e.inner.Error()
}

func (e nonRetryableError) InnerError() error {
	return e.inner
}

// IsNonRetryableError returns true if this is a non-retryable error.
func IsNonRetryableError(err error) bool {
	return GetInnerNonRetryableError(err) != nil
}

// GetInnerNonRetryableError returns an inner non-retryable error
// if contained by this error, nil otherwise.
func GetInnerNonRetryableError(err error) error {
	for err != nil {
		if _, ok := err.(nonRetryableError); ok {
			return InnerError(err)
		}
		// nolint:errorlint
		if multiErr, ok := err.(MultiError); ok {
			for _, e := range multiErr.Errors() {
				if inner := GetInnerNonRetryableError(e); err != nil {
					return inner
				}
			}
		}
		err = InnerError(err)
	}
	return nil
}

// IsMultiError returns true if this is a multi-error error.
func IsMultiError(err error) bool {
	_, ok := GetInnerMultiError(err)
	return ok
}

// GetInnerMultiError returns an inner multi-error error
// if contained by this error, nil otherwise.
func GetInnerMultiError(err error) (MultiError, bool) {
	for err != nil {
		if v, ok := err.(MultiError); ok {
			return v, true
		}
		err = InnerError(err)
	}
	return MultiError{}, false
}

// MultiError is an immutable error that packages a list of errors.
//
// TODO(xichen): we may want to limit the number of errors included.
type MultiError struct {
	err    error // optimization for single error case
	errors []error
}

// NewMultiError creates a new MultiError object.
func NewMultiError() MultiError {
	return MultiError{}
}

// Empty returns true if the MultiError has no errors.
func (e MultiError) Empty() bool {
	return e.err == nil
}

func (e MultiError) Error() string {
	if e.err == nil {
		return ""
	}
	if len(e.errors) == 0 {
		return e.err.Error()
	}
	var b bytes.Buffer
	for i := range e.errors {
		b.WriteString(e.errors[i].Error())
		b.WriteString("\n")
	}
	b.WriteString(e.err.Error())
	return b.String()
}

// Errors returns all the errors to inspect individually.
func (e MultiError) Errors() []error {
	if e.err == nil {
		return nil // No errors
	}
	// Need to prepend the first error to result
	// since we avoid allocating array if we don't need it
	// when we accumulate the first error
	result := make([]error, 1+len(e.errors))
	result[0] = e.err
	copy(result[1:], e.errors)
	return result
}

// Contains returns true if any of the errors match the provided error using the Is check.
func (e MultiError) Contains(err error) bool {
	if errors.Is(e.err, err) {
		return true
	}
	for _, e := range e.errors {
		if errors.Is(e, err) {
			return true
		}
	}
	return false
}

// Add adds an error returns a new MultiError object.
func (e MultiError) Add(err error) MultiError {
	if err == nil {
		return e
	}
	me := e
	if me.err == nil {
		me.err = err
		return me
	}
	me.errors = append(me.errors, me.err)
	me.err = err
	return me
}

// FinalError returns all concatenated error messages if any.
func (e MultiError) FinalError() error {
	if e.err == nil {
		return nil
	}
	return e
}

// LastError returns the last received error if any.
func (e MultiError) LastError() error {
	if e.err == nil {
		return nil
	}
	return e.err
}

// NumErrors returns the total number of errors.
func (e MultiError) NumErrors() int {
	if e.err == nil {
		return 0
	}
	return len(e.errors) + 1
}

// Errors is a slice of errors that itself is an error too.
type Errors []error

// Error implements error.
func (e Errors) Error() string {
	buf := bytes.NewBuffer(nil)
	buf.WriteString("[")
	for i, err := range e {
		if err == nil {
			buf.WriteString("<nil>")
		} else {
			buf.WriteString("<")
			buf.WriteString(err.Error())
			buf.WriteString(">")
		}
		if i < len(e)-1 {
			buf.WriteString(", ")
		}
	}
	buf.WriteString("]")
	return buf.String()
}

// GetErrorsFromMultiError returns all errors in the multierror
// as an array of type "error". In case err is not of type
// MultiError then it simply returns an array with err in it.
// In case err is nil then the function will return nil as well.
func GetErrorsFromMultiError(err error) []error {
	if err == nil {
		return nil
	}

	merr, ok := GetInnerMultiError(err)
	if ok {
		// is a MultiError
		return merr.Errors()
	}

	return []error{err}
}
