# Normal Framework - Independent Data Layer 

This directory is an example of how to set up an Independent Data Layer (IDL) using Normal Framework. The entry point is a `docker-compose` file which defines the various required services and connections.

## Running this example
1. Copy the files in this directory
2. Run `docker-compose pull && docker-compose up -d`

## Overview
In this example setup, the `Historian` service listens for MQTT Sparkplug and adds them to the [Timescale](https://www.timescale.com/) database. [Mosquitto](https://mosquitto.org/) is the MQTT message broker. The IDL data ultimately resides in the Timescale database which can be queried by dependent services.

## User Interface
This example setup uses [Grafana](https://grafana.com/), with a few pre-defined dashboards for displaying the IDL data. Additional Dashboards can be configured through the Grafana User Interface, or by changing the default configuration in the `./dashboards/normal` directory.

When running locally, you can visit Grafana at http://localhost:3000, and Normal Framework at http://localhost:8080.  You can also connect to TimescaleDB or Mosquitto, to explore connecting other solutions.

## Moving to Production
This solution template is intended to be an example of how Normal can be used to build an end-to-end IDL solution.  Before deploying this in production, you should ensure you've considered:

   * Creating persistent volumes for Redis, Timescale, and Normal so data are persistent
   * Securing the connections between the various components by changing the password defaults and creating firewall rules.
   * Ensure you have a back up strategy for your valuable data.  For instance, you could move the TimescaleDB to a cloud service like [Timescale Cloud](https://www.timescale.com/cloud) which will ensure you have highly available storage for your data.
