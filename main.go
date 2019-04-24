package main

import (
	"flag"
	"fmt"
	"github.com/edgexfoundry/device-sdk-go"
	ds_models "github.com/edgexfoundry/device-sdk-go/pkg/models"
	"github.com/edgexfoundry-holding/device-video-analytics-go/videoanalyticsprovider"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"
)

const (
	version     string = "0.1"
	serviceName string = "device-video-analytics-go"
)

var (
	confProfile string
	confDir     string
	useRegistry bool
)

// SourceList represents the parameters instructing to scan for available VA Pipelines
type SourceList []string

var sourceFlags SourceList

func (list *SourceList) String() string {
	return ""
}

// Set is overriding SourceList interface method
func (list *SourceList) Set(val string) error {
	*list = append(*list, val)
	return nil
}

func main() {
	// Default device-video-analytics-go parameters

	// IP Range default
	scanDuration := "15s"
	interval := 60

	vaInfoFile := "./res/va-pipelineinfo-cache.json"
	tagsFile := "./res/tags.json"
	vaCredentialsFile := "./res/va-credentials.conf"
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	// Process EdgeX DeviceService parameters
	flag.BoolVar(&useRegistry, "registry", false, "Indicates the service should use the registry.")
	flag.BoolVar(&useRegistry, "r", false, "Indicates the service should use registry.")
	flag.StringVar(&confProfile, "profile", "", "Specify a profile other than default.")
	flag.StringVar(&confProfile, "p", "", "Specify a profile other than default.")
	flag.StringVar(&confDir, "confdir", "", "Specify an alternate configuration directory.")
	flag.StringVar(&confDir, "c", "", "Specify an alternate configuration directory.")

	// Process VAPipelineDiscovery parameters
	flag.StringVar(&scanDuration, "scanduration", scanDuration, "Duration to permit for each network discovery scan; -scanduration \"10s\"")
	flag.IntVar(&interval, "interval", interval, "Interval between discovery scans, in seconds. Must be > scanduration; -interval 180")
	flag.Var(&sourceFlags, "source", "source to scan, -source http://localhost,http://10.24.25.26")
	flag.StringVar(&tagsFile, "tagsFile", tagsFile, "Location of file where VA Pipeline tags are cached")
	flag.StringVar(&vaCredentialsFile, "vapipelinecredentials", vaCredentialsFile,
		"Path to file containing credentials for accessing VA Pipeline endpoints (user and password, tab-separated)")
	flag.Parse()

	camInfoCache := videoanalyticsprovider.PipelineCache{}
	tagCache := videoanalyticsprovider.Tags{}

	options := videoanalyticsprovider.Options{
		Interval:         	interval,
		ScanDuration:     	scanDuration,
		Credentials:      	vaCredentialsFile,
		Name:				"vapipename",
		DeviceNamePrefix:   "prefix",
		ProfileName: 		"profilename",
	}
	ac := videoanalyticsprovider.AppCache{
		PipelineCache:  &camInfoCache,
		InfoFileVAPipelines:  vaInfoFile,
		TagCache:      &tagCache,
		TagsFile:      tagsFile,
	}

	sd := videoanalyticsprovider.VideoAnalyticsProvider{}
	// NOTE: If we were only interested to load device configurations at startup using .YML resources, we could use:
	// startup.Bootstrap(serviceName, version, &sd)

	if err := startService(serviceName, version, &sd, &options, &ac); err != nil {
		Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// Fprintf method overrides fmt.Fprintf to perform error check in one place
//func Fprintf(w io.Writer, format string, a ...interface{}) (n int, err error) {
func Fprintf(w io.Writer, format string, a ...interface{}) {
	_, err := fmt.Fprintf(w, format, a...)
	if err != nil {
		log.Fatalf("Fatal error accessing output target: %s", err)
	}
}

func startService(serviceName string, serviceVersion string, driver ds_models.ProtocolDriver, options *videoanalyticsprovider.Options, ac *videoanalyticsprovider.AppCache) error {
	// Instantiate our VAPipeline provider (this service) as a "device"
	videoanalyticsprovider := videoanalyticsprovider.New(options, ac)
	// Create EdgeX service and pass in the provider "device"
	deviceService, err := device.NewService(serviceName, serviceVersion, confProfile, confDir, useRegistry, videoanalyticsprovider)
	if err != nil {
		return err
	}
	errChan := make(chan error, 2)
	listenForInterrupt(errChan)
	// Start the EdgeX service
	Fprintf(os.Stdout, "Calling service.Start.\n")
	err = deviceService.Start(errChan)
	if err != nil {
		return err
	}
	err = <-errChan
	Fprintf(os.Stdout, "Terminating: %v.\n", err)
	return deviceService.Stop(false)
}

func listenForInterrupt(errChan chan error) {
	go func() {
		// Set up channel on which to send signal notifications.
		// We must use a buffered channel or risk missing the signal
		// if we're not ready to receive when the signal is sent.
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		errChan <- fmt.Errorf("%s", <-c)
	}()
}
