# Zigbee (ZHA)

Read when pairing, removing or inspecting Zigbee devices on Home Assistant's own Zigbee integration (ZHA). Back to [`ha-gateway`](../SKILL.md).

## Pairing

```bash
hass-cli service call zha.permit --data '{"duration":60}'
hass-cli service call zha.permit --data '{"duration":60,"ieee":"<router ieee>"}'   # join through one router, for a device far from the stick
```

The coordinator accepts new devices for `duration` seconds (up to 254). Tell the user to put the device in pairing mode now (usually a long press or a power cycle); it appears as a new device under the ZHA entry a few seconds after joining. Then:

```bash
hass-cli registry device list -o json      # the new device: its id, name and model
```

Renaming it and putting it in a room is [`ha-registry`](../../ha-registry/SKILL.md). A device that installs with a code (some Aqara and Danfoss devices) needs `source_ieee` and `install_code`, or `qr_code`; run `hass-cli service describe zha.permit` for the fields.

## Removing

```bash
hass-cli service call zha.remove --data '{"ieee":"00:0d:6f:00:0a:90:69:e7"}'
```

This tells the device to leave and drops it from Home Assistant. A battery device that is asleep may not hear it; it leaves Home Assistant anyway and rejoins only when reset and paired again.

## Inspecting

```bash
hass-cli raw ws zha/devices -o json                                   # every device: ieee, name, model, lqi, rssi, available
hass-cli raw ws zha/device --data '{"ieee":"00:0d:6f:00:0a:90:69:e7"}'
hass-cli raw ws zha/network/settings -o json                          # radio type, channel, PAN id
```

- `available: false` on one device: it is out of range, out of battery, or was reset. Several at once in one place: a router device (a mains-powered plug or bulb) between them and the stick went off.
- Low `lqi` across the board after a move: the stick is too close to a USB 3 port or Wi-Fi router; an extension cable usually fixes it. Say that rather than changing settings.

## Coordinator backups and the channel

```bash
hass-cli raw ws zha/network/backups/list
hass-cli raw ws zha/network/backups/create
```

A coordinator backup holds the network keys, which is what lets a replacement stick keep every paired device. Restoring one (`zha/network/backups/restore`) and changing the channel (`zha/network/change_channel`) affect the whole network: only on the user's explicit request, after a fresh `backups/create`.
