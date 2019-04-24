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
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

const (
	uriPipeline="/api/v1/pipeline"
)

// PipelineInfo holds VA pipeline device details.
type PipelineInfo struct {
	SerialNumber	string					`json:"serialnumber"`
	Name     		string                 	`json:"name"`
	Tags     		map[string]interface{} 	`json:"tags,omitempty"`
}

func (p *VideoAnalyticsProvider) getVAPipelineDetails(address string) (PipelineInfo, error) {
	p.lc.Debug(fmt.Sprintf("Querying VA Pipeline details from: %v", address))
	client := &http.Client{}
	url := "http://" + address + uriPipeline
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return PipelineInfo{}, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return PipelineInfo{}, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return PipelineInfo{}, err
	}
	entries := strings.Split(string(body), "\n")
	entries = entries[:len(entries)-1]
	for i := range entries {
		result := strings.Split(entries[i], "=")
		if len(result) > 1 {
			entries[i] = result[1]
		} else {
			err := fmt.Errorf("Invalid data structure returned from Pipeline API for host: %v", address)
			return PipelineInfo{}, err
		}
	}
	pipelineInfo := PipelineInfo{
		Name:       address,
//		....
	}
	p.lc.Debug(fmt.Sprintf("%v", pipelineInfo))
	return pipelineInfo, nil
}
