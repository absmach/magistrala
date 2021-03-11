// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"fmt"
	"sort"

	"github.com/mainflux/mainflux/things"
)

// Since mocks will store data in map, and they need to resemble the real
// identifiers as much as possible, a key will be created as combination of
// owner and their own identifiers. This will allow searching either by
// prefix or suffix.
func key(owner string, id string) string {
	return fmt.Sprintf("%s-%s", owner, id)
}

func sortThings(pm things.PageMetadata, ths []things.Thing) []things.Thing {
	switch pm.Order {
	case "name":
		if pm.Dir == "asc" {
			sort.SliceStable(ths, func(i, j int) bool {
				return ths[i].Name < ths[j].Name
			})
		}
		if pm.Dir == "desc" {
			sort.SliceStable(ths, func(i, j int) bool {
				return ths[i].Name > ths[j].Name
			})
		}
	case "id":
		if pm.Dir == "asc" {
			sort.SliceStable(ths, func(i, j int) bool {
				return ths[i].ID < ths[j].ID
			})
		}
		if pm.Dir == "desc" {
			sort.SliceStable(ths, func(i, j int) bool {
				return ths[i].ID > ths[j].ID
			})
		}
	default:
		sort.SliceStable(ths, func(i, j int) bool {
			return ths[i].ID < ths[j].ID
		})
	}

	return ths
}

func sortChannels(pm things.PageMetadata, chs []things.Channel) []things.Channel {
	switch pm.Order {
	case "name":
		if pm.Dir == "asc" {
			sort.SliceStable(chs, func(i, j int) bool {
				return chs[i].Name < chs[j].Name
			})
		}
		if pm.Dir == "desc" {
			sort.SliceStable(chs, func(i, j int) bool {
				return chs[i].Name > chs[j].Name
			})
		}
	case "id":
		if pm.Dir == "asc" {
			sort.SliceStable(chs, func(i, j int) bool {
				return chs[i].ID < chs[j].ID
			})
		}
		if pm.Dir == "desc" {
			sort.SliceStable(chs, func(i, j int) bool {
				return chs[i].ID > chs[j].ID
			})
		}
	default:
		sort.SliceStable(chs, func(i, j int) bool {
			return chs[i].ID < chs[j].ID
		})
	}

	return chs
}
