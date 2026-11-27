---
name: ha-notify
version: "0.0.0"
description: "Send a message or alert to a phone, announce or say something out loud on a speaker or voice assistant (text to speech), post or dismiss a notification on the Home Assistant dashboard. 发通知、推送到手机、在音箱上播报."
compatibility: Requires hass-cli on PATH
metadata:
  requires:
    bins: ["hass-cli"]
---

# ha-notify

> **Shared front door:** read [`../ha-shared/SKILL.md`](../ha-shared/SKILL.md) first; it holds the two-command control and the rules every request follows.

Three ways to tell someone something: a push message to a phone or another service, speech on a speaker, and a notification on the Home Assistant dashboard. Playing music is [`ha-media`](../ha-media/SKILL.md). A message that should go out when something happens ("tell me when the washer finishes") is an automation: [the builder](../ha-workflow-automation-builder/SKILL.md).

## Fast paths

| The user wants | Read | Ends at |
|---|---|---|
| "Send my phone a message", "notify Alice" | this file | `notify.mobile_app_<device>` with `message` |
| Say something out loud on a speaker: "announce dinner is ready", "播报" | this file | `tts.speak`, or `assist_satellite.announce` on a voice device |
| A reminder on the dashboard | this file | `persistent_notification.create` |
| Clear a dashboard notification | this file | `persistent_notification.dismiss` |

## Where a message can go

Home Assistant has two kinds of notify target, and an instance usually has both:

```bash
hass-cli service list notify          # services: notify.mobile_app_alice_phone, notify.notify, ...
hass-cli state list --domain notify   # entities: notify.<name>, sent with notify.send_message
```

- **A service per target**, `notify.<target>`. The companion app registers `notify.mobile_app_<device>`, one per phone; that is the usual "send to my phone". `notify.notify` goes to a default target the integration chose, which may not be the user's phone: prefer the named one.
- **A notify entity**, sent with `notify.send_message` and its `entity_id`. Newer integrations use these.
- Pick the target from the user's words ("Alice's phone", "我的手机") against the service or entity names. Several phones and no person named: ask.

```bash
hass-cli service call notify.mobile_app_alice_phone --data '{"title":"Home","message":"The garage is still open"}'
hass-cli service call notify.send_message --data '{"entity_id":"notify.office_display","message":"Meeting in 5 minutes"}'
```

`data` adds what the platform supports (actions, priority, an image, a tag that replaces an earlier message). It differs between iOS, Android and other services; `service describe notify.<target>` lists what this one takes, and the companion app's documentation the rest. Send the plain message unless the user asked for more.

## Speaking out loud

Two services, depending on the device:

```bash
hass-cli state list --domain tts                  # the speech engines: tts.google_translate_en_com, tts.piper, ...
hass-cli service call tts.speak --data '{"entity_id":"tts.piper","media_player_entity_id":"media_player.kitchen","message":"Dinner is ready"}'

hass-cli state list --domain assist_satellite     # voice devices with their own speaker
hass-cli service call assist_satellite.announce --data '{"entity_id":"assist_satellite.kitchen","message":"Dinner is ready"}'
```

- `tts.speak`: `entity_id` is the **speech engine** (`tts.*`), `media_player_entity_id` the speaker. Mixing them up is the usual `HTTP 400`. Add `language` when the message is not in the engine's default language; a Chinese sentence on an English-only engine comes out wrong or silent.
- `assist_satellite.announce` is for voice-assistant devices; it uses the device's own pipeline voice.
- Speaking on a speaker interrupts what it was playing. Most players do not resume by themselves.
- Older setups have a legacy `tts.<engine>_say` service instead of the entity; it takes the speaker as `entity_id`.

## Dashboard notifications

```bash
hass-cli service call persistent_notification.create --data '{"title":"Reminder","message":"Change the filter","notification_id":"filter"}'
hass-cli service call persistent_notification.dismiss --data '{"notification_id":"filter"}'
hass-cli raw ws persistent_notification/get    # the ones showing now
```

A `notification_id` lets a later call replace or dismiss it; without one it can only be dismissed from the UI or by `dismiss_all`. These show only in the Home Assistant UI, not on a phone.

## Verb index

| Service | For | Read when triggered |
|---|---|---|
| `notify.<target>` | a push or message through that integration; `message`, `title`, `data` | [where a message can go](#where-a-message-can-go) |
| `notify.send_message` | a notify entity; `entity_id`, `message`, `title` | same |
| `tts.speak` | speech on a media player | [speaking out loud](#speaking-out-loud) for which id goes where |
| `assist_satellite.announce` | speech on a voice-assistant device | `service describe assist_satellite.announce` for chimes and media |
| `persistent_notification.create`, `dismiss`, `dismiss_all` | the dashboard's notification list | [dashboard notifications](#dashboard-notifications) |

## Safety and escalation

- Send to the person the user named. A message meant for one person sent to `notify.notify` or to every phone reaches people it was not for.
- Say back where it went ("sent to Alice's phone") so the user can tell a wrong target.
- Speech is heard by everyone in the room: do not read out codes, passwords or private messages on a speaker unless the user asked for exactly that.
- One request, one message. Do not retry a send that returned without error because the user has not seen it yet; a phone that is offline gets it later.
- `dismiss_all` clears notifications other integrations raised (repairs, new devices); only on the user's explicit request.
