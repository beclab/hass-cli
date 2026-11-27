# Matter and Thread

Read when adding a Matter device, sharing one with another app, or checking the Thread network and border router. Back to [`ha-gateway`](../SKILL.md).

Matter is the device protocol; Thread is one of the networks it runs over (Wi-Fi and Ethernet are the others). Home Assistant reaches Matter through a Matter server (an add-on or container), seen as the `matter` entry. Thread devices also need a border router: Home Assistant's own (`otbr`) or one from Apple or Google, whose network credentials are the Thread datasets.

## Adding a Matter device

```bash
hass-cli raw ws matter/commission --data '{"code":"MT:<from the QR code>"}' --timeout 120
hass-cli raw ws matter/commission_on_network --data '{"pin":20202021}' --timeout 120
```

- `commission` takes the pairing code from the device's QR code (`MT:...`) or its 11-digit manual code. Use it for a new device in pairing mode.
- `commission_on_network` takes the PIN of a device already on the network, typically one shared from another Matter app.
- Commissioning takes up to a minute or two, so raise `--timeout`. A timeout does not mean it failed: check `hass-cli registry device list` before trying again.
- A Thread device commissioned from the CLI needs Home Assistant to know the Thread network's credentials; when it fails with a network error, the Thread dataset below is usually missing, and adding the device from the Home Assistant phone app is the reliable path.

## A device already in Home Assistant

```bash
hass-cli raw ws matter/open_commissioning_window --data '{"device_id":"<device_id>"}'   # share it with Apple Home, Google Home...
hass-cli raw ws matter/node_diagnostics --data '{"device_id":"<device_id>"}' -o json
```

`open_commissioning_window` answers with a code the user enters in the other app; the window closes after a few minutes. `device_id` comes from `hass-cli registry device list -o json`.

## Thread

```bash
hass-cli raw ws thread/list_datasets -o json     # known Thread networks; one is preferred
hass-cli raw ws otbr/info -o json                # Home Assistant's own border router, when it has one
```

- Several datasets means several Thread networks (Home Assistant's and, say, an Apple TV's). Devices only reach each other within one; setting the preferred one (`thread/set_preferred_dataset`) decides where new devices go.
- Creating a new network on the border router (`otbr/create_network`) or changing its channel moves every Thread device off the current one: only on an explicit request, with that warning.
