// -*- mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2018-2019 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

// Package videoanalyticsprovider implements a pipeline discovery provider which queries
// the VA Pipeline service to create virtual devices from VA Pipeline objects.
// This facilitates registration of these as EdgeX devices, with corresponding management/control
// interfaces exposed as EdgeX Commands.
package videoanalyticsprovider

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
)

// Tags is a struct containing tags to be assigned to VA Pipelines in a form of
// map where pipeline name is the key and tags map is the value
type Tags struct {
	Tags map[string]map[string]interface{}
}

// LoadTags loads tags from a json file
func (st *Tags) LoadTags(filePath string) error {
	bytes, err := ioutil.ReadFile(filepath.Clean(filePath))
	if err != nil {
		fmt.Println("Loading file failed. Using empty tags")
		st.Tags = map[string]map[string]interface{}{}
		return err
	}
	err = json.Unmarshal(bytes, &(st.Tags))
	if err != nil {
		fmt.Println("Parsing JSON failed. Using empty tags")
		st.Tags = map[string]map[string]interface{}{}
	}
	return err
}

// SaveTags saves tags as a json file
func (st *Tags) SaveTags(filePath string) error {
	tagsJSON, err := json.Marshal(&st.Tags)
	if err != nil {
		fmt.Println("Saving tags as JSON failed. Using runtime tag cache only")
		return err
	}
	err = ioutil.WriteFile(filePath, tagsJSON, 0600)
	if err != nil {
		fmt.Println("Saving file failed. Using runtime tag cache only")
		return err
	}
	return nil
}
