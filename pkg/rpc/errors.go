/*
   Copyright 2022 CESS scheduler authors

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

      http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package rpc

import "fmt"

// Error wraps RPC errors, which contain an error code in addition to the message.
type Error interface {
	Error() string    // returns the message
	ErrorCode() int32 // returns the code
}

const (
	defaultErrorCode = -1 - iota
	ParseErrorCode
	MethodNotFoundErrorCode
)

type parseError struct{ message string }

func (e *parseError) ErrorCode() int32 { return ParseErrorCode }

func (e *parseError) Error() string { return e.message }

type methodNotFoundError struct{ method string }

func (e *methodNotFoundError) ErrorCode() int { return MethodNotFoundErrorCode }

func (e *methodNotFoundError) Error() string {
	return fmt.Sprintf("the method %s does not exist/is not available", e.method)
}
