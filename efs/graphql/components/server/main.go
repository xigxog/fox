// Copyright 2023 XigXog
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
//
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"embed"
	"html/template"

	"github.com/xigxog/kubefox/kit"
	"github.com/xigxog/kubefox/kit/graphql"
)

//go:embed static/*
//go:embed templates/*
var EFS embed.FS

var (
	tpl            *template.Template
	graphqlAdapter kit.ComponentDep
	hasuraAdapter  kit.ComponentDep
)

func main() {
	k := kit.New()

	var err error
	tpl, err = template.ParseFS(EFS, "templates/*.html")
	if err != nil {
		k.Log().Fatal(err)
	}

	graphqlAdapter = k.HTTPAdapter("graphql")
	hasuraAdapter = k.HTTPAdapter("hasura")

	k.Static("/{{.Vars.subPath}}/hasura/static", "static", EFS)
	k.Route("Path(`/{{.Vars.subPath}}/hasura/heroes`)", listHeroes)
	k.Route("PathPrefix(`/{{.Vars.subPath}}/hasura`)", forwardHasura)

	k.Start()
}

func listHeroes(k kit.Kontext) error {
	client := graphql.New(k, graphqlAdapter)

	// For additional documentation check out
	// https://github.com/hasura/go-graphql-client.
	var query struct {
		Superhero []struct {
			Name      string `graphql:"superhero_name"`
			RealName  string `graphql:"full_name"`
			Alignment struct {
				Value string `graphql:"alignment"`
			}
		} `graphql:"superhero(order_by: {superhero_name: asc})"`
	}
	if err := client.Query(&query, nil); err != nil {
		return err
	}

	return k.Resp().SendHTMLTemplate(tpl, "index.html", query)
}

func forwardHasura(k kit.Kontext) error {
	req := k.Forward(hasuraAdapter)
	req.RewritePath(k.PathSuffix())

	resp, err := req.Send()
	if err != nil {
		return err
	}

	return k.Resp().Forward(resp)
}
