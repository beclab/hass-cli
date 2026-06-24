---
name: ha-media
version: 0.1.0
description: "Control Home Assistant media players with hass-cli: play/pause/stop, next/previous track, set volume, mute, select source, and play media. Use for 'pause the TV', 'set volume to 30%', 'play music in the kitchen', 'switch input to HDMI2' requests. All control goes through service call."
metadata:
  requires:
    bins: ["hass-cli"]
  cliHelp: "hass-cli service call --help"
---

# ha-media

Media players (speakers, TVs, receivers). **Prerequisites:**
[`../ha-shared/SKILL.md`](../ha-shared/SKILL.md), [`../ha-services/SKILL.md`](../ha-services/SKILL.md).

## Transport

```bash
hass-cli service call media_player.media_play         --arguments entity_id=media_player.kitchen
hass-cli service call media_player.media_pause        --arguments entity_id=media_player.kitchen
hass-cli service call media_player.media_play_pause   --arguments entity_id=media_player.kitchen
hass-cli service call media_player.media_stop         --arguments entity_id=media_player.kitchen
hass-cli service call media_player.media_next_track   --arguments entity_id=media_player.kitchen
hass-cli service call media_player.media_previous_track --arguments entity_id=media_player.kitchen
```

## Volume / mute / power

```bash
# volume_level is 0.0 .. 1.0
hass-cli service call media_player.volume_set  --data '{"entity_id":"media_player.kitchen","volume_level":0.3}'
hass-cli service call media_player.volume_mute --data '{"entity_id":"media_player.kitchen","is_volume_muted":true}'
hass-cli service call media_player.turn_off    --arguments entity_id=media_player.kitchen
```

## Source / sound mode

```bash
hass-cli state get media_player.kitchen -o json   # attributes.source_list / sound_mode_list
hass-cli service call media_player.select_source --data '{"entity_id":"media_player.tv","source":"HDMI 2"}'
```

## Play media

```bash
hass-cli service call media_player.play_media --data '{"entity_id":"media_player.kitchen","media_content_type":"music","media_content_id":"http://.../stream.mp3"}'
```

Browse available content with the WS API when needed:
`hass-cli raw ws media_player/browse_media --data '{"entity_id":"media_player.kitchen"}'`.
