---
name: ha-assist
version: "0.0.0"
description: "Home Assistant's own voice assistant, Assist: ask it a sentence, why it cannot control a device (exposure, aliases), change its language, voice, speech engines or the AI agent behind it, voice satellites and pipelines. 语音助手、唤醒词、换声音. Not for controlling devices directly."
compatibility: Requires hass-cli on PATH
metadata:
  requires:
    bins: ["hass-cli"]
---

# ha-assist

> **Shared front door:** read [`../ha-shared/SKILL.md`](../ha-shared/SKILL.md) first; it holds the two-command control and the rules every request follows.

Assist is the assistant built into Home Assistant, separate from the agent reading this. Use this skill when the request is about Assist itself. A request to turn something on is the [front door's control](../ha-shared/SKILL.md#control-a-device), even when it arrives by voice; it is faster and says what changed. Making a speaker say a sentence is [`ha-notify`](../ha-notify/SKILL.md).

## Fast paths

| The user wants | Read | Ends at |
|---|---|---|
| Ask Assist something, or test how it understands a sentence | this file | `raw ws conversation/process` |
| Assist or a voice satellite "doesn't know" a device | this file | expose the entity, or give it an alias |
| A different language, voice, speech engine or AI agent for Assist | this file | `assist pipeline update` |
| Which assistant a voice satellite uses | this file | `assist devices`, then the satellite's select entity |

## The model

- A **pipeline** is one assistant: a `language`, a conversation agent (`conversation_engine`), speech-to-text (`stt_engine`), text-to-speech (`tts_engine`, `tts_voice`) and optionally a wake word. One pipeline is preferred; voice satellites set to "preferred" follow it.
- A **conversation agent** turns text into actions. `conversation.home_assistant` is the built-in one, which matches sentences against entity names and aliases. LLM agents (OpenAI, Google, Ollama and others) are `conversation.*` entities too: `hass-cli state list --domain conversation`.
- Speech engines are entities, `stt.*` and `tts.*`; older integrations are plain engine ids such as `cloud`. A pipeline field takes either form.
- Assist only sees **exposed** entities. Exposure is per assistant, and new entities are not always exposed by default.

## Ask Assist

```bash
hass-cli raw ws conversation/process --data '{"text":"打开客厅灯","language":"zh-cn"}' -o json
hass-cli raw ws conversation/process --data '{"text":"what is the temperature outside","agent_id":"conversation.openai"}' -o json
```

This runs the sentence for real: a command in it changes devices. `response.speech.plain.speech` is what Assist would say; `response.response_type` is `action_done`, `query_answer` or `error`; `response.data.success` lists the entities it acted on. Pass the returned `conversation_id` back to continue a conversation. `service call conversation.process --data '{"text":"..."}' --response` does the same and prints the reply as YAML.

## When Assist cannot control a device

1. Check exposure, then expose what the user named:

   ```bash
   hass-cli raw ws homeassistant/expose_entity/list -o json
   hass-cli raw ws homeassistant/expose_entity --data '{"assistants":["conversation"],"entity_ids":["light.kitchen"],"should_expose":true}'
   ```

2. When the sentence uses a name the entity does not have ("大灯" for `light.living_room_main`), add an alias; aliases are a list, and the update replaces it whole, so include the existing ones from `registry entity list`:

   ```bash
   hass-cli registry entity update light.living_room_main --data '{"aliases":["大灯","客厅主灯"]}'
   ```

3. Test with `conversation/process` and the same sentence. An `error` with `no_intent_match` means the built-in agent has no sentence like it, not a missing device; rephrase, or use an LLM agent.

## Pipelines

```bash
hass-cli assist pipeline list -o json            # all pipelines and preferred_pipeline
hass-cli assist pipeline get --id <id> -o json   # omit --id for the preferred one
hass-cli raw ws tts/engine/voices --data '{"engine_id":"tts.piper","language":"zh-cn"}' -o json
hass-cli assist pipeline create --data '{"name":"中文助手","language":"zh-cn","conversation_engine":"conversation.home_assistant","conversation_language":"zh-cn","stt_engine":null,"stt_language":null,"tts_engine":null,"tts_language":null,"tts_voice":null,"wake_word_entity":null,"wake_word_id":null}'
```

- Language codes are Home Assistant's, as `assist languages` prints them (`zh-cn`, `en`).
- Create takes every field; `null` leaves a stage out (a text-only assistant has no STT or TTS). Update takes the same full set: start from `pipeline get`, drop `id`, change what was asked.
- Engines for a language: `hass-cli raw ws stt/engine/list --data '{"language":"zh-cn"}'`, the same with `tts/engine/list`. A voice must come from the engine's `tts/engine/voices` list.
- A voice satellite (ESPHome, Voice PE, Wyoming) picks its pipeline through a `select.*` entity on the device, usually named Assistant; `assist devices` lists them. Changing it is `select.select_option` on that entity.

## Verb index

| Verb | For | Read when triggered |
|---|---|---|
| `assist pipeline list`, `get [--id]` | the assistants and which is preferred | before any change, for the ids and the current fields |
| `assist pipeline create --data`, `update <id> --data` | a new or changed assistant | [pipelines](#pipelines): every field, `null` for a stage left out |
| `assist pipeline set-preferred <id>` | the default for "preferred" satellites and the app | it changes what every such device uses |
| `assist pipeline delete <id>` | remove an assistant | the preferred one cannot be deleted; set another first |
| `assist languages`, `assist devices` | languages some pipeline can serve; satellites and their pipelines | choosing a language; [pipelines](#pipelines) for a satellite |
| `raw ws conversation/process`, `conversation/agent/list`, `stt/engine/list`, `tts/engine/list`, `tts/engine/voices`, `homeassistant/expose_entity[/list]` | everything without a typed verb | the sections above; [raw access](../ha-shared/references/ha-shared-raw.md) for how `raw ws` sends |

## Safety and escalation

- `conversation/process` acts on what it is told. Do not send a sentence that unlocks, opens or disarms something to "test" Assist; test with a harmless one, and say what it changed.
- Exposing an entity lets anyone who can talk to the satellite control it. Expose what the user named; for locks, alarms and garage doors, say so and confirm first.
- `set-preferred` and `pipeline update` change every satellite and phone that uses that assistant. Say which pipeline changed and what it now uses.
- Delete a pipeline only on the user's explicit request, and say which satellites used it.
