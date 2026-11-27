---
name: ha-media
version: "0.0.0"
description: "Speakers, TVs and receivers: play, pause, next song, louder, quieter, mute, volume to 30%, switch input or app, play a song, playlist or radio station, group speakers together. 音箱、电视、音量、暂停、下一首、播放."
compatibility: Requires hass-cli on PATH
metadata:
  requires:
    bins: ["hass-cli"]
---

# ha-media

> **Shared front door:** read [`../ha-shared/SKILL.md`](../ha-shared/SKILL.md) first; it holds the two-command control and the rules every request follows.

Media players (`media_player.*`): speakers, TVs, receivers, casting devices. Saying a sentence aloud on a speaker ("announce dinner is ready") is [`ha-notify`](../ha-notify/SKILL.md). A TV whose power is only a smart plug is a `switch.*`.

## Fast paths

| The user wants | Read | Ends at |
|---|---|---|
| Play, pause, stop, next, previous | this file | `media_player.media_play_pause`, `media_next_track`, ... |
| Volume to a level: "30%", "音量调到一半" | this file | `media_player.volume_set` with `volume_level` 0 to 1 |
| Louder, quieter, mute | this file | `volume_up` / `volume_down` or `volume_set`, `volume_mute` |
| Switch input or app: "HDMI 2", "Netflix" | this file | `media_player.select_source` with a value from `source_list` |
| Play a song, artist, playlist or station | this file | `media_player.play_media`, or the integration's own search |
| Play on several speakers together | this file | `media_player.join` |
| What is playing | this file | `state get`; `media_title`, `media_artist` |

## The media player model

State is `off`, `on`, `idle`, `playing`, `paused`, `standby` or `buffering`. What it can do now is in its attributes (`hass-cli state get media_player.tv -o json`):

| Attribute | Means |
|---|---|
| `volume_level`, `is_volume_muted` | 0.0 to 1.0, and mute |
| `source`, `source_list` | the current input or app, and the only values `select_source` takes |
| `sound_mode`, `sound_mode_list` | the same for `select_sound_mode` |
| `media_title`, `media_artist`, `app_name` | what is playing |
| `group_members` | the speakers it is grouped with, on platforms that group |

- Most attributes disappear while the player is `off`. Turn it on first (`media_player.turn_on`) when the user asked for a source or volume on a TV that is off, then read the attributes again.
- Percentages are fractions: "30%" is `volume_level: 0.3`. "A bit louder" is `volume_up` (one device step) or `volume_level` plus 0.1; prefer the second when the device's step is unknown and the user wants a noticeable change.
- Pick a source by matching the user's word to `source_list` as written ("HDMI 2", "Netflix"). No match: list the sources and ask.
- `media_play_pause` toggles. Use `media_pause` for "pause" and `media_play` for "resume" so a repeated request does not flip it back.

## Playing something

```bash
hass-cli service call media_player.play_media --data '{"entity_id":"media_player.kitchen","media_content_type":"music","media_content_id":"https://example.com/stream.mp3"}'
```

- `media_content_id` is what the player's integration understands: a URL, or an id from its own library. A song title alone is not an id.
- To find an id, browse the player's library:

  ```bash
  hass-cli service call media_player.browse_media --entity media_player.kitchen
  hass-cli service call media_player.browse_media --entity media_player.kitchen --data '{"media_content_id":"<child id>","media_content_type":"<child type>"}'
  ```

  Each child has a `media_content_id`, a `media_content_type`, and `can_play` / `can_expand`.
- Some integrations search by name with their own service (Music Assistant's `music_assistant.play_media` takes an artist, album or track name). `hass-cli service list` shows which this instance has; `service describe <domain.service>` gives the fields. Prefer it over browsing when it exists.
- `play_media`'s fields vary between Home Assistant versions; on an `HTTP 400`, `service describe media_player.play_media`.

## Groups

`media_player.join` with `entity_id` the leader and `group_members` the others plays the leader's audio on all of them; `media_player.unjoin` on one member takes it out. Only players of the same platform group (Sonos with Sonos, Cast with Cast). When they cannot, say so instead of starting the same stream on each.

## Verb index

| Service | For | Read when triggered |
|---|---|---|
| `media_play`, `media_pause`, `media_play_pause`, `media_stop`, `media_next_track`, `media_previous_track` | transport | [the media player model](#the-media-player-model) for pause versus toggle |
| `volume_set`, `volume_up`, `volume_down`, `volume_mute` | volume; `volume_mute` takes `is_volume_muted` | 0 to 1, not 0 to 100 |
| `select_source`, `select_sound_mode` | input or app, sound mode | the value must be in the list attribute |
| `turn_on`, `turn_off`, `toggle` | power | some TVs cannot be turned on over the network: say so |
| `play_media`, `browse_media` | something specific; `browse_media` prints the library | [playing something](#playing-something) |
| `join`, `unjoin` | speaker groups | [groups](#groups) |
| `media_seek`, `shuffle_set`, `repeat_set`, `clear_playlist` | the rest of the player | `service describe media_player.<service>` for the fields |

## Safety and escalation

- A volume the user did not give is a guess: keep jumps to about 0.1 to 0.2, and above 0.7 only when they asked for it. At night or on an unknown speaker, start low.
- "Pause the music" with several players playing and no room named: pause the one in the room the user is in if known, otherwise ask; do not stop every player in the house.
- Turning off a TV someone else may be watching, or taking a speaker out of a group, is a change the user should hear about in the reply.
- Do not start playback the user did not ask for to test a player.
