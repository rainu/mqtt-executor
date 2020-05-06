# mqtt-executor

![Go](https://github.com/rainu/mqtt-executor/workflows/Go/badge.svg)

A simple MQTT client written in go that subscribes to a configurable list of MQTT topics on the specified broker and 
executes a given shell script/command whenever a message arrives. Furthermore you can define commands that are executed 
in a configurable interval. This can be useful for sensor data. Also this application supports [Homeassitant](https://www.home-assistant.io)
[mqtt descovery](https://www.home-assistant.io/docs/mqtt/discovery/).

# Get the Binary
You can build it on your own (you will need [golang](https://golang.org/) installed):
```bash
go build -a -installsuffix cgo ./cmd/mqtt-executor/
```

Or if you are **logged in** at github, you can download the pre-build artifacts: [here](https://github.com/rainu/mqtt-executor/actions?query=is%3Asuccess+branch%3Amaster)

# Configuration
Create a configuration file named "config.json"
```json5
{
  "availability": {
    "topic": "tele/__DEVICE_ID__/status",
    "interval": "10s",
    "payload": {
      "available": "On",
      "unavailable": "Off"
    }
  },
  "trigger": [{
    "name": "Touch file",
    "topic": "cmnd/touch/file",
    "command": {
      "name": "/usr/bin/touch",
      "arguments": ["/tmp/example"]
    }
  }],
  "sensor": [{
    "name": "Free Memory",
    "topic": "tele/__DEVICE_ID__/memory/free",
    "unit": "kB",       //used for hassio
    "icon": "hass:eye", //used for hassio
    "interval": "10s",
    "command": {
      "name": "/bin/sh",
      "arguments": [
        "-c",
        "cat /proc/meminfo | grep MemFree | cut -d\\: -f2 | sed 's/ //g' | grep -o [0-9]*"
      ]
    }
  }]
}
```

# Usage

Start the tool with the path to the config file and the URL of the MQTT broker
```bash
mqtt-executor -broker tcp://127.0.0.1:1883 -config /path/to/config.json
```

Enable the Homeassitant discovery support
```bash
mqtt-executor -broker tcp://127.0.0.1:1883 -config /path/to/config.json -home-assistant
```

## Trigger command execution

To execute a trigger:

```bash
mosquitto_pub -t cmnd/touch/file -m "START"
```

To interrupt a trigger
```bash
mosquitto_pub -t cmnd/touch/file -m "STOP"
```

Read the trigger state:
```bash
mosquitto_sub -t cmnd/touch/file/STATE
```
* RUNNING -> the command is running
* STOPPED -> the command has (not) been executed

Read the trigger result (command's output):
```bash
mosquitto_sub -t cmnd/touch/file/RESULT
```

### Get the trigger results

Read the trigger state:
```bash
mosquitto_sub -t cmnd/touch/file/STATE
```
* RUNNING -> the command is running
* STOPPED -> the command has (not) been executed

Read the trigger result (command's output):
```bash
mosquitto_sub -t cmnd/touch/file/RESULT
```

## Get the sensor results

Read the trigger state:
```bash
mosquitto_sub -t tele/+/memory/free
```
