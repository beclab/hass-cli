---
name: ha-assist
version: 0.1.0
description: "Use Home Assistant's Assist voice/conversation layer with hass-cli: send a natural-language command to the conversation agent, list conversation agents and assist pipelines, list STT/TTS engines and voices, and speak text on a media player. Use for 'tell Assist to turn off the lights', 'what voice assistants are configured', 'run a sentence through HA' requests."
metadata:
  requires:
    bins: ["hass-cli"]
  cliHelp: "hass-cli assist --help"
---

# ha-assist

Conversation, pipelines, STT/TTS. **Prerequisite:**
[`../ha-shared/SKILL.md`](../ha-shared/SKILL.md).

## Send a natural-language command

The conversation agent maps a sentence to actions — a quick way to control HA
by intent instead of crafting a `service call`:

```bash
hass-cli raw ws conversation/process --data '{"text":"turn off the kitchen lights"}' -o json
# Optional: "language":"en", "agent_id":"<agent>", "conversation_id":"..."
```

Also available as REST: `hass-cli raw api POST conversation/process --data '{"text":"..."}'`.

## Manage Assist pipelines

A pipeline chains conversation + STT + TTS (+ wake word). Manage them directly:

```bash
hass-cli assist pipeline list                 # all pipelines + preferred_pipeline
hass-cli assist pipeline get                   # the preferred one (--id for a specific)
hass-cli assist pipeline create --data '{"name":"My Assist","language":"en","conversation_engine":"conversation.home_assistant","conversation_language":"en","stt_engine":null,"stt_language":null,"tts_engine":null,"tts_language":null,"tts_voice":null,"wake_word_entity":null,"wake_word_id":null}'
hass-cli assist pipeline update <id> --data '{"name":"Renamed"}'
hass-cli assist pipeline set-preferred <id>
hass-cli assist pipeline delete <id>

hass-cli assist languages                      # languages supported by available pipelines
hass-cli assist devices                        # devices bound to a pipeline
```

`conversation_engine` is required on create; use `conversation.home_assistant`
for the built-in agent. STT/TTS/wake fields can be null for a text-only pipeline.

Conversation agents (not pipelines):

```bash
hass-cli raw ws conversation/agent/list -o json
```

## STT / TTS engines

```bash
hass-cli raw ws stt/engine/list -o json
hass-cli raw ws tts/engine/list -o json
hass-cli raw ws tts/engine/voices --data '{"engine_id":"tts.google_translate","language":"en"}' -o json
```

## Speak text on a speaker

```bash
hass-cli service call tts.speak --data '{"entity_id":"tts.google_translate","media_player_entity_id":"media_player.kitchen","message":"Hello"}'
```

## Notes

- `conversation/process` is the highest-leverage entry point: one sentence, HA
  figures out the entities. Good for ambiguous requests.
- Full speech (audio in/out) runs over `assist_pipeline/run`, which is a binary
  streaming flow not suited to a one-shot CLI; use `conversation/process` for
  text, and `tts.speak` for output.
