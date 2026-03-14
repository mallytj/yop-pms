# Database Conventions

## Timestamps
- Always use `TIMESTAMPTZ` instead of `TIMESTAMP`
- Always use `NOW()` instead of `CURRENT_TIMESTAMP`
- Always use `NOW()` instead of `CURRENT_TIMESTAMP`

## Booleans
- Always use `BOOLEAN` instead of `BIT` or `TINYINT`
- Always have a default value for boolean columns
- Always have a `NOT NULL` constraint for boolean columns

## Primary Keys
- Always use `UUIDv7` instead of `SERIAL`
- Always have a `NOT NULL` constraint for primary keys
- Always have a `UNIQUE` constraint for primary keys
- Always have a `CHECK` constraint for primary keys

## Foreign Keys
- Always have a `NOT NULL` constraint for foreign keys
- Always have a `RESTRICT` constraint for foreign keys

## Case Sensitivity
- Always use `CITEXT` for case-insensitive columns (e.g. emails, codes etc)

## Row Level Security
- Always `ENABLE` and `FORCE` Row Level Security (RLS) on all tables with tenant-isolated data
- Always create a `RLS` policy for each table with tenant-isolated data

## Indexes
- Always create an index for each foreign key

