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
	"github.com/pkg/errors"
	"io/ioutil"
	"path/filepath"
)

// PipelineCache is a struct containing info to associate with pipelines in a form of
// map where serial number is the key and info map is the value
type PipelineCache struct {
	//	PipelineCache map[string]map[VAPipelineInfo]interface{}
	VAPipelineInfo []PipelineInfo `json:"va-pipelines"`
}

// LoadInfo loads pipeline info from a json file
func (st *PipelineCache) LoadInfo(lc AppLoggingClient, filePathVa string) error {
	if filePathVa != "" {
		bytes, err := ioutil.ReadFile(filepath.Clean(filePathVa))
		if err != nil {
			return errors.Wrap(err, "Loading file failed for Va pipelines. Using empty pipelineinfo cache.")
		}
		err = json.Unmarshal(bytes, &(st.VAPipelineInfo))
		if err != nil {
			return errors.Wrap(err, "Parsing JSON failed for Va pipelines. Using empty pipelineinfo cache.")
		}
		lc.Info(fmt.Sprintf("*** Loaded %d Va devices from pipelineInfoCache... ***", len(st.VAPipelineInfo)))
	}
	return nil
}

// SaveInfo flushes cached pipeline info to json file
// if target pathfile parameter is supplied.
func (st *PipelineCache) SaveInfo(filePathVa string) error {
	if filePathVa != "" {
		infoJSON, err := json.Marshal(&st.VAPipelineInfo)
		if err != nil {
			return errors.Wrap(err, "Saving Va pipeline Info as JSON failed. Using in-memory cache only!")
		}
		err = ioutil.WriteFile(filePathVa, infoJSON, 0600)
		if err != nil {
			return errors.Wrap(err, "Saving Va pipeline Info file failed. Using in-memory cache only!")
		}
	}
	return nil
}

// TransformpipelineInfoToString marshals structured JSON data to string
func (st *PipelineCache) TransformPipelineInfoToString(deviceVendor string, serialNum string) string {
	var ci []PipelineInfo
	if deviceVendor == "va" {
		ci = st.VAPipelineInfo
	}
	for i := range ci {
		if ci[i].SerialNumber == serialNum {
			b, err := json.Marshal(ci[i])
			if err != nil {
				fmt.Println("Error marshaling pipeline info from index: ", i, " from PipelineInfoCache for SerialNumber: ", serialNum)
			} else {
				fmt.Println("Fetched PipelineInfo from CamInfoCache: ", string(b))
				return string(b)
			}
		}
	}
	return "[pipeline info not found!]"
}

func (st *PipelineCache) _appendIfMissing(lc AppLoggingClient, pipelines []PipelineInfo, item PipelineInfo) []PipelineInfo {
	for _, ele := range pipelines {
		if ele.SerialNumber == item.SerialNumber {
			lc.Debug(fmt.Sprintf("Found EXISTING pipeline in cache with serialNum: %v", item.SerialNumber))
			return pipelines
		}
	}
	lc.Debug(fmt.Sprintf("Adding NEW pipeline to cache with serialNum: %v", item.SerialNumber))
	return append(pipelines, item)
}

// AddVaPipeline appends pipeline device to Va cache if not existing - cache invalidation tbd
func (st *PipelineCache) AddVaPipeline(lc AppLoggingClient, item PipelineInfo) []PipelineInfo {
	st.VAPipelineInfo = st._appendIfMissing(lc, st.VAPipelineInfo, item)
	return st.VAPipelineInfo
}
