---
name: ha-gateway
version: "0.0.0"
description: "Zigbee (ZHA), Z-Wave, Matter and Thread networks: pair or add a new device, let devices join, remove a device from the network, check the coordinator, dongle or border router, back up a Zigbee network."
compatibility: Requires hass-cli on PATH
metadata:
  requires:
    bins: ["hass-cli"]
---

# ha-gateway

> **Shared front door:** read [`../ha-shared/SKILL.md`](../ha-shared/SKILL.md) first; it holds the two-command control and the rules every request follows.

A gateway is a radio (a Zigbee or Z-Wave USB stick, a Matter server, a Thread border router) whose network carries devices that have no IP address of their own. In Home Assistant the gateway is one config entry, and every device on its network is a device of that entry. The entry's lifecycle (add, reload, remove the whole integration) is [`ha-integrations`](../ha-integrations/SKILL.md); naming a paired device and putting it in a room is [`ha-registry`](../ha-registry/SKILL.md). This skill is the network itself: joining, removing and inspecting devices on it.

## Fast paths

| The user wants | Read | Ends at |
|---|---|---|
| Pair a new Zigbee device, "let devices join" | [Zigbee](references/ha-gateway-zigbee.md) | `service call zha.permit` |
| Remove a Zigbee device from the network | [Zigbee](references/ha-gateway-zigbee.md) | `service call zha.remove` |
| Add or remove a Z-Wave device, a dead Z-Wave node | [Z-Wave](references/ha-gateway-zwave.md) | the frontend for inclusion; `raw ws` for status |
| Add a Matter device with its pairing code | [Matter and Thread](references/ha-gateway-matter-thread.md) | `raw ws matter/commission` |
| Which Thread network or border router is in use | [Matter and Thread](references/ha-gateway-matter-thread.md) | `raw ws thread/list_datasets` |
| "Is the Zigbee stick working", a whole network offline | this file | `integration list --domain zha`, then [`ha-integrations`](../ha-integrations/SKILL.md) |

## Find the gateway

```bash
hass-cli integration list --domain zha          # also zwave_js, matter, thread, otbr
hass-cli system hardware -o json                # the USB radio, on HA OS and Home Assistant hardware
```

- No entry for the domain: that radio is not set up here, and its commands answer `unknown_command`. Adding it is a config flow in [`ha-integrations`](../ha-integrations/SKILL.md).
- An entry not `loaded` takes every device on that network down with it. Fix the entry first; pairing cannot work until it is `loaded`.
- Zigbee addresses a device by its `ieee` (`00:0d:6f:00:0a:90:69:e7`); Z-Wave and Matter commands take the Home Assistant `device_id`, which `hass-cli registry device list -o json` shows next to the device's name.

## What the CLI can and cannot drive

The protocol commands have no typed hass-cli verb. One-shot services (`zha.permit`, `zha.remove`, `zwave_js.set_config_parameter`) go through `service call`; single-answer WebSocket commands through `raw ws`, as [raw access](../ha-shared/references/ha-shared-raw.md) describes.

Some operations are **subscriptions** that stream progress over minutes: Z-Wave inclusion and exclusion with its S2 PIN exchange, a Z-Wave NVM backup, discovering Thread routers. `raw ws` returns the first answer and disconnects, so it cannot carry them. For those, open Home Assistant's own pairing screen (Settings > Devices & services > the integration > Add device) and tell the user that is where to finish.

## Verb index

| Command | For | Read when triggered |
|---|---|---|
| `service call zha.permit`, `zha.remove` | Zigbee join window, removal | [Zigbee](references/ha-gateway-zigbee.md) |
| `raw ws zha/...` | Zigbee devices, network settings, coordinator backups | [Zigbee](references/ha-gateway-zigbee.md) |
| `raw ws zwave_js/...`, `service call zwave_js.*` | Z-Wave status, node health, device parameters | [Z-Wave](references/ha-gateway-zwave.md) |
| `raw ws matter/...`, `thread/...`, `otbr/...` | Matter commissioning, Thread datasets, the border router | [Matter and Thread](references/ha-gateway-matter-thread.md) |
| `integration list --domain <radio>` | whether the gateway itself is up | the gateway section above |

Zigbee2MQTT is not a Home Assistant radio integration: its devices arrive through MQTT, and pairing is done in Zigbee2MQTT's own frontend. Its devices are controlled like any other once they exist.

## Safety and escalation

- An open join window lets any nearby device of that protocol join. Open it only when the user is about to pair, for the shortest time that fits (60 seconds is usual), and say when it closes.
- Removing a device from the network (`zha.remove`, a Z-Wave exclusion, a Matter fabric removal) needs the user to name that device; afterwards it must be reset and paired again to come back.
- Changing a Zigbee channel, restoring a coordinator backup, forming a new Thread network or resetting a Z-Wave controller can drop every device on the network. Do these only on an explicit request, and say before acting that devices may need re-pairing.
- Pairing codes and install codes come from the device or the user; never guess one. Two refused codes in a row: stop and ask the user to read it again.
