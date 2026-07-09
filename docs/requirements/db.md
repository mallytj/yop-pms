| ID      | Requirement Description                                                                                                                        |
| ------- | ---------------------------------------------------------------------------------------------------------------------------------------------- |
| REQ-001 | Database must support SQL-based migrations via Goose                                                                                           |
| REQ-002 | All Primary Keys must use UUIDv7 for time-sortable uniqueness                                                                                  |
| REQ-003 | Monetary values must be stored as integers (pence)                                                                                             |
| REQ-004 | Audit fields (created_at, updated_at, deleted_at) on all core tables                                                                           |
| REQ-005 | Any indexes created on a table dependent on property must include the property_id                                                              |
| REQ-006 | All foreign keys must have ON DELETE and ON UPDATE actions defined                                                                             |
| REQ-007 | All text/varchar columns must have CHECK constraints enforcing length. Exception: columns with a strict regex constraint (e.g. `^[a-z0-9_]{1,20}$`) that already constrains input length do not need a separate length CHECK. |
| REQ-008 | All tables must have a schema defined                                                                                                          |
| REQ-009 | All timestamps must be stored as TIMESTAMPTZ (UTC).                                                                                            |
| REQ-010 | High-concurrency tables must support Optimistic Locking (via a version column).                                                                |
| REQ-011 | All Foreign Key columns must have an explicit Index.                                                                                           |
| REQ-012 | Constraint names must follow a strict convention ({table}_{column}_{suffix}).                                                                  |
| REQ-013 | All foreign keys must RESTRICT NOT CASCADE to preserve historical data                                                                         |
| REQ-014 | If a reference depends on property, there must be a FK created references the property and entity                                              |
| REQ-015 | Any boolean columns must have a set default                                                                                                    |
| REQ-016 | Partial Uniqueness: Unique constraints on soft-deletable tables MUST include WHERE (deleted_at IS NULL) to allow record re-creation.           |
| REQ-017 | Case-Insensitivity: Use CITEXT for emails and usernames to prevent duplicate account creation via casing variance.                             |
| REQ-018 | Logic Separation: No business logic (price calcs, state logic) in Triggers/Procedures; DB is for integrity, Go is for logic.                   |
| REQ-019 | PII Isolation: Personally Identifiable Information (PII) must be documented and logically separated to support "Right to be Forgotten" (GDPR). |
| REQ-020 | RLS Mandatory: Every table containing property_id MUST have Row-Level Security enabled and forced.                                             |
| REQ-021 | Least Privilege: The Go application must connect via a non-superuser role that cannot bypass RLS policies.                                     |
| REQ-022 | Context Propagation: All Go transactions must set app.current_property_id before executing queries.                                            |
| REQ-023 | Functional Auth: Functional RBAC (Roles) must be handled in the App Layer; RLS is strictly for Tenant (Property) isolation.                    |
| REQ-024 | Date Exclusion: Overlapping bookings for the same resource MUST be prevented at the DB level using EXCLUDE constraints (GiST indexes).         |
| REQ-025 | Standardized Naming: All ID columns in foreign tables must follow the {table}_id convention for join clarity.                                  |
| REQ-026 | Initialize extensions (uuid-ossp, gist)                                                                                                        |
| REQ-027 | Create empty schemas (operations, relations, identity, auth, etc.)                                                                             |
| REQ-028 | Each string type where the values are preset MUST be an enum                                                                                   |
| REQ-029 | System must store Staff/Admin user credentials and logs                                                                                        |
| REQ-030 | There must be a table for licences                                                                                                             |
| REQ-031 | The licence key must be unique                                                                                                                 |
| REQ-032 | The licence key must be in format YOP-XXXX where x is an int                                                                                   |
| REQ-033 | The organisation name is required                                                                                                              |
| REQ-034 | The organisation name must not exceed 50 characters                                                                                            |
| REQ-035 | The contact email is required                                                                                                                  |
| REQ-036 | The contact email must be valid                                                                                                                |
| REQ-037 | The licence notes must not exceed 1500 characters                                                                                              |
| REQ-038 | Licence must be inactive if soft deleted                                                                                                       |
| REQ-039 | There must be a table for properties                                                                                                           |
| REQ-040 | The licence must exist & be active                                                                                                             |
| REQ-041 | The name is required                                                                                                                           |
| REQ-042 | The name must not exceed 50 characters                                                                                                         |
| REQ-043 | The address is required                                                                                                                        |
| REQ-044 | The address must not exceed 250 characters                                                                                                     |
| REQ-045 | The timezone is required                                                                                                                       |
| REQ-046 | The timezone must be in IANA format                                                                                                            |
| REQ-047 | The property notes must not exceed 1500 characters                                                                                             |
| REQ-048 | There must be only one property per address per licence                                                                                        |
| REQ-049 | There must be an index on properties for licences                                                                                              |
| REQ-050 | There must be a function for checking if a licence exists on a property                                                                        |
| REQ-051 | There must be a index for a properties name by licence                                                                                         |
| REQ-052 | There must be an index for if a property is active by licence                                                                                  |
| REQ-053 | There must be a table for users                                                                                                                |
| REQ-054 | The licence must exist & be active                                                                                                             |
| REQ-055 | The username is required                                                                                                                       |
| REQ-056 | The username must be unique                                                                                                                    |
| REQ-057 | The username must be alphanumerical or \_                                                                                                      |
| REQ-058 | The username must not exceed 20 characters                                                                                                     |
| REQ-059 | The email is required                                                                                                                          |
| REQ-060 | The password_hash is required                                                                                                                  |
| REQ-061 | The role is required                                                                                                                           |
| REQ-062 | The role must be a valid role                                                                                                                  |
| REQ-063 | The first name is required                                                                                                                     |
| REQ-064 | The first name must not exceed 50 characters                                                                                                   |
| REQ-065 | The first name must not have any special characters or numbers (other than -)                                                                  |
| REQ-066 | The last name is required                                                                                                                      |
| REQ-067 | The last name must not exceed 50 characters                                                                                                    |
| REQ-068 | The last name must not have any special characters or numbers (other than -)                                                                   |
| REQ-069 | A user's email address must be unique                                                                                                          |
| REQ-070 | A user's username must be unique                                                                                                               |
| REQ-071 | A user's password must not be stored in plain text                                                                                             |
| REQ-072 | A user's email must be valid                                                                                                                   |
| REQ-073 | There must be an enum for a user's roles                                                                                                       |
| REQ-074 | There must be an index on users for licences                                                                                                   |
| REQ-075 | There must be an index for users for name (Last, First)                                                                                        |
| REQ-076 | There must be an index for user's role                                                                                                         |
| REQ-077 | There must be an index for if a user is active                                                                                                 |
| REQ-078 | The email must be unique                                                                                                                       |
| REQ-079 | There must be an index for a user's email                                                                                                      |
| REQ-080 | There must be a table for guests                                                                                                               |
| REQ-081 | The guests property must exist                                                                                                                 |
| REQ-082 | The first name is required                                                                                                                     |
| REQ-083 | The first name must not exceed 50 characters                                                                                                   |
| REQ-084 | The first name must not have any special characters or numbers (other than -)                                                                  |
| REQ-085 | The last name is required                                                                                                                      |
| REQ-086 | The last name must not exceed 50 characters                                                                                                    |
| REQ-087 | The last name must not have any special characters or numbers (other than -)                                                                   |
| REQ-088 | The marketing opt in must be defaulted as false for GDPR                                                                                       |
| REQ-089 | Annoymising a guest must hide all personal data                                                                                                |
| REQ-090 | A guest's email must be valid                                                                                                                  |
| REQ-091 | A guest's phone number must be valid                                                                                                           |
| REQ-092 | A guest must have an is_anonymised column for GDPR                                                                                             |
| REQ-093 | There must be a index for a guest's name (Last, First)                                                                                         |
| REQ-094 | There Must be an index for a guests email                                                                                                      |
| REQ-095 | There must be an index for a guests phone number                                                                                               |
| REQ-096 | There must be an index for a guests marketing preference                                                                                       |
| REQ-097 | There must be an index for if a guest is anonymised                                                                                            |
| REQ-098 | There must be an index for a guests property                                                                                                   |
| REQ-099 | There must be a table for audit logs                                                                                                           |
| REQ-100 | The user must exist                                                                                                                            |
| REQ-101 | The entity must exist                                                                                                                          |
| REQ-102 | The action is required                                                                                                                         |
| REQ-103 | The entity is required                                                                                                                         |
| REQ-104 | Change must be in format {field: X, before: Y, after: Z)                                                                                       |
| REQ-105 | There must be an enum for a audit log's entity                                                                                                 |
| REQ-106 | There must be an enum for a audit log's type                                                                                                   |
| REQ-107 | There must be an index for an audit log's user                                                                                                 |
| REQ-108 | There must be an index for an audit log's entity                                                                                               |
| REQ-109 | There must be an index for an audit log's action                                                                                               |
| REQ-110 | The property must exist                                                                                                                        |
| REQ-111 | The entity ID is required                                                                                                                      |
| REQ-112 | There must be a table for amenities                                                                                                            |
| REQ-113 | The amenity's property must exist                                                                                                              |
| REQ-114 | The amenity's name is required                                                                                                                 |
| REQ-115 | The amenity's name must not exceed 100 characters                                                                                              |
| REQ-116 | The amenity's short code must not exceed 5 characters                                                                                          |
| REQ-117 | The amenity's description must not exceed 250 characters                                                                                       |
| REQ-118 | An amenity's short_code must be unique for each property                                                                                       |
| REQ-119 | An amenity's name must be unique for each property                                                                                             |
| REQ-120 | There must be an index for an amenities property                                                                                               |
| REQ-121 | There must be an index for if an amenity is active                                                                                             |
| REQ-122 | The shortcode must be alphanumerical or \ \_ or / or -                                                                                         |
| REQ-123 | There must be a join table for property amenities                                                                                              |
| REQ-124 | The property must exist                                                                                                                        |
| REQ-125 | The amenity must exist                                                                                                                         |
| REQ-126 | There must be an index for the property                                                                                                        |
| REQ-127 | There must be an index for the amenity                                                                                                         |
| REQ-128 | The amenity must be of the same property id as the relation                                                                                    |
| REQ-129 | There must be a table for travel agents                                                                                                        |
| REQ-130 | The travel agents name is required                                                                                                             |
| REQ-131 | A travel agent's email must be valid                                                                                                           |
| REQ-132 | A travel agent's phone number must be valid                                                                                                    |
| REQ-133 | A travel agent's name must not exceed 100 characters                                                                                           |
| REQ-134 | A travel agent's notes must not exceed 1500 characters                                                                                         |
| REQ-135 | A travel agent's name must be unique by property                                                                                               |
| REQ-136 | The commisson percentage must be a positive integer                                                                                            |
| REQ-137 | The commission percentage must not exceed 75%                                                                                                  |
| REQ-138 | There must be an index for a travel agent's property                                                                                           |
| REQ-139 | There must be an identity docs table                                                                                                           |
| REQ-140 | The doc type is required and must be in the enum                                                                                               |
| REQ-141 | The issuing country must be in ISO format                                                                                                      |
| REQ-142 | The doc image url must be valid                                                                                                                |
| REQ-143 | There must be a doc number                                                                                                                     |
| REQ-144 | An identity doc's guest must exist                                                                                                             |
| REQ-145 | There must be an enum for doc type                                                                                                             |
| REQ-146 | The doc number must be encrypted                                                                                                               |
| REQ-147 | There must be an index on the identity doc's guest                                                                                             |
| REQ-148 | The issuing country is required                                                                                                                |
| REQ-149 | There must be a room types table                                                                                                               |
| REQ-150 | The room type's property must exist                                                                                                            |
| REQ-151 | The room type's name is required                                                                                                               |
| REQ-152 | The room type's name must not exceed 75 characters                                                                                             |
| REQ-153 | The room type's code is required                                                                                                               |
| REQ-154 | The room type's code must not exceed 7 characters                                                                                              |
| REQ-155 | The standard occupancy must be a postive integer                                                                                               |
| REQ-156 | The min occupancy must be a positive integer                                                                                                   |
| REQ-157 | The min occupancy must be less than or equal to the standard occupancy                                                                         |
| REQ-158 | The max occupancy must be a positive integer                                                                                                   |
| REQ-159 | The max occupancy must be greater than or equal to the standard occupancy                                                                      |
| REQ-160 | A room type's code must be unique by property                                                                                                  |
| REQ-161 | A room type's name must by unqiue by property                                                                                                  |
| REQ-162 | There must be an index for a room type's property                                                                                              |
| REQ-163 | There must be a join table for a room type's amenities                                                                                         |
| REQ-164 | The room type must exist                                                                                                                       |
| REQ-165 | The amenity must exist                                                                                                                         |
| REQ-166 | There must be an index for the room type                                                                                                       |
| REQ-167 | There must be an index for the amenity                                                                                                         |
| REQ-168 | The amenity must be of the same property id as the room type                                                                                   |
| REQ-169 | There must be a rooms table                                                                                                                    |
| REQ-170 | The rooms name must not exceed 75 characters                                                                                                   |
| REQ-171 | The rooms property must exist                                                                                                                  |
| REQ-172 | The rooms room type must exist                                                                                                                 |
| REQ-173 | There must be an enum for housekeeping status                                                                                                  |
| REQ-174 | There must be an enum for occupancy status                                                                                                     |
| REQ-175 | A room's name must be unique by property                                                                                                       |
| REQ-176 | There must be an index for a rooms room type                                                                                                   |
| REQ-177 | There must be an index for a room's housekeeping status                                                                                        |
| REQ-178 | There must be an index for a room's occupancy status                                                                                           |
| REQ-179 | There must be an index for a room's property                                                                                                   |
| REQ-180 | There must be a join table for a rooms amenities                                                                                               |
| REQ-181 | The room must exist                                                                                                                            |
| REQ-182 | The amenity must exist                                                                                                                         |
| REQ-183 | There must be an index for the room                                                                                                            |
| REQ-184 | There must be an index for the amenity                                                                                                         |
| REQ-185 | The amentiy & room must be owned by the same property                                                                                          |
| REQ-186 | There must be a table for maintenace blocks                                                                                                    |
| REQ-187 | The maintenance block's room must exist                                                                                                        |
| REQ-188 | The maintenace block's creator must exist                                                                                                      |
| REQ-189 | A maintenace blocks block_period must be Start->End not end->start                                                                             |
| REQ-190 | There must be an enum for a maintenace blocks type                                                                                             |
| REQ-191 | A maintenace block's reason must not be longer than 150 characters                                                                             |
| REQ-192 | There must not be multiple maintenace blocks on one room at the same time                                                                      |
| REQ-193 | There must be an index for a maintenace block's room by block_period                                                                           |
| REQ-194 | There must be an index for a maintenace block's block_period                                                                                   |
| REQ-195 | There must be an index for a maintenace block's type                                                                                           |
| REQ-196 | There must be an index for a maintenace block's created by user                                                                                |
| REQ-197 | There must be an index for a maintenace block's room                                                                                           |
| REQ-198 | Room ID is required                                                                                                                            |
| REQ-199 | Block Period is required                                                                                                                       |
| REQ-200 | Reason is required                                                                                                                             |
| REQ-201 | Type is required                                                                                                                               |
| REQ-202 | Created by user id is required                                                                                                                 |
| REQ-203 | There must be a table for rate plans                                                                                                           |
| REQ-204 | The rate plan name is required                                                                                                                 |
| REQ-205 | The rate plan code is required                                                                                                                 |
| REQ-206 | A rate plan's derivation must be in format {'type': 'percentage' \| 'fixed', value: '+-x'}                                                     |
| REQ-207 | A rate plans currency code must be in a ISO 4217 format                                                                                        |
| REQ-208 | A rate plans code must not be longer than 7 characters                                                                                         |
| REQ-209 | A rate plans name must not be longer than 30 characters                                                                                        |
| REQ-210 | A rate plans description must not be longer than 300 characters                                                                                |
| REQ-211 | There must be an index for a rate plans property                                                                                               |
| REQ-212 | There must be an index for a rate plans parent rate plan                                                                                       |
| REQ-213 | There must be an index for if a rate plan is active                                                                                            |
| REQ-214 | If derivation rule is set, there must be a parent rate plan that exists                                                                        |
| REQ-215 | The parent rate plan must belong to the same property                                                                                          |
| REQ-216 | If there is no parent rate plan, there can not be a derivation rule                                                                            |
| REQ-217 | There must be a company profiles table                                                                                                         |
| REQ-218 | The company name must be unique by property                                                                                                    |
| REQ-219 | The tax ID must be in a valid format                                                                                                           |
| REQ-220 | The company profile's property must exist                                                                                                      |
| REQ-221 | The negotiated rate plan ID must exist (if not null)                                                                                           |
| REQ-222 | The company name must not exceed 50 characters                                                                                                 |
| REQ-223 | The company name is required                                                                                                                   |
| REQ-224 | The contact email must be in a valid format                                                                                                    |
| REQ-225 | The contact phone number must be in a valid format                                                                                             |
| REQ-226 | The billing address must be in a valid format                                                                                                  |
| REQ-227 | The billing address must not exceed 300 characters                                                                                             |
| REQ-228 | The company notes must not exceed 1500 characters                                                                                              |
| REQ-229 | There must be an index for a company profile's property                                                                                        |
| REQ-230 | There must be an index for a company profile's negotiated rate plan                                                                            |
| REQ-231 | The rate plan must exist in the same property                                                                                                  |
| REQ-232 | The tax ID must be unique per property                                                                                                         |
| REQ-233 | There must be a daily price grid table                                                                                                         |
| REQ-234 | The room type must exist                                                                                                                       |
| REQ-235 | The rate plan must exist                                                                                                                       |
| REQ-236 | The property must exist                                                                                                                        |
| REQ-237 | The calendar date is required                                                                                                                  |
| REQ-238 | The calendar date must be in ISO-87601 format (YYYY-MM-DD)                                                                                     |
| REQ-239 | On creation, the calendar date nsut be in the future                                                                                           |
| REQ-240 | The base price must be an integer                                                                                                              |
| REQ-241 | The min_los_restrction must be positive                                                                                                        |
| REQ-242 | The max_los_restriction must be positive                                                                                                       |
| REQ-243 | The max_los_restriction must be greater than the min_los_restriction                                                                           |
| REQ-244 | The row must be unique by room type, calendar date and rate plan                                                                               |
| REQ-245 | There must be an index for a daily price's room type                                                                                           |
| REQ-246 | There must be an index for a daily price's rate plan                                                                                           |
| REQ-247 | There must be an index for a daily price's date                                                                                                |
| REQ-248 | There must be an index for a daily price's availability                                                                                        |
| REQ-249 | There must be an index for a daily price's room type, by date when available                                                                   |
| REQ-250 | There must be an index for a daily price's rate plan, by date when available                                                                   |
| REQ-251 | The room type must exist in the same property                                                                                                  |
| REQ-252 | The rate plan must exist in the same property                                                                                                  |
| REQ-253 | There must be a tax rules table                                                                                                                |
| REQ-254 | The tax rule's propery must exist                                                                                                              |
| REQ-255 | The tax rule's name is required                                                                                                                |
| REQ-256 | The tax rule's name must not exceed 50 characters                                                                                              |
| REQ-257 | The tax rules description must not exceed 250 characters                                                                                       |
| REQ-258 | The tax percentage must not exceed 75%                                                                                                         |
| REQ-259 | The tax percentage must be positive                                                                                                            |
| REQ-260 | The tax rule name must be unique by property                                                                                                   |
| REQ-261 | There must be an index for a tax rule's property                                                                                               |
| REQ-262 | There must be a ledger code table                                                                                                              |
| REQ-263 | The ledger code's property must exist                                                                                                          |
| REQ-264 | The code is required                                                                                                                           |
| REQ-265 | The code must be unique to the property                                                                                                        |
| REQ-266 | The code must not exceed 50 characters                                                                                                         |
| REQ-267 | The description must not exceed 250 characters                                                                                                 |
| REQ-268 | The ledger code's tax rule must exist if set                                                                                                   |
| REQ-269 | There must be an index for a ledger code's property                                                                                            |
| REQ-270 | There must be an index for a ledger code's tax rule                                                                                            |
| REQ-271 | There must be a accounts table                                                                                                                 |
| REQ-272 | The account's property must exist                                                                                                              |
| REQ-273 | The account's company profile must exist if set                                                                                                |
| REQ-274 | The credit limit must be a positive integer                                                                                                    |
| REQ-275 | The payment terms must be a positive integer                                                                                                   |
| REQ-276 | Each company profile should only have one account per property                                                                                 |
| REQ-277 | There must be an index for account's property                                                                                                  |
| REQ-278 | There must be an index for the account's company profile                                                                                       |
| REQ-279 | The name is required                                                                                                                           |
| REQ-280 | The name must not exceed 100 characters                                                                                                        |
| REQ-281 | The code is required                                                                                                                           |
| REQ-282 | The code must not exceed 10 characters                                                                                                         |
| REQ-283 | The company profile must exist in the same property                                                                                            |
| REQ-284 | The name must be unique per property                                                                                                           |
| REQ-285 | The code must be unique per property                                                                                                           |
| REQ-286 | There must be a reservation groups table                                                                                                       |
| REQ-287 | The group's property must exist                                                                                                                |
| REQ-288 | The master folio must exist if set                                                                                                             |
| REQ-289 | The code is required                                                                                                                           |
| REQ-290 | The code must be in format GRP-XXXXX where X is a number                                                                                       |
| REQ-291 | Inserting an item into reservartion groups must increment the sequential                                                                       |
| REQ-292 | The group's name should not exceed 50 characters                                                                                               |
| REQ-293 | The group's notes should not exceed 2500 characters                                                                                            |
| REQ-294 | The group should be unique by code per property                                                                                                |
| REQ-295 | The group should unique by sequential per property                                                                                             |
| REQ-296 | There must be an index for the group's property                                                                                                |
| REQ-297 | The folio must exist in the same property                                                                                                      |
| REQ-298 | There must be a reservations table                                                                                                             |
| REQ-299 | The reservation's property must exist                                                                                                          |
| REQ-300 | The reservation's primary guest must exist                                                                                                     |
| REQ-301 | The reservation's group must exist if set                                                                                                      |
| REQ-302 | Inserting an row into reservations must increment the sequential                                                                               |
| REQ-303 | The reservation's code must be in format RES-XXXXXX where X is a number                                                                        |
| REQ-304 | The reservation's source must have an enum type                                                                                                |
| REQ-305 | The reservation's status must have an enum type                                                                                                |
| REQ-306 | The reservation's travel agent must exist if set                                                                                               |
| REQ-307 | The reservation's note's must not exceed 2500 characters                                                                                       |
| REQ-308 | There must be an index for the reservation's property                                                                                          |
| REQ-309 | There must be an index for the reservation's primary guest                                                                                     |
| REQ-310 | There must be an index for the reservation's group                                                                                             |
| REQ-311 | There must be an index for the reservation's travel agent                                                                                      |
| REQ-312 | There must be an index for the reservation's status                                                                                            |
| REQ-313 | There must be an index for the reservation's source                                                                                            |
| REQ-314 | The guest must exist in the same property                                                                                                      |
| REQ-315 | The group must exist in the same property                                                                                                      |
| REQ-316 | The travel agent must exist in the same property                                                                                               |
| REQ-317 | There must be a reservation items table                                                                                                        |
| REQ-318 | The reservation item's reservation must exist                                                                                                  |
| REQ-319 | The reservation item's booked_room_type must exist                                                                                             |
| REQ-320 | The reservation item's assigned_room must exist if set                                                                                         |
| REQ-321 | The reservation item's rate plan must exist                                                                                                    |
| REQ-322 | The stay period must be provided                                                                                                               |
| REQ-323 | The stay period must be in chronological order                                                                                                 |
| REQ-324 | The stay period must have both upper and lower values                                                                                          |
| REQ-325 | The stay period must satisfy length of stay requirements                                                                                       |
| REQ-326 | The stay period must not be in the past (upon creation)                                                                                        |
| REQ-327 | For each day of the stay period, there must be a booked_daily_rates row                                                                        |
| REQ-328 | The base rate must be in pence                                                                                                                 |
| REQ-329 | The adults count must not exceed the rooms max adult count                                                                                     |
| REQ-330 | The adults count must be a positive number                                                                                                     |
| REQ-331 | The children cont must not exceed the rooms max child count                                                                                    |
| REQ-332 | The children count must be a positive number                                                                                                   |
| REQ-333 | The total occupancy must not exceed the rooms max occupancy                                                                                    |
| REQ-334 | The total occupancy must not be lower than the rooms min occupancy                                                                             |
| REQ-335 | The status must be a type of enum                                                                                                              |
| REQ-336 | There must only ever be one reservation item in assigned to a room at any instance of time                                                     |
| REQ-337 | There must be an index for the reservation                                                                                                     |
| REQ-338 | There must be an index for the assigned room                                                                                                   |
| REQ-339 | There must be an index for the booked room type                                                                                                |
| REQ-340 | There must be an index for the rate plan                                                                                                       |
| REQ-341 | There must be an index for the status                                                                                                          |
| REQ-342 | There must be an index for the stay period                                                                                                     |
| REQ-343 | The reservation must be in the same property                                                                                                   |
| REQ-344 | The booked room type must be in the same property                                                                                              |
| REQ-345 | The assigned room must be in the same property                                                                                                 |
| REQ-346 | The rate plan must be in the same property                                                                                                     |
| REQ-347 | There must be a function for validating a rooms occupancy                                                                                      |
| REQ-348 | Each reservation item must have a booked daily rate                                                                                            |
| REQ-349 | The property must exist                                                                                                                        |
| REQ-350 | There must be reservation-items guests table                                                                                                   |
| REQ-351 | The reservation-items must exist                                                                                                               |
| REQ-352 | The guest must exist                                                                                                                           |
| REQ-353 | There must be an enum type for the guest role                                                                                                  |
| REQ-354 | There must be an index for the guest                                                                                                           |
| REQ-355 | There must be an index for the reservation item                                                                                                |
| REQ-356 | There must be an index for the guest role                                                                                                      |
| REQ-357 | The item and guest must exist in the same property                                                                                             |
| REQ-358 | There must be a booked daily rates table                                                                                                       |
| REQ-359 | The reservation item must exist                                                                                                                |
| REQ-360 | The rate plan must exist                                                                                                                       |
| REQ-361 | The calendar date must be in ISO-87601 format (YYYY-MM-DD)                                                                                     |
| REQ-362 | The calendar date must be in today or in the future                                                                                            |
| REQ-363 | The base price must be an positive integer                                                                                                     |
| REQ-364 | The adjustment must follow {type: 'fixed \| percentage', value: +-X, reason: 'y'}                                                              |
| REQ-365 | If the user can approve the adjustment, it will automatically be approved                                                                      |
| REQ-366 | The user who approves must exist                                                                                                               |
| REQ-367 | The final price in pence must be the base price after adjustment if approved                                                                   |
| REQ-368 | The final price must be a positive integer                                                                                                     |
| REQ-369 | Each day must have only one row per reservation item                                                                                           |
| REQ-370 | There must be an index for the booked daily rate's calendar date                                                                               |
| REQ-371 | There must be an index for the booked daily rate's reservation item                                                                            |
| REQ-372 | There must be an index for the booked daily rate's rate plan                                                                                   |
| REQ-373 | There must be an index for if the rate has been approved                                                                                       |
| REQ-374 | There must be an index for the approver of the rate                                                                                            |
| REQ-375 | There must be an index for the booked daily rate's calendar_date by rate plan                                                                  |
| REQ-376 | There must be a function for calculating the final price                                                                                       |
| REQ-377 | There must be a folios table                                                                                                                   |
| REQ-378 | A folio's property must exist                                                                                                                  |
| REQ-379 | A folio's reservation must exist if set                                                                                                        |
| REQ-380 | A folio's sales ledger must exist if set                                                                                                       |
| REQ-381 | There must be an enum type for folio part                                                                                                      |
| REQ-382 | The folio part is required                                                                                                                     |
| REQ-383 | The balance must be in pence                                                                                                                   |
| REQ-384 | The balance must be a positive integer                                                                                                         |
| REQ-385 | There must be an index on a folio's property                                                                                                   |
| REQ-386 | There must be an index for a folio's reservation                                                                                               |
| REQ-387 | There must be an index for a folio's sales ledger                                                                                              |
| REQ-388 | There must be a folio transactions table                                                                                                       |
| REQ-389 | A folio transaction's folio must exist                                                                                                         |
| REQ-390 | A folio transaction's ledger code must exist if set                                                                                            |
| REQ-391 | A folio transaction's description must not exceed 250 characters                                                                               |
| REQ-392 | A folio transaction's net unit price must be in pence                                                                                          |
| REQ-393 | A folio transaction's net unit price must be a int                                                                                             |
| REQ-394 | A folio transaction's quantity must be greater than 1                                                                                          |
| REQ-395 | A folio transaction's tax rule must exist if set                                                                                               |
| REQ-396 | A folio transaction's total net price must be generated                                                                                        |
| REQ-397 | A folio transaction's tax rate snapshot must be of the tax rule at time of transaction                                                         |
| REQ-398 | A folio transaction's tax amount must be a positive integer                                                                                    |
| REQ-399 | A folio transaction's tax amount must be generated                                                                                             |
| REQ-400 | A folio transaction's gross amount must be generated                                                                                           |
| REQ-401 | A folio transaction's posted only be updated when the status is changed to 'posted'                                                            |
| REQ-402 | A folio transaction's posted by user must exist if a folio is posted                                                                           |
| REQ-403 | A folio transaction's status must have an enum type                                                                                            |
| REQ-404 | There must be an index for a folio transaction's folio                                                                                         |
| REQ-405 | There must be an index for a folio transaction's ledger code                                                                                   |
| REQ-406 | There must be an index for a folio transaction's tax rule                                                                                      |
| REQ-407 | There must be an index for a folio transaction's posted by user                                                                                |
| REQ-408 | There must be an index for a folio transaction's status                                                                                        |
| REQ-409 | There must be an invoices table                                                                                                                |
| REQ-410 | The invoice's property must exist                                                                                                              |
| REQ-411 | The invoice's folio must exist if set                                                                                                          |
| REQ-412 | The invoice's property code msut be 3/4 characters                                                                                             |
| REQ-413 | The invoice's fiscal year is required                                                                                                          |
| REQ-414 | The invoice's fiscal year must be generated as the current fiscal year                                                                         |
| REQ-415 | The fiscal sequential must be unqiue by year by property                                                                                       |
| REQ-416 | The invoice number must be generated as PROPERTY_CODE-FISCAL_YEAR-FISCAL_SEQUENTIAL                                                            |
| REQ-417 | The billing address is required                                                                                                                |
| REQ-418 | The billing address must not exceed 300 characters                                                                                             |
| REQ-419 | The issue date must be the current date                                                                                                        |
| REQ-420 | The due date must be on/after the issue date                                                                                                   |
| REQ-421 | There must be an index for the invoice's folio                                                                                                 |
| REQ-422 | There must be an index for the invoice's property                                                                                              |
| REQ-423 | There must be an index for the invoice's fiscal year by property                                                                               |
| REQ-424 | There must be an index for the invoice's issue date by property                                                                                |
| REQ-425 | There must be an index for the invoice's due date by property                                                                                  |
| REQ-426 | There must be a sales ledger transactions table                                                                                                |
| REQ-427 | The SLTX's ledger account must exist                                                                                                           |
| REQ-428 | The SLTX's source invoice must exist if set                                                                                                    |
| REQ-429 | The SLTX's amount in pence must be an integer & is required                                                                                    |
| REQ-430 | The due date must be on/after the posted at date                                                                                               |
| REQ-431 | The due date must be defaulted to today + 30 days                                                                                              |
| REQ-432 | The is fully paid column must be generated as if the amount pence is <= 0                                                                      |
| REQ-433 | The posted by user must exist                                                                                                                  |
| REQ-434 | The type must have an enum type                                                                                                                |
| REQ-435 | There must be an index on a SLTX's ledger account                                                                                              |
| REQ-436 | There must be an index on a SLTX's invoice                                                                                                     |
| REQ-437 | There must be an index on a SLTX's posted timestamp                                                                                            |
| REQ-438 | There must be an index on a SLTX's type                                                                                                        |
| REQ-439 | There must be an index on a SLTX's due date                                                                                                    |
| REQ-440 | There must be an index on a SLTX's poster (user)                                                                                               |
| REQ-441 | There must be an index on a SLTX's paid status                                                                                                 |
| REQ-442 | There must be a checkout sessions table                                                                                                        |
| REQ-443 | The session's property must exist                                                                                                              |
| REQ-444 | The session's reservation must exist                                                                                                           |
| REQ-445 | The session's payment intent must exist                                                                                                        |
| REQ-446 | The session's expiration time must default to Now + 15 mins                                                                                    |
| REQ-447 | The idemoptency key must be unique and set                                                                                                     |
| REQ-448 | There must be an enum type for a session's status                                                                                              |
| REQ-449 | There must be an index for a session's property                                                                                                |
| REQ-450 | There must be an index for a session's reservation                                                                                             |
| REQ-451 | There must be an index for a session's payment intent                                                                                          |
| REQ-452 | There must be an index for a session's expiration time                                                                                         |
| REQ-453 | There must be an index for a session's status                                                                                                  |
| REQ-454 | There must be a room inventory ledger table                                                                                                    |
| REQ-455 | The RIL's room must exist                                                                                                                      |
| REQ-456 | The RIL's checkout session must exist if set                                                                                                   |
| REQ-457 | The RIL's calendar date is required                                                                                                            |
| REQ-458 | The RIL's calendar date must be in ISO 8601 format (YYYY-MM-DD)                                                                                |
| REQ-459 | There must be an enum type for the RIL's status                                                                                                |
| REQ-460 | Each row must be unique by room & calendar date                                                                                                |
| REQ-461 | There must be a constraint to ensure that a sold room has a reservation                                                                        |
| REQ-462 | There must be a constraint to ensure that on hold requires a checkout session                                                                  |
| REQ-463 | There must be an index for an RIL's room is available on a date                                                                                |
| REQ-464 | There must be an index for an RIL's availability by date                                                                                       |
| REQ-465 | There must be an index for an RIL's rooms by date                                                                                              |
| REQ-466 | There must be an index for an RIL's by calendar date                                                                                           |
| REQ-467 | There must be an index for an RIL's by status                                                                                                  |
| REQ-468 | There must be an index for an RIL's reservation                                                                                                |
| REQ-469 | There must be an index for an RIL's checkout session                                                                                           |
| REQ-470 | There must be an index for an RIL's room                                                                                                       |
| REQ-471 | There must be a housekeeping log table                                                                                                         |
| REQ-472 | The HSK log's property must exist                                                                                                              |
| REQ-473 | The HSK log's user must exist                                                                                                                  |
| REQ-474 | The HSK log's room must exist                                                                                                                  |
| REQ-475 | The status to & from must use the housekeeping_status enum                                                                                     |
| REQ-476 | The notes must not exceed 250 characters                                                                                                       |
| REQ-477 | There must be an index for a HSK log's property                                                                                                |
| REQ-478 | There must be an index for a HSK log's user                                                                                                    |
| REQ-479 | There must be an index for a HSK log's room                                                                                                    |
| REQ-480 | Must be an API endpoint for CreateReservation                                                                                                  |
| REQ-481 | User must be authorised to create reservation                                                                                                  |
| REQ-482 | If a user is not authorised, return 403                                                                                                        |
| REQ-483 | If the room is not available, return 409                                                                                                       |
| REQ-484 | If the room does not exist, return 404                                                                                                         |
| REQ-485 | If successful, return 209                                                                                                                      |
| REQ-486 | If 209, send to the reservation page                                                                                                           |
| REQ-487 | If 209, return details to JSON                                                                                                                 |
| REQ-488 | Must be an API endpoint for GetPlannerData                                                                                                     |
| REQ-489 | User must be authorised to see the planner                                                                                                     |
| REQ-490 | If user is not authorised, return 403                                                                                                          |
| REQ-491 | If the dates are invalid, return 409                                                                                                           |
| REQ-492 | If there is no data, 404                                                                                                                       |
| REQ-493 | If successful, return 200                                                                                                                      |
| REQ-494 | Return JSON object for the rows                                                                                                                |
| REQ-495 | Render in Svelte                                                                                                                               |
| REQ-496 | Implement reactive cache for planner data from start -> end date                                                                               |
| REQ-497 | If any of the planner data fetched is changed, invalid the cache                                                                               |
| REQ-498 | Attempt to hit cache first when getting planner data                                                                                           |
|         |                                                                                                                                                |