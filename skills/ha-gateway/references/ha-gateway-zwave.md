# Z-Wave (Z-Wave JS)

Read when adding, removing, checking or configuring Z-Wave devices. Back to [`ha-gateway`](../SKILL.md).

Home Assistant reaches Z-Wave through a Z-Wave JS server (an add-on or a separate container) that owns the USB stick. The `zwave_js` entry is the connection to that server; if it is not `loaded`, the server or the stick is down, and no Z-Wave command below works.

## Adding and removing devices

Inclusion (adding) and exclusion (removing) stream events for minutes and, for secure S2 devices, ask the user for the PIN printed on the device in the middle. They cannot be carried by one `raw ws` call. Send the user to Settings > Devices & services > Z-Wave > Add device (or Remove device), and stay available for the steps after: naming and placing the device in [`ha-registry`](../../ha-registry/SKILL.md).

## Checking

```bash
hass-cli integration list --domain zwave_js                                        # the entry_id
hass-cli raw ws zwave_js/network_status --data '{"entry_id":"<entry_id>"}' -o json # controller and node summary
hass-cli raw ws zwave_js/node_status --data '{"device_id":"<device_id>"}' -o json  # one node: status alive, asleep or dead
```

- A node that is `asleep` is a battery device between wake-ups; that is normal, and commands to it wait for its next wake-up.
- A node that is `dead` stopped answering. Check its power first; a mains device that stays dead after a power cycle can be removed as a failed node in the frontend.
- Network-level commands take the entry's `entry_id`; node-level commands take the Home Assistant `device_id`. A refused field names the one it wanted.

## Device parameters and values

Configuration parameters (an LED colour, a motion timeout) and refreshing a stale value are services:

```bash
hass-cli service call zwave_js.set_config_parameter --data '{"entity_id":"light.hall_dimmer","parameter":3,"value":1}'
hass-cli service call zwave_js.refresh_value --data '{"entity_id":"sensor.hall_temperature"}'
```

Parameter numbers and their values are specific to each device model and come from its manual or the device page; do not guess them. `hass-cli service describe zwave_js.set_config_parameter` lists the service's fields.

## Controller operations

Rebuilding routes, backing up or restoring the stick's NVM and resetting the controller exist as `zwave_js/...` commands, but they stream progress or erase the network. Leave them to the frontend, and a controller reset only on the user's explicit request with a warning that every device will need to be included again.
