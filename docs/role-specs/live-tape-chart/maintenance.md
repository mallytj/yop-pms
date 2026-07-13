# Live Tape Chart — Maintenance

## Problem

Maintenance staff schedule and track repairs across rooms. They need to see which rooms are under maintenance, when blocks start and end, and whether rooms are occupied during work. They do not manage bookings, prices, or guest accounts; the interface should expose only what's needed to plan and complete work.

## Solution

Provide a maintenance view within the tape chart, filtered to rooms with active or scheduled maintenance blocks. Maintenance blocks are visualized as red diagonal-striped overlays on room rows. Staff can create, resize, resolve, and edit blocks directly on the grid. Reservation blocks are visible as contextual read-only overlays — no guest name, no drag — so staff know room occupancy status while scheduling. Room status (out-of-order, dirty) is a manual toggle, not auto-implied by block presence.

## Who

Maintenance staff repair rooms, replace fixtures, handle plumbing/electrical work, and respond to guest-reported issues. They work from a task queue and need to answer: "What needs fixing today?" and "Can I access room 212 now?" They are not reservation agents. They should not need to understand pricing, guest folios, or booking lifecycle.

## Goals

- See all rooms that need maintenance, ordered by urgency or date
- Create and schedule maintenance blocks by clicking on room rows
- Adjust block dates by dragging edges
- Know whether a room is occupied (reservation exists) without seeing guest details
- Mark rooms out-of-order or dirty after work; leave clean to housekeeping

## Tasks

### Daily (every shift)

- Review today's scheduled maintenance blocks
- Mark in-progress blocks as resolved when work is complete
- Create new maintenance blocks for reported issues
- Toggle rooms out-of-order before starting work, back to occupied/vacant after

### Frequent (weekly)

- Schedule upcoming preventive maintenance across multiple rooms
- Edit block reason or dates as scope changes

### Rare

- Delete cancelled maintenance blocks
- Schedule whole-floor maintenance (batch-create blocks across rooms)

## Data They Need

### To plan the day

- List of rooms with active maintenance blocks (today)
- Block reason and status per room
- Whether each room has an active reservation (occupied/vacant) — no guest name

### To create a block

- Room identifier (number, type)
- Date range picker or click-to-select on grid
- Free-text reason field

### To resolve work

- Block status (scheduled → in_progress → resolved)
- Room status toggle (dirty after work, out-of-order during work)

## Pain Points

- Can't tell if a room is occupied without asking reception; shows up to work and room has guest
- Paper/whiteboard-based maintenance log is out of sync with actual room state
- No way to schedule maintenance without walking to each room to check availability

## Constraints

- Must not see guest names (PoLP — maintenance doesn't need guest identity)
- Must not modify, drag, or cancel reservations
- Cannot mark rooms clean (housekeeping must verify and clear)
- Each maintenance block covers exactly one room; multi-room coverage uses separate blocks
- Out-of-order status is manual, not auto-implied by block presence (minor maintenance doesn't make a room unusable)

## Requirements (Draft)

- R-MAINT-001: When viewing the tape chart, show only rooms that have at least one active or scheduled maintenance block
- R-MAINT-002: Display reservation blocks as read-only overlays on maintenance-filtered rooms, showing occupancy status but no guest name
- R-MAINT-003: Show maintenance blocks as transparent red overlays with repeating diagonal stripes, distinct from reservation blocks
- R-MAINT-004: Allow creating a maintenance block by clicking empty space on a room row, entering reason and date range
- R-MAINT-005: Allow resizing a maintenance block by dragging its edges to adjust start/end dates
- R-MAINT-006: Provide one-click resolution of maintenance blocks (status: resolved)
- R-MAINT-007: Allow clicking a maintenance block to edit its reason text inline
- R-MAINT-008: Support deleting a maintenance block
- R-MAINT-009: Allow batch-creating maintenance blocks across multiple rooms via multi-select
- R-MAINT-010: Provide manual toggle to mark rooms out-of-order (not auto-implied by block)
- R-MAINT-011: Allow maintenance staff to mark rooms dirty after completing work
- R-MAINT-012: Prevent maintenance staff from marking rooms clean (housekeeping must verify)
- R-MAINT-013: Show maintenance block status (scheduled, in_progress, resolved) on the block overlay

## Next step

Feed into Live Tape Chart Feature Job Spec → to-tickets → Linear issues.
