// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package fieldmaskpb

import (
	"strconv"
	"strings"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// validator returns the path check used by [NewE]: against the descriptor of
// message when it is not nil, and by syntax otherwise.
func validator(message proto.Message) func([]string) error {
	if message == nil {
		return checkSyntax
	}

	desc := message.ProtoReflect().Descriptor()

	return func(paths []string) error {
		return checkFields(desc, paths)
	}
}

// checkSyntax reports the first path that is not lower snake_case segments
// separated by dots. The wildcard is accepted as the only path.
func checkSyntax(paths []string) error {
	if len(paths) == 1 && paths[0] == wildcard {
		return nil
	}

	for _, path := range paths {
		if reason := syntaxError(path); reason != "" {
			return PathError{Path: path, Reason: reason}
		}
	}

	return nil
}

// syntaxError describes why path is not well-formed, or returns "".
func syntaxError(path string) string {
	switch path {
	case wildcard:
		return "the wildcard must be the only path"
	case "":
		return "empty path"
	}

	for segment := range strings.SplitSeq(path, ".") {
		if segment == "" {
			return "empty field name"
		}

		if !isLower(segment[0]) {
			return "field name " + strconv.Quote(segment) + " must start with a lower-case letter"
		}

		for i := 1; i < len(segment); i++ {
			if c := segment[i]; !isLower(c) && !isDigit(c) && c != '_' {
				return "field name " + strconv.Quote(segment) + " is not lower snake_case"
			}
		}
	}

	return ""
}

// checkFields reports the first path that does not select a field of desc.
// The wildcard is accepted as the only path.
func checkFields(desc protoreflect.MessageDescriptor, paths []string) error {
	if len(paths) == 1 && paths[0] == wildcard {
		return nil
	}

	for _, path := range paths {
		if _, err := resolve(desc, path); err != nil {
			return err
		}
	}

	return nil
}
