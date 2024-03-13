// Copyright 2023 XigXog
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
//
// SPDX-License-Identifier: MPL-2.0

package efs

import (
	"embed"
)

const (
	HelloWorldPath = "hello-world"
	GraphQLPath    = "graphql"
)

// Go will not embed directories containing a go.mod file. To resolve this the
// extension .trim is added. This should be removed when writing the template
// files to disk. Additionally, files starting with . or _ are ignored. Adding
// all: prefix ensures they are included.

//go:embed all:*
var EFS embed.FS
