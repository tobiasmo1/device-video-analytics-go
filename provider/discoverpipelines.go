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
	"bytes"
	"fmt"
	"github.com/edgexfoundry/device-sdk-go"
	contract "github.com/edgexfoundry/go-mod-core-contracts/models"
	"log"
	"strconv"
	"strings"
	"time"
)

const (
	scanOnStartup         = true
	showTerminalCountdown = true
	countdownFreqSeconds  = 1
)

type PipelineSource struct {
	Name             string
	DeviceNamePrefix string
	ProfileName      string
	DefaultPort      int
}

// Options contains input parameters and runtime objects to simplify access by device service methods.
// Since Options is a member of VideoAnalyticsProvider, consider  as appropriate.
type Options struct {
	Interval         int      // Interval, in seconds, between device scans
	ScanDuration     string   // Duration to permit for network scan - "40s"  "10m"
	Credentials      string   // Credentials/DevKey to connect to VA Pipeline service
	// SupportedSources []VAPipelineSource  flattened with properties below for now
	Name             string // name of pipeline
	DeviceNamePrefix string // prefix of pipeline id
	ProfileName      string // ffmpeg vs gstreamer - will they have distinct edgex profiles?
}/*
type Options struct {
	Interval         int      // Interval, in seconds, between device scans
	ScanDuration     string   // Duration to permit for network scan - "40s"  "10m"
	IP               string   // IP subnet(s) to use for network scan
	NetMask          string   // Network mask to use for network scan
	SourceFlags      []string // Array holding vendor(s) and port(s) for scan
	Credentials      string   // Credentials to connect to VA Pipeline service endpoints
	SupportedSources []PipelineSource
}*/

func (p *VideoAnalyticsProvider) _terminalCountdown(doneChan *chan bool, ticker **time.Ticker, activity string, durCountdown string) error {
	*ticker = time.NewTicker(countdownFreqSeconds * time.Second)
	*doneChan = make(chan bool)
	go func(doneChan chan bool) {
		var pct int
		countdownDur, err := time.ParseDuration(durCountdown)
		if err != nil {
			p.lc.Error(fmt.Sprintf("Error parsing duration: %v", err.Error()))
		}
		countdownSecs := countdownDur.Seconds()
		remaining := "remaining"
		progressBar := ""
		divScan := 20
		for {
			if *ticker != nil {
				select {
				case <-(*ticker).C:
					if countdownSecs >= 0 {
						pct = 100 - int(100*(countdownSecs/countdownDur.Seconds()))
					} else {
						remaining = "\033[41moverdue\033[0m"
						pct = 99
					}
					progressBar = strings.Repeat("#", ((divScan/10)*pct)/10)
					fmt.Printf("\033[2K\r[%-20s] %s %d%% complete (%v seconds %s)", progressBar, activity, pct, countdownSecs, remaining)
					countdownSecs -= countdownFreqSeconds
				case <-doneChan:
					if countdownSecs >= 0 {
						remaining = "early"
					}
					progressBar = strings.Repeat("#", divScan)
					fmt.Printf("\033[2K\r[%-20s] %s 100%% complete (%v seconds %s)", progressBar, activity, countdownSecs, remaining)
					fmt.Println()
					return
				}
			}
		}
	}(*doneChan)
	return nil
}


func createKeyValuePairString(m map[string]interface{}) string {
	b := new(bytes.Buffer)
	counter := 0
	for k, v := range m {
		counter++
		if counter > 1 {
			if _, err := fmt.Fprintf(b, ","); err != nil {
				log.Fatalf("failed to write to console: %s", err)
			}
		}
		switch v.(type) {
		case float64:
			if _, err := fmt.Fprintf(b, "'%s':%v", k, v); err != nil {
				log.Fatalf("failed to write float64 to console: %s", err)
			}
		case bool:
			if _, err := fmt.Fprintf(b, "'%s':%v", k, v); err != nil {
				log.Fatalf("failed to write bool to console: %s", err)
			}
		case string:
			if _, err := fmt.Fprintf(b, "'%s':'%v'", k, v); err != nil {
				log.Fatalf("failed to write string to console: %s", err)
			}
		}
	}
	return b.String()
}

func (p *VideoAnalyticsProvider) discoverHosts(options Options) (discoveredHosts []string, err error) {
	p.lc.Info(fmt.Sprintf("\033[34mScanning configured VA Pipeline hosts with %v timeout...\033[0m", options.ScanDuration))
	var doneChan chan bool
	if showTerminalCountdown {
		err := p._terminalCountdown(&doneChan, &(p.scanDurationTicker), "Scanning", options.ScanDuration)
		if err != nil {
			fmt.Println("ERROR: during terminal progress bar" + err.Error())
		}
	}

	discoveredHosts = []string{"http://localhost",}

	if showTerminalCountdown {
		p.scanDurationTicker.Stop()
		doneChan <- true
		close(doneChan)
		time.Sleep(1 * time.Second)
	}
	p.lc.Debug(fmt.Sprintf("Scan found %d VA Pipeline hosts", len(discoveredHosts)))
	return discoveredHosts, err
}

// DiscoverDevices is a scheduled action that performs registration of pipelines, population of memory cache,
// registration with EdgeX and persistance of cache to disk.
func (p *VideoAnalyticsProvider) DiscoverDevices(options Options) (int, error) {
	deviceCountTotal := 0
	// assume a single VA Pipeline service for now
	discoveredHosts, err := p.discoverHosts(*p.options)
	if err != nil {
		p.lc.Error(err.Error())
		return 0, err
	}
	if len(discoveredHosts) > 0 {
		p.lc.Info(fmt.Sprintf("Scanning %d discovered hosts for requested pipeline [device] signatures...", len(discoveredHosts)))
		deviceCountAxis, err2 := p.addPipelinesAsEdgexDevices(p.options.Name, p.options.ProfileName, discoveredHosts)
		if err2 != nil {
			p.lc.Error(err2.Error())
			return deviceCountTotal, err2
		}
		deviceCountTotal += deviceCountAxis
	} else {
		p.lc.Debug("No hosts discovered listening on requested ports. Aborted the scan for requested compatible pipeline devices.")
	}
	return deviceCountTotal, nil
}

// addPipelineAsEdgexDevices populates the provided discoveredPipelines empty array of PipelineInfo structure with
// metadata about the requested Pipeline device.
func (p *VideoAnalyticsProvider) addPipelinesAsEdgexDevices(pipelineSourceName string, vendorProfile string, discoveredHosts []string) (int, error) {
	deviceCountVendor := 0

		p.lc.Info(fmt.Sprintf("Checking for Pipelines on pipelineSource [%s]...", pipelineSourceName))
		discoveredPipelines := p.getPipelineMetadata(discoveredHosts)
		p.lc.Info(fmt.Sprintf("\033[32mDiscovered: %v %s pipelines\033[0m", len(discoveredPipelines), pipelineSourceName))
		p.lc.Debug(fmt.Sprintf("Attaching Tags to discovered %s pipelines...", pipelineSourceName))
		p.assignTags(discoveredPipelines, p.ac.TagCache)
		p.lc.Debug(fmt.Sprintf("Storing %s metadata to cache...", pipelineSourceName))
		p.memoizePipelineInfo(discoveredPipelines, p.ac.PipelineCache)
		p.lc.Debug(fmt.Sprintf("Adding EdgeX %s pipelines...", pipelineSourceName))
		ids, err := p.addEdgeXPipelineDevices(discoveredPipelines, vendorProfile)
		if err != nil {
			p.lc.Debug(err.Error())
			return deviceCountVendor, err
		}
		deviceCountVendor += len(ids)
		p.lc.Debug(fmt.Sprintf("Finished adding %d new %s Pipelines to EdgeX: %v", len(ids), pipelineSourceName, strings.Join(ids, ",")))
		err = p.saveToFile(pipelineSourceName)
		if err != nil {
			p.lc.Debug("Error persisting device info: " + err.Error())
			return deviceCountVendor, err
		}

	return deviceCountVendor, nil
}

func (p *VideoAnalyticsProvider) saveToFile(pipelineSourceName string) error {
	var err error
	p.lc.Debug(fmt.Sprintf("Saving %s PipelineInfo cache to disk", pipelineSourceName))
	err = p.ac.PipelineCache.SaveInfo(p.ac.InfoFileVAPipelines)
	if err != nil {
		p.lc.Error(err.Error())
	} else {
		p.lc.Info(fmt.Sprintf("Saved %s PipelineInfo cache to disk", pipelineSourceName))
	}
	return err
}

func (p *VideoAnalyticsProvider) addEdgeXPipelineDevices(pipelines []PipelineInfo, profileName string) ([]string, error) {
	var idstr string
	var err error
	var ids []string
	var counter int
	// Add each discovered pipeline as an EdgeX device, managed by this device service
	deviceNamePrefix := p.options.DeviceNamePrefix
	if profileName == p.options.ProfileName {
		deviceNamePrefix = p.options.DeviceNamePrefix
	}
	for i := range pipelines {
		var edgexDevice contract.Device
		if pipelines[i].SerialNumber == "" {
			p.lc.Error(fmt.Sprintf("ERROR adding EdgeX pipeline device. Check credentials. No serial number at index: %v", i))
		} else {
			// Transfer all Tags to EdgeX device labels...
			labels := make([]string, len(pipelines[i].Tags))
			var value string
			counter = 0
			for tagKey, tagVal := range pipelines[i].Tags {
				value = fmt.Sprintf("%v", tagVal)
				labels[counter] = tagKey + ":" + value
				p.lc.Debug(fmt.Sprintf("processing tag @ idx: %v", tagKey))
				counter++
			}
			deviceName := deviceNamePrefix + pipelines[i].SerialNumber
			edgexDevice, err = device.RunningService().GetDeviceByName(deviceName)
			if err != nil {
				// In this case expect EdgeX error:
				// "Device 'edgex-pipeline-<SerialNumber>' cannot be found in cache"
				p.lc.Info("GetDeviceByName for device " + deviceName + " ErrResponse: " + err.Error())

				// TODO: Identify why ProtocolProperties are required, fails to create device if missing
				edgexDevice = contract.Device{
					Name:           deviceName,
					AdminState:     contract.Unlocked,
					OperatingState: contract.Enabled,
					Protocols:      p.getProtocols(),
					Labels: labels,
					//Location: tag.deviceLocation,
					Profile: contract.DeviceProfile{
						Name: profileName,
					},
					Service: contract.DeviceService{
						AdminState:     contract.Unlocked,
						Service: contract.Service{
							Name:           VAPipelineManagementServiceName,
							OperatingState: contract.Enabled,
						},
					},
				}
				edgexDevice.Origin = time.Now().UnixNano() / int64(time.Millisecond)
				edgexDevice.Description = "EdgeX Discovered VA Pipeline"
				p.lc.Debug(fmt.Sprintf("Adding Device: %v", edgexDevice))

				p.lc.Debug(fmt.Sprintf("Adding NEW EdgeX device named: %s", deviceName))
				idstr, err = device.RunningService().AddDevice(edgexDevice)
				if err != nil {
					p.lc.Error(fmt.Sprintf("ERROR adding device named: %s", deviceName))
				}
				ids = append(ids, idstr)
				p.lc.Info(fmt.Sprintf("Added NEW EdgeX device named: %s", deviceName))
			} else {
				p.lc.Debug(fmt.Sprintf("Updating EXISTING EdgeX device named: %s", deviceName))
				err = device.RunningService().UpdateDevice(edgexDevice)
				if err != nil {
					p.lc.Error(fmt.Sprintf("ERROR adding/updating device named: %s", deviceName))
				} else {
					p.lc.Info(fmt.Sprintf("Updated EXISTING EdgeX device named: %s", deviceName))
				}
			}
			// TODO: Update operational state to 'disabled'
			//  a) Query EdgeX (or our local caches) for all devices by name prefix.
			//  b) Filter out the discovered devices from above
			//  c) Assign remaining devices as 'disabled' if not discovered after [configurable #] scans.
		}
	}
	return ids, err
}

func (p *VideoAnalyticsProvider) assignTags(pipelines []PipelineInfo, tagCache *Tags) {
	if tagCache == nil {
		p.lc.Warn(fmt.Sprintf("TagCache is not available. Ignoring tag assignment"))
		return
	}
	for i := range pipelines {
		p.lc.Debug(fmt.Sprintf("Attaching tagCache tags for serialNumber: %v to pipeline %d", pipelines[i].SerialNumber, i))
		pipelines[i].Tags = tagCache.Tags[pipelines[i].SerialNumber]
	}
}

func (p *VideoAnalyticsProvider) memoizePipelineInfo(pipelines []PipelineInfo, pipelineCache *PipelineCache) {
	if pipelineCache == nil {
		p.lc.Warn(fmt.Sprintf("PipelineCache is not available. Ignoring pipelineInfo storage"))
		return
	}
	for i := range pipelines {
		p.lc.Debug(fmt.Sprintf("Storing PipelineCache to disk.. for device associated with serialNumber: %v pipeline index %d", pipelines[i].SerialNumber, i))
		pipelineCache.AddVaPipeline(p.lc, pipelines[i])
	}
}

// getPipelineMetadata loops across requested VA pipelines to resolve across potential hosts and updated pipeline versions, etc.
// This builds an array of PipelineInfo elements
func (p *VideoAnalyticsProvider) getPipelineMetadata(discoveredHosts []string) []PipelineInfo {
	pipelines := []PipelineInfo{}
	p.lc.Debug(fmt.Sprintf("VAPipelineDiscovery.GetPipelineMetadata scanning discovered hosts for requested pipeline signatures"))
	for _,host := range discoveredHosts {
		pipelines = p.getRequestedPipelineDetails(host, pipelines)
	}
	return pipelines
}

func isRequestedPort(sourcePorts []int, port int) bool {
	for _, p := range sourcePorts {
		if p == port {
			return true
		}
	}
	return false
}

func (p *VideoAnalyticsProvider) splitPorts(requestedPorts string) ([]int, error) {
	var err error
	a := strings.Split(requestedPorts, ",")
	b := make([]int, len(a))
	for i, v := range a {
		b[i], err = strconv.Atoi(v)
		if err != nil {
			return b, err
		}
	}
	return b, nil
}

// getRequestedPipelineDetails appends a pipeline to returned pipeline list
func (p *VideoAnalyticsProvider) getRequestedPipelineDetails(host string, pipelines []PipelineInfo) []PipelineInfo {
	p.lc.Trace(fmt.Sprintf("VAPipelineDiscovery.GetPipelineMetadata invoking getVAPipelineDetails(%s)", host))
	PipelineInfo, err := p.getVAPipelineDetails(host)
	if err != nil {
		// Continue on error - other pipelines may return valid response
		p.lc.Debug(err.Error())
	} else {
		pipelines = append(pipelines, PipelineInfo)
	}
	return pipelines
}
