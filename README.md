# Plantmonitor

_Backend for a LoRaWAN-based / MQTT-based soil moisture sensor, which sends notifications via XMPP._

## Context

This project is meant to be used with `plantmonitor-sensor`. It will receive raw ADC sensor values from the TTN network MQTT broker and notify defined XMPP users when moisture thresholds are hit. 

## Features

Features of Plantmonitor:

* Receive sensor values via MQTT
* Convert raw values to normalized percentage values
* Quantify percentage values and assign a quantification level, such as "low moisture", "normal moisture" and "high moisture" level.
* Notify users via XMPP chat messages if moisture level is not "normal"
* Remind users if no action has been taken against non-normal levels for a certain period of time
* Notify users if no more sensor updates have been received 
* Respond to users via XMPP if they ask for the current status

_Chat messages can be defined via a language-specific config file. If a language is unsupported, yet, define your own chat message set!_


## Building Plantmonitor

Download source:

    git clone https://github.com/ThomasLeister/plantmonitor.git
    cd plantmonitor

Download module dependencies:

    go mod download

Run build script:

    ./build-release.sh

The `plantmonitor` binary will contain the application.


## Configuring Plantmonitor

(TBD)
* Copy config.example.yaml to config.yaml
* Set MQTT, XMPP, Giphy credentials
* Define levels
* Set language
* Define your own language file if needed.


## Running Plantmonitor

(TBD)
* Systemd Startup file
* 