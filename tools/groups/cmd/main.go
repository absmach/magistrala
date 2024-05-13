// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// Package main contains a tool for testing groups performance.
package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/0x6flab/namegenerator"
	sdk "github.com/absmach/magistrala/pkg/sdk/go"
)

var (
	namesgenerator = namegenerator.NewGenerator()
	defPass        = "12345678"
)

func main() {
	num := flag.Uint64("num", 10, "number of groups to create")
	flag.Parse()

	sdkConf := sdk.Config{
		UsersURL:   "http://localhost:9002",
		DomainsURL: "http://localhost:8189",
	}

	s := sdk.NewSDK(sdkConf)

	token, err := createUser(s)
	if err != nil {
		panic(err)
	}

	parentID := ""
	for i := range int(*num) {
		id, err := createGroup(s, token, parentID)
		if err != nil {
			panic(err)
		}
		parentID = id
		if i == 0 {
			fmt.Printf("\nCURL Command to fetch children:\ncurl -X \"GET\"  -H \"accept: application/json\" -H \"Authorization: Bearer %s\" \"http://localhost:9002/groups/%s/children?tree=true\"\n\n", token, id)
		}
	}
	fmt.Printf("\nCURL Command to fetch parents:\ncurl -X \"GET\"  -H \"accept: application/json\" -H \"Authorization: Bearer %s\" \"http://localhost:9002/groups/%s/parents?tree=true\"\n\n", token, parentID)
}

func createGroup(s sdk.SDK, token, parentID string) (string, error) {
	group := sdk.Group{
		Name:     namesgenerator.Generate(),
		Status:   sdk.EnabledStatus,
		ParentID: parentID,
	}

	group, err := s.CreateGroup(group, token)
	if err != nil {
		return "", fmt.Errorf("failed to create the group: %w", err)
	}

	return group.ID, nil
}

func createUser(s sdk.SDK) (string, error) {
	name := namesgenerator.Generate()
	user := sdk.User{
		Name: name,
		Credentials: sdk.Credentials{
			Identity: fmt.Sprintf("%s@email.com", name),
			Secret:   defPass,
		},
		Status: sdk.EnabledStatus,
	}

	if _, err := s.CreateUser(user, ""); err != nil {
		return "", fmt.Errorf("unable to create user: %w", err)
	}

	login := sdk.Login{
		Identity: user.Credentials.Identity,
		Secret:   user.Credentials.Secret,
	}
	token, err := s.CreateToken(login)
	if err != nil {
		return "", fmt.Errorf("unable to login user: %w", err)
	}

	dname := namesgenerator.Generate()
	domain := sdk.Domain{
		Name:  dname,
		Alias: strings.ToLower(dname),
	}
	domain, err = s.CreateDomain(domain, token.AccessToken)
	if err != nil {
		return "", fmt.Errorf("unable to create domain: %w", err)
	}

	login = sdk.Login{
		Identity: user.Credentials.Identity,
		Secret:   user.Credentials.Secret,
		DomainID: domain.ID,
	}
	token, err = s.CreateToken(login)
	if err != nil {
		return "", fmt.Errorf("unable to login user: %w", err)
	}

	return token.AccessToken, nil
}

// Parent: 9b1991c4-09da-4e38-935e-2028931de9fc
// Child: c2ee829d-a2e7-4e11-84ee-7b327bf6ad2d
