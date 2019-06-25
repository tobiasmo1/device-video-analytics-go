# Device-Video-Analytics-Go

This repository contains a Video Analytics device service bridge compatible with EdgeX Foundry. The project utilizes [EdgeX Device SDK Go](https://github.com/edgexfoundry/device-sdk-go) to communicate with dependent EdgeX Go services. This means the service relies on the appropriate EdgeX services to be running in order to start up and function properly (Consul, Core Data, Core Metadata and others).

Use Device-Video-Analytics-Go to manage interactions between your EdgeX applications, IP network cameras, and Intel Video Analytics Pipelines. This includes the ability to capture still images or RTP video streams from ONVIF compliant IP cameras connected to your network. It also supports accessing Axis cameras, making available a similar but different interface to perform functions with the camera.

The project is intended to provide both functionality to serve as a springboard for visual application development at the edge, as well as a reference example of how to implement device services using the EdgeX Device SDK Go package.

# Getting Started

The instructions below are the minimal steps needed to get you up and going quickly.

Additional details and answers related to EdgeX itself are available on:

- [EdgeX Wiki](https://docs.edgexfoundry.org/Ch-GettingStartedUsers.html)

# Prerequisites - Developer Setup

Perform the following steps to set up your development system. These are necessary to build and extend the device-camera-go component.

Further detailed information is available on [EdgeX-Go repository](https://github.com/edgexfoundry/edgex-go/blob/master/docs/getting-started/Ch-GettingStartedGoDevelopers.rst).

1. Install **Go 1.12** and Glide (package manager) using instructions found here:

   - https://github.com/golang/go/wiki/Ubuntu

   - NOTE: If upgrading from an earlier version of Go, it is best to first remove it; ref: https://golang.org/doc/install

   - Install **dep** with:

     ```
     $ curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh
     ```

2. Install **Docker**.

   - If upgrading from an earlier version of Docker, BKM is to first remove it using:

     ```
     $ sudo apt-get remove docker docker-engine docker.io
     ```

   - Follow instructions from [Install Docker using repository](https://docs.docker.com/install/linux/docker-ce/ubuntu/#install-using-the-repository)

# Preparing EdgeX Stack

1. The easiest way to launch and explore device-camera-go using existing EdgeX services is to clone the edgexfoundry/developer-scripts repository and run the published Docker containers for the **EdgeX Delhi 0.7.1 release**.

   ```
   $ cd ~/dev
   $ git clone https://github.com/edgexfoundry/developer-scripts.git
   $ cd ~/dev/developer-scripts/compose-files
   ```

   At time of writing, you will see that the latest docker-compose.yml is equivalent to Delhi 0.7.1.

2. The device-camera-go device service does not require other EdgeX device services. To reduce runtime footprint a bit, and possible confusion about new devices, comment out the device-virtual section (using # at beginning of the line) in **./compose-files/docker-compose.yml**. 

   ```
   # device-virtual:
   #   image: edgexfoundry/docker-device-virtual:0.6.0
   #   ports:device-virtual:
   ...
   ```

3. If you are already running MongoDB on its default port (27017), you will want to also update **./compose-files/docker-compose.yml** to assign a unique port for EdgeX MongoDB (e.g., 27018 as the port referenced by your local system):

   ```
   mongo:
     ...
     ports:
      - "27018:27017"
   ```

4. The first time launching EdgeX, Docker will automatically **pull the EdgeX images** to your system. This requires an Internet connection and allowance through proxies/firewall. Detailed information about pulling EdgeX containers is available here: https://docs.edgexfoundry.org/Ch-GettingStartedUsersNexus.html

# Preparing Device-SDK-Go

1. Get latest device-camera-go source code:

   ```
   cd $GOPATH/src/github.com/edgexfoundry-holding
   git clone https://github.com/edgexfoundry-holding/device-camera-go.git
   ```

   Note: You may alternately clone to a separate project folder so long as you create a symlink to it from $GOPATH/src/github.com/edgexfoundry-holding folder.

2. In GoLand, load device-camera-go project, execute the following within View/Tools/Terminal:

   ```
   $ make prepare
   $ make build
   ```


# Running device-video-analytics-go

1. **Launch the EdgeX stack** by instructing Docker to use developer-scripts/compose-files/docker-compose.yml to run the services in detached state. This way control will return to your terminal window. Of course you can leave off the -d parameter and open additional terminals if you prefer watching EdgeX-specific logging activities in real time.

   ```
   $ cd ~/dev/developer-scripts/compose-files
   $ docker-compose up -d
   ```

2. **Confirm EdgeX services** are running properly by navigating to the Consul dashboard in your browser:

   - http://127.0.0.1:8500/ui/#/dc1/services

     [image of green/passing services]

3. (optional) **Modify your /etc/hosts file** to override DNS resolution to resolve EdgeX service URLs. This will allow you to negotiate consistently whether initiating commands from your host computer or in the docker network established by EdgeX. Add a single line below your existing localhost entry: 

   > 127.0.0.1	localhost
   > 127.0.0.1	edgex-core-command edgex-core-metadata edgex-core-data
   >
   > ...

   Alternately, adjust EdgeX service URLs used in this guide and the tests/postman import file so they point to  "127.0.0.1", "localhost". 

4. **Modify the ./run.sh script** with appropriate options (described below) and then launch the device-camera-go service.
   NOTE: Be sure to update the parameters you supply to device-camera-go with expected sources and IP address ranges according to your network topology and IP camera deployment and configuration:

   ```
   $ cd $GOPATH/src/github.com/edgexfoundry-holding/device-camera-go
   $ make run
   $ ./run
   ```


## Known Issues and Improvement Opportunities

Please add as "issues" in github above ^ 



