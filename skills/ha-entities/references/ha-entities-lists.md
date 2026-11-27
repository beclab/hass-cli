# To-do lists and calendars

Read when the user wants to read or change a shopping or to-do list, or read or add a calendar event. Back to [`ha-entities`](../SKILL.md).

## To-do lists

A `todo.*` entity is a list; its state is the number of items not yet done. The shopping list is usually `todo.shopping_list`.

```bash
hass-cli state list --domain todo
hass-cli service call todo.get_items --entity todo.shopping_list --data '{"status":"needs_action"}'
hass-cli service call todo.add_item --data '{"entity_id":"todo.shopping_list","item":"牛奶"}'
hass-cli service call todo.update_item --data '{"entity_id":"todo.shopping_list","item":"牛奶","status":"completed"}'
hass-cli service call todo.remove_item --data '{"entity_id":"todo.shopping_list","item":"牛奶"}'
```

- `todo.get_items` prints the items under `<entity_id>.items`. Leave out `status` for every item, or give `completed`.
- `item` in `update_item` and `remove_item` is the item's current summary (or its `uid` from `get_items`). Read the list first when the user's words may not match the text exactly ("milk" against "Milk 2L").
- "Tick off" is `update_item` with `status: completed`; "rename" is `update_item` with `rename`. `add_item` can also take `due_date` or `due_datetime` and `description` on lists that support them.
- `todo.remove_completed_items` clears every ticked item at once.

## Calendars

A `calendar.*` entity's state is `on` while an event is happening; its attributes show the next event only.

```bash
hass-cli state list --domain calendar
hass-cli service call calendar.get_events --entity calendar.family --data '{"start_date_time":"2026-06-01 00:00:00","end_date_time":"2026-06-08 00:00:00"}'
hass-cli service call calendar.create_event --data '{"entity_id":"calendar.family","summary":"看牙医","start_date_time":"2026-06-25 09:00:00","end_date_time":"2026-06-25 10:00:00"}'
```

- `get_events` takes `start_date_time` and either `end_date_time` or `duration`; it prints the events under `<entity_id>.events`.
- An all-day event uses `start_date` and `end_date` (`end_date` is the day after the last day). A timed one uses `start_date_time` and `end_date_time`, in Home Assistant's time zone unless an offset is given.
- `create_event` also takes `description` and `location`. Not every calendar can be written (a subscribed or holiday calendar cannot); the call fails on those.
- "Tomorrow", "next Monday": resolve the date from today's date in the house's time zone and say the date back.
- Changing or deleting an event is not a service; it is `raw ws calendar/event/update` or `calendar/event/delete`, which need the event's `uid` from `get_events`. Only on the user's explicit request, and read the event back first.
