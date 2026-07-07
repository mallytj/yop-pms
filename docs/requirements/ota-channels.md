# OTA Channels RTM

> **⚠️ PLACEHOLDER — DEFERRED**
>
> OTA channel integration (Booking.com, Expedia, Airbnb) is **out of scope** for the
> current reservation-flow work. This file exists only to acknowledge that OTA
> support is a planned feature so other docs can reference it.
>
> Do **not** implement against this doc. The full webhook contract, payload mappings,
> idempotency rules, channel-specific auth, room/rate mapping schema, currency
> conversion, modify/cancel semantics, outbound inventory sync, and dead-letter
> handling will all be designed in the OTA PR.
>
> When the OTA sprint starts, this file will be replaced with a full RTM.

## Known requirements (placeholders)

- Inbound webhook surface per channel.
- Signature verification (likely HMAC-SHA256, secret per channel, rotatable).
- Idempotent inbound processing keyed on a channel-supplied message id.
- Canonical mapping from each channel's reservation payload to internal
  `POST /api/v1/reservations` shape with `source='ota'` + `channel_id` metadata.
- Modify / cancel event handling.
- Outbound rates + availability sync (deferred further — possibly own PR).

## Schema deferred to OTA PR

The reservation-API sprint commits the inbound action enum (`ota_action`)
and the dead-letter pattern but defers the schema needed to **route** a
cancel / modify back to the local reservation. Pinned here so the
reservations spec doesn't carry OTA-specific columns prematurely.

- `operations.reservations.channel_reservation_id TEXT` (nullable; non-OTA
  rows leave it NULL). Unique per `(channel_id, channel_reservation_id)`.
  Used by R-RES-OTA-006 to look up cancel targets.
- Index on `(channel_id, channel_reservation_id) WHERE channel_reservation_id IS NOT NULL`.
- Migration to add the column lands in the OTA PR, not the reservations
  sprint. v1 cancel-action rows accumulate in
  `operations.ota_inbound_messages` and dead-letter until this column +
  lookup logic exist.

## Action enum reference

`operations.ota_action ENUM ('create','modify','cancel')` is created in
reservations migration M5 (see `reservations.md` §13). v1 routes `create`;
`cancel` rows dead-letter until the schema above lands; `modify` always
dead-letters.
