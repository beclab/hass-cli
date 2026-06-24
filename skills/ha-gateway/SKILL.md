---
name: ha-gateway
version: 0.1.0
description: "Manage Home Assistant radio gateways and their networks with hass-cli: Zigbee (ZHA), Z-Wave JS, Matter, and Thread/OTBR. Use to pair/remove a Zigbee device, permit join, add/exclude a Z-Wave node, read network status/topology, back up a Zigbee/Z-Wave coordinator, commission a Matter device, or inspect Thread datasets. The gateway integration's own lifecycle (add/reload/delete) lives in ha-integrations; the devices it exposes live in ha-registry."
metadata:
  requires:
    bins: ["hass-cli"]
  cliHelp: "hass-cli integration --help"
---

# ha-gateway

Radio gateways (a.k.a. coordinators / hubs / bridges) and their device networks.
**Prerequisite:** [`../ha-shared/SKILL.md`](../ha-shared/SKILL.md).

A gateway in Home Assistant is an **integration (config entry)** that owns a set
of **devices**. So split the work:

- Gateway integration lifecycle (add / reload / enable / disable / delete) ->
  [`../ha-integrations/SKILL.md`](../ha-integrations/SKILL.md).
- Renaming / assigning areas to the devices behind it ->
  [`../ha-registry/SKILL.md`](../ha-registry/SKILL.md).
- **Protocol-layer operations (pairing, joining nodes, networks, backups)** ->
  this skill.

These are radio-specific WebSocket commands and services that have no typed
hass-cli command, so they are driven with `service call` and `raw ws`.

## Find the gateway

```bash
# which radio gateways exist
hass-cli integration list -o json | rg '"domain": "(zha|zwave_js|matter|thread|otbr)"'
# the devices a gateway exposes
hass-cli registry device list -o json | rg -i 'zha|zwave|matter|thread'
```

Z-Wave/Matter WS commands key off the gateway's `entry_id` (from
`integration list`); ZHA addresses devices by their `ieee`.

## Zigbee (ZHA)

Pairing and removal are **services** (one-shot, CLI-friendly):

```bash
hass-cli service call zha.permit --data '{"duration":60}'          # open join window
hass-cli service call zha.permit --data '{"source_ieee":"xx:..","install_code":"..."}'
hass-cli service call zha.remove --data '{"ieee":"00:0d:6f:..."}'  # remove a device
hass-cli service call zha.reconfigure_device --data '{"ieee":"00:0d:6f:..."}'
```

Inspect and manage the network (WebSocket):

```bash
hass-cli raw ws zha/devices                                        # all Zigbee devices
hass-cli raw ws zha/device --data '{"ieee":"00:0d:6f:..."}'        # one device
hass-cli raw ws zha/network/settings                               # coordinator/network info
hass-cli raw ws zha/topology/update                                # refresh the mesh map
hass-cli raw ws zha/network/change_channel --data '{"new_channel":20}'
# coordinator backups
hass-cli raw ws zha/network/backups/list
hass-cli raw ws zha/network/backups/create
hass-cli raw ws zha/network/backups/restore --data '{"backup":{...}}'
```

## Z-Wave JS

Inclusion/exclusion and controller ops are WebSocket; most take `entry_id`:

```bash
ENTRY=$(hass-cli integration list --domain zwave_js -o json | jq -r '.[0].entry_id')
hass-cli raw ws zwave_js/network_status --data "{\"entry_id\":\"$ENTRY\"}"
hass-cli raw ws zwave_js/add_node --data "{\"entry_id\":\"$ENTRY\"}"        # start inclusion
hass-cli raw ws zwave_js/stop_inclusion --data "{\"entry_id\":\"$ENTRY\"}"
hass-cli raw ws zwave_js/remove_failed_node --data "{\"entry_id\":\"$ENTRY\",\"node_id\":5}"
hass-cli raw ws zwave_js/node_status --data "{\"entry_id\":\"$ENTRY\",\"node_id\":5}"
hass-cli raw ws zwave_js/rebuild_node_routes --data "{\"entry_id\":\"$ENTRY\",\"node_id\":5}"
# controller NVM backup / restore, factory reset
hass-cli raw ws zwave_js/backup_nvm --data "{\"entry_id\":\"$ENTRY\"}"
hass-cli raw ws zwave_js/hard_reset_controller --data "{\"entry_id\":\"$ENTRY\"}"
```

Device values and config parameters are services:

```bash
hass-cli service call zwave_js.set_config_parameter --data '{"entity_id":"...","parameter":3,"value":1}'
hass-cli service call zwave_js.refresh_value --data '{"entity_id":"..."}'
```

## Matter

```bash
hass-cli raw ws matter/commission_on_network --data '{"pin":12345678}'
hass-cli raw ws matter/open_commissioning_window --data '{"device_id":"<ha_device_id>"}'
hass-cli raw ws matter/node_diagnostics --data '{"device_id":"<ha_device_id>"}'
hass-cli raw ws matter/remove_matter_fabric --data '{"device_id":"<ha_device_id>","fabric_index":N}'
```

## Thread / OpenThread Border Router (OTBR)

```bash
hass-cli raw ws thread/list_datasets                    # known Thread networks
hass-cli raw ws thread/discover_routers                 # mDNS border routers
hass-cli raw ws thread/set_preferred_dataset --data '{"dataset_id":"..."}'
hass-cli raw ws otbr/info                               # local border router
hass-cli raw ws otbr/create_network
hass-cli raw ws otbr/set_channel --data '{"channel":15}'
```

## Limits (read before promising an outcome)

- **Interactive / streaming flows** (Z-Wave inclusion DSK + S2 PIN exchange, ZHA
  `zha/devices/reconfigure` progress, Matter/Z-Wave commissioning progress) are
  WebSocket *subscriptions*. `raw ws` issues a single command and returns one
  result; it can kick these off but cannot carry a multi-message exchange. For a
  full secure pairing, finish in the HA frontend or open the join window with
  `zha.permit` and let the device join.
- **Cloud gateways** (Hue Cloud, Nest, ...) are added through an OAuth
  `external_step` that needs a browser; not completable from the CLI (see
  `ha-integrations`).
- **Zigbee2MQTT** is not a native HA radio integration; it speaks MQTT. Drive it
  with `hass-cli service call mqtt.publish` to its `zigbee2mqtt/bridge/request/*`
  topics, or use its own frontend.
- A command returns `unknown_command` / "integration not loaded" if that radio
  integration is not installed on the target instance.
