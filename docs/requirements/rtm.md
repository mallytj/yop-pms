| ID      | Requirement Description                                                                           |
| ------- | ------------------------------------------------------------------------------------------------- |
| REQ-001 | Database must support SQL-based migrations via Goose                                              |
| REQ-002 | All Primary Keys must use UUIDv7 for time-sortable uniqueness                                     |
| REQ-003 | Monetary values must be stored as integers (pence)                                                |
| REQ-004 | Audit fields (created_at, updated_at, deleted_at) on all core tables                              |
| REQ-005 | Any indexes created on a table dependent on property must include the property_id                 |
| REQ-006 | All foreign keys must have ON DELETE and ON UPDATE actions defined                                |
| REQ-007 | All text/varchar columns must have CHECK constraints enforcing length                             |
| REQ-008 | All tables must have a schema defined                                                             |
| REQ-009 | All timestamps must be stored as TIMESTAMPTZ (UTC).                                               |
| REQ-010 | High-concurrency tables must support Optimistic Locking (via a version column).                   |
| REQ-011 | All Foreign Key columns must have an explicit Index.                                              |
| REQ-012 | Constraint names must follow a strict convention ({table}_{column}_{suffix}).                     |
| REQ-013 | All foreign keys must RESTRICT NOT CASCADE to preserve historical data                            |
| REQ-014 | If a reference depends on property, there must be a FK created references the property and entity |
| REQ-015 | Any boolean columns must have a set default                                                       |
| REQ-016 | Initialize extensions (uuid-ossp, gist)                                                           |
| REQ-017 | Create empty schemas (operations, relations, identity, auth, etc.)                                |
| REQ-018 | Each string type where the values are preset MUST be an enum                                      |
| REQ-019 | System must store Staff/Admin user credentials and logs                                           |
| REQ-020 | There must be a table for licences                                                                |
| REQ-021 | The licence key must be unique                                                                    |
| REQ-022 | The licence key must be in format YOP-XXXX where x is an int                                      |
| REQ-023 | The organisation name is required                                                                 |
| REQ-024 | The organisation name must not exceed 50 characters                                               |
| REQ-025 | The contact email is required                                                                     |
| REQ-026 | The contact email must be valid                                                                   |
| REQ-027 | The licence notes must not exceed 1500 characters                                                 |
| REQ-028 | Licence must be inactive if soft deleted                                                          |
| REQ-029 | There must be a table for properties                                                              |
| REQ-030 | The licence must exist & be active                                                                |
| REQ-031 | The name is required                                                                              |
| REQ-032 | The name must not exceed 50 characters                                                            |
| REQ-033 | The address is required                                                                           |
| REQ-034 | The address must not exceed 250 characters                                                        |
| REQ-035 | The timezone is required                                                                          |
| REQ-036 | The timezone must be in IANA format                                                               |
| REQ-037 | The property notes must not exceed 1500 characters                                                |
| REQ-038 | There must be only one property per address per licence                                           |
| REQ-039 | There must be an index on properties for licences                                                 |
| REQ-040 | There must be a function for checking if a licence exists on a property                           |
| REQ-041 | There must be a index for a properties name by licence                                            |
| REQ-042 | There must be an index for if a property is active by licence                                     |
| REQ-043 | There must be a table for users                                                                   |
| REQ-044 | The licence must exist & be active                                                                |
| REQ-045 | The username is required                                                                          |
| REQ-046 | The username must be unique                                                                       |
| REQ-047 | The username must be alphanumerical or \_                                                         |
| REQ-048 | The username must not exceed 20 characters                                                        |
| REQ-049 | The email is required                                                                             |
| REQ-050 | The password_hash is required                                                                     |
| REQ-051 | The role is required                                                                              |
| REQ-052 | The role must be a valid role                                                                     |
| REQ-053 | The first name is required                                                                        |
| REQ-054 | The first name must not exceed 50 characters                                                      |
| REQ-055 | The first name must not have any special characters or numbers (other than -)                     |
| REQ-056 | The last name is required                                                                         |
| REQ-057 | The last name must not exceed 50 characters                                                       |
| REQ-058 | The last name must not have any special characters or numbers (other than -)                      |
| REQ-059 | A user's email address must be unique                                                             |
| REQ-060 | A user's username must be unique                                                                  |
| REQ-061 | A user's password must not be stored in plain text                                                |
| REQ-062 | A user's email must be valid                                                                      |
| REQ-063 | There must be an enum for a user's roles                                                          |
| REQ-064 | There must be an index on users for licences                                                      |
| REQ-065 | There must be an index for users for name (Last, First)                                           |
| REQ-066 | There must be an index for user's role                                                            |
| REQ-067 | There must be an index for if a user is active                                                    |
| REQ-068 | The email must be unique                                                                          |
| REQ-069 | There must be an index for a user's email                                                         |
| REQ-070 | There must be a table for guests                                                                  |
| REQ-071 | The guests property must exist                                                                    |
| REQ-072 | The first name is required                                                                        |
| REQ-073 | The first name must not exceed 50 characters                                                      |
| REQ-074 | The first name must not have any special characters or numbers (other than -)                     |
| REQ-075 | The last name is required                                                                         |
| REQ-076 | The last name must not exceed 50 characters                                                       |
| REQ-077 | The last name must not have any special characters or numbers (other than -)                      |
| REQ-078 | The marketing opt in must be defaulted as false for GDPR                                          |
| REQ-079 | Annoymising a guest must hide all personal data                                                   |
| REQ-080 | A guest's email must be valid                                                                     |
| REQ-081 | A guest's phone number must be valid                                                              |
| REQ-082 | A guest must have an is_anonymised column for GDPR                                                |
| REQ-083 | There must be a index for a guest's name (Last, First)                                            |
| REQ-084 | There Must be an index for a guests email                                                         |
| REQ-085 | There must be an index for a guests phone number                                                  |
| REQ-086 | There must be an index for a guests marketing preference                                          |
| REQ-087 | There must be an index for if a guest is anonymised                                               |
| REQ-088 | There must be an index for a guests property                                                      |
| REQ-089 | There must be a table for audit logs                                                              |
| REQ-090 | The user must exist                                                                               |
| REQ-091 | The entity must exist                                                                             |
| REQ-092 | The action is required                                                                            |
| REQ-093 | The entity is required                                                                            |
| REQ-094 | Change must be in format {field: X, before: Y, after: Z)                                          |
| REQ-095 | There must be an enum for a audit log's entity                                                    |
| REQ-096 | There must be an enum for a audit log's type                                                      |
| REQ-097 | There must be an index for an audit log's user                                                    |
| REQ-098 | There must be an index for an audit log's entity                                                  |
| REQ-099 | There must be an index for an audit log's action                                                  |
| REQ-100 | The property must exist                                                                           |
| REQ-101 | The entity ID is required                                                                         |
| REQ-102 | There must be a table for amenities                                                               |
| REQ-103 | The amenity's property must exist                                                                 |
| REQ-104 | The amenity's name is required                                                                    |
| REQ-105 | The amenity's name must not exceed 100 characters                                                 |
| REQ-106 | The amenity's short code must not exceed 5 characters                                             |
| REQ-107 | The amenity's description must not exceed 250 characters                                          |
| REQ-108 | An amenity's short_code must be unique for each property                                          |
| REQ-109 | An amenity's name must be unique for each property                                                |
| REQ-110 | There must be an index for an amenities property                                                  |
| REQ-111 | There must be an index for if an amenity is active                                                |
| REQ-112 | The shortcode must be alphanumerical or \ \_ or / or -                                            |
| REQ-113 | There must be a join table for property amenities                                                 |
| REQ-114 | The property must exist                                                                           |
| REQ-115 | The amenity must exist                                                                            |
| REQ-116 | There must be an index for the property                                                           |
| REQ-117 | There must be an index for the amenity                                                            |
| REQ-118 | The amenity must be of the same property id as the relation                                       |
| REQ-119 | There must be a table for travel agents                                                           |
| REQ-120 | The travel agents name is required                                                                |
| REQ-121 | A travel agent's email must be valid                                                              |
| REQ-122 | A travel agent's phone number must be valid                                                       |
| REQ-123 | A travel agent's name must not exceed 100 characters                                              |
| REQ-124 | A travel agent's notes must not exceed 1500 characters                                            |
| REQ-125 | A travel agent's name must be unique by property                                                  |
| REQ-126 | The commisson percentage must be a positive integer                                               |
| REQ-127 | The commission percentage must not exceed 75%                                                     |
| REQ-128 | There must be an index for a travel agent's property                                              |
| REQ-129 | There must be an identity docs table                                                              |
| REQ-130 | The doc type is required and must be in the enum                                                  |
| REQ-131 | The issuing country must be in ISO format                                                         |
| REQ-132 | The doc image url must be valid                                                                   |
| REQ-133 | There must be a doc number                                                                        |
| REQ-134 | An identity doc's guest must exist                                                                |
| REQ-135 | There must be an enum for doc type                                                                |
| REQ-136 | The doc number must be encrypted                                                                  |
| REQ-137 | There must be an index on the identity doc's guest                                                |
| REQ-138 | The issuing country is required                                                                   |
| REQ-139 | There must be a room types table                                                                  |
| REQ-140 | The room type's property must exist                                                               |
| REQ-141 | The room type's name is required                                                                  |
| REQ-142 | The room type's name must not exceed 75 characters                                                |
| REQ-143 | The room type's code is required                                                                  |
| REQ-144 | The room type's code must not exceed 7 characters                                                 |
| REQ-145 | The standard occupancy must be a postive integer                                                  |
| REQ-146 | The min occupancy must be a positive integer                                                      |
| REQ-147 | The min occupancy must be less than or equal to the standard occupancy                            |
| REQ-148 | The max occupancy must be a positive integer                                                      |
| REQ-149 | The max occupancy must be greater than or equal to the standard occupancy                         |
| REQ-150 | A room type's code must be unique by property                                                     |
| REQ-151 | A room type's name must by unqiue by property                                                     |
| REQ-152 | There must be an index for a room type's property                                                 |
| REQ-153 | There must be a join table for a room type's amenities                                            |
| REQ-154 | The room type must exist                                                                          |
| REQ-155 | The amenity must exist                                                                            |
| REQ-156 | There must be an index for the room type                                                          |
| REQ-157 | There must be an index for the amenity                                                            |
| REQ-158 | The amenity must be of the same property id as the room type                                      |
| REQ-159 | There must be a rooms table                                                                       |
| REQ-160 | The rooms name must not exceed 75 characters                                                      |
| REQ-161 | The rooms property must exist                                                                     |
| REQ-162 | The rooms room type must exist                                                                    |
| REQ-163 | There must be an enum for housekeeping status                                                     |
| REQ-164 | There must be an enum for occupancy status                                                        |
| REQ-165 | A room's name must be unique by property                                                          |
| REQ-166 | There must be an index for a rooms room type                                                      |
| REQ-167 | There must be an index for a room's housekeeping status                                           |
| REQ-168 | There must be an index for a room's occupancy status                                              |
| REQ-169 | There must be an index for a room's property                                                      |
| REQ-170 | There must be a join table for a rooms amenities                                                  |
| REQ-171 | The room must exist                                                                               |
| REQ-172 | The amenity must exist                                                                            |
| REQ-173 | There must be an index for the room                                                               |
| REQ-174 | There must be an index for the amenity                                                            |
| REQ-175 | The amentiy & room must be owned by the same property                                             |
| REQ-176 | There must be a table for maintenace blocks                                                       |
| REQ-177 | The maintenance block's room must exist                                                           |
| REQ-178 | The maintenace block's creator must exist                                                         |
| REQ-179 | A maintenace blocks block_period must be Start->End not end->start                                |
| REQ-180 | There must be an enum for a maintenace blocks type                                                |
| REQ-181 | A maintenace block's reason must not be longer than 150 characters                                |
| REQ-182 | There must not be multiple maintenace blocks on one room at the same time                         |
| REQ-183 | There must be an index for a maintenace block's room by block_period                              |
| REQ-184 | There must be an index for a maintenace block's block_period                                      |
| REQ-185 | There must be an index for a maintenace block's type                                              |
| REQ-186 | There must be an index for a maintenace block's created by user                                   |
| REQ-187 | There must be an index for a maintenace block's room                                              |
| REQ-188 | Room ID is required                                                                               |
| REQ-189 | Block Period is required                                                                          |
| REQ-190 | Reason is required                                                                                |
| REQ-191 | Type is required                                                                                  |
| REQ-192 | Created by user id is required                                                                    |
| REQ-193 | There must be a table for rate plans                                                              |
| REQ-194 | The rate plan name is required                                                                    |
| REQ-195 | The rate plan code is required                                                                    |
| REQ-196 | A rate plan's derivation must be in format {'type': 'percentage' \| 'fixed', value: '+-x'}        |
| REQ-197 | A rate plans currency code must be in a ISO 4217 format                                           |
| REQ-198 | A rate plans code must not be longer than 7 characters                                            |
| REQ-199 | A rate plans name must not be longer than 30 characters                                           |
| REQ-200 | A rate plans description must not be longer than 300 characters                                   |
| REQ-201 | There must be an index for a rate plans property                                                  |
| REQ-202 | There must be an index for a rate plans parent rate plan                                          |
| REQ-203 | There must be an index for if a rate plan is active                                               |
| REQ-204 | If derivation rule is set, there must be a parent rate plan that exists                           |
| REQ-205 | The parent rate plan must belong to the same property                                             |
| REQ-206 | If there is no parent rate plan, there can not be a derivation rule                               |
| REQ-207 | There must be a company profiles table                                                            |
| REQ-208 | The company name must be unique by property                                                       |
| REQ-209 | The tax ID must be in a valid format                                                              |
| REQ-210 | The company profile's property must exist                                                         |
| REQ-211 | The negotiated rate plan ID must exist (if not null)                                              |
| REQ-212 | The company name must not exceed 50 characters                                                    |
| REQ-213 | The company name is required                                                                      |
| REQ-214 | The contact email must be in a valid format                                                       |
| REQ-215 | The contact phone number must be in a valid format                                                |
| REQ-216 | The billing address must be in a valid format                                                     |
| REQ-217 | The billing address must not exceed 300 characters                                                |
| REQ-218 | The company notes must not exceed 1500 characters                                                 |
| REQ-219 | There must be an index for a company profile's property                                           |
| REQ-220 | There must be an index for a company profile's negotiated rate plan                               |
| REQ-221 | The rate plan must exist in the same property                                                     |
| REQ-222 | The tax ID must be unique per property                                                            |
| REQ-223 | There must be a daily price grid table                                                            |
| REQ-224 | The room type must exist                                                                          |
| REQ-225 | The rate plan must exist                                                                          |
| REQ-226 | The property must exist                                                                           |
| REQ-227 | The calendar date is required                                                                     |
| REQ-228 | The calendar date must be in ISO-87601 format (YYYY-MM-DD)                                        |
| REQ-229 | On creation, the calendar date nsut be in the future                                              |
| REQ-230 | The base price must be an integer                                                                 |
| REQ-231 | The min_los_restrction must be positive                                                           |
| REQ-232 | The max_los_restriction must be positive                                                          |
| REQ-233 | The max_los_restriction must be greater than the min_los_restriction                              |
| REQ-234 | The row must be unique by room type, calendar date and rate plan                                  |
| REQ-235 | There must be an index for a daily price's room type                                              |
| REQ-236 | There must be an index for a daily price's rate plan                                              |
| REQ-237 | There must be an index for a daily price's date                                                   |
| REQ-238 | There must be an index for a daily price's availability                                           |
| REQ-239 | There must be an index for a daily price's room type, by date when available                      |
| REQ-240 | There must be an index for a daily price's rate plan, by date when available                      |
| REQ-241 | The room type must exist in the same property                                                     |
| REQ-242 | The rate plan must exist in the same property                                                     |
| REQ-243 | There must be a tax rules table                                                                   |
| REQ-244 | The tax rule's propery must exist                                                                 |
| REQ-245 | The tax rule's name is required                                                                   |
| REQ-246 | The tax rule's name must not exceed 50 characters                                                 |
| REQ-247 | The tax rules description must not exceed 250 characters                                          |
| REQ-248 | The tax percentage must not exceed 75%                                                            |
| REQ-249 | The tax percentage must be positive                                                               |
| REQ-250 | The tax rule name must be unique by property                                                      |
| REQ-251 | There must be an index for a tax rule's property                                                  |
| REQ-252 | There must be a ledger code table                                                                 |
| REQ-253 | The ledger code's property must exist                                                             |
| REQ-254 | The code is required                                                                              |
| REQ-255 | The code must be unique to the property                                                           |
| REQ-256 | The code must not exceed 50 characters                                                            |
| REQ-257 | The description must not exceed 250 characters                                                    |
| REQ-258 | The ledger code's tax rule must exist if set                                                      |
| REQ-259 | There must be an index for a ledger code's property                                               |
| REQ-260 | There must be an index for a ledger code's tax rule                                               |
| REQ-261 | There must be a accounts table                                                                    |
| REQ-262 | The account's property must exist                                                                 |
| REQ-263 | The account's company profile must exist if set                                                   |
| REQ-264 | The credit limit must be a positive integer                                                       |
| REQ-265 | The payment terms must be a positive integer                                                      |
| REQ-266 | Each company profile should only have one account per property                                    |
| REQ-267 | There must be an index for account's property                                                     |
| REQ-268 | There must be an index for the account's company profile                                          |
| REQ-269 | The name is required                                                                              |
| REQ-270 | The name must not exceed 100 characters                                                           |
| REQ-271 | The code is required                                                                              |
| REQ-272 | The code must not exceed 10 characters                                                            |
| REQ-273 | The company profile must exist in the same property                                               |
| REQ-274 | The name must be unique per property                                                              |
| REQ-275 | The code must be unique per property                                                              |
| REQ-276 | There must be a reservation groups table                                                          |
| REQ-277 | The group's property must exist                                                                   |
| REQ-278 | The master folio must exist if set                                                                |
| REQ-279 | The code is required                                                                              |
| REQ-280 | The code must be in format GRP-XXXXX where X is a number                                          |
| REQ-281 | Inserting an item into reservartion groups must increment the sequential                          |
| REQ-282 | The group's name should not exceed 50 characters                                                  |
| REQ-283 | The group's notes should not exceed 2500 characters                                               |
| REQ-284 | The group should be unique by code per property                                                   |
| REQ-285 | The group should unique by sequential per property                                                |
| REQ-286 | There must be an index for the group's property                                                   |
| REQ-287 | The folio must exist in the same property                                                         |
| REQ-288 | There must be a reservations table                                                                |
| REQ-289 | The reservation's property must exist                                                             |
| REQ-290 | The reservation's primary guest must exist                                                        |
| REQ-291 | The reservation's group must exist if set                                                         |
| REQ-292 | Inserting an row into reservations must increment the sequential                                  |
| REQ-293 | The reservation's code must be in format RES-XXXXXX where X is a number                           |
| REQ-294 | The reservation's source must have an enum type                                                   |
| REQ-295 | The reservation's status must have an enum type                                                   |
| REQ-296 | The reservation's travel agent must exist if set                                                  |
| REQ-297 | The reservation's note's must not exceed 2500 characters                                          |
| REQ-298 | There must be an index for the reservation's property                                             |
| REQ-299 | There must be an index for the reservation's primary guest                                        |
| REQ-300 | There must be an index for the reservation's group                                                |
| REQ-301 | There must be an index for the reservation's travel agent                                         |
| REQ-302 | There must be an index for the reservation's status                                               |
| REQ-303 | There must be an index for the reservation's source                                               |
| REQ-304 | The guest must exist in the same property                                                         |
| REQ-305 | The group must exist in the same property                                                         |
| REQ-306 | The travel agent must exist in the same property                                                  |
| REQ-307 | There must be a reservation items table                                                           |
| REQ-308 | The reservation item's reservation must exist                                                     |
| REQ-309 | The reservation item's booked_room_type must exist                                                |
| REQ-310 | The reservation item's assigned_room must exist if set                                            |
| REQ-311 | The reservation item's rate plan must exist                                                       |
| REQ-312 | The stay period must be provided                                                                  |
| REQ-313 | The stay period must be in chronological order                                                    |
| REQ-314 | The stay period must have both upper and lower values                                             |
| REQ-315 | The stay period must satisfy length of stay requirements                                          |
| REQ-316 | The stay period must not be in the past (upon creation)                                           |
| REQ-317 | For each day of the stay period, there must be a booked_daily_rates row                           |
| REQ-318 | The base rate must be in pence                                                                    |
| REQ-319 | The adults count must not exceed the rooms max adult count                                        |
| REQ-320 | The adults count must be a positive number                                                        |
| REQ-321 | The children cont must not exceed the rooms max child count                                       |
| REQ-322 | The children count must be a positive number                                                      |
| REQ-323 | The total occupancy must not exceed the rooms max occupancy                                       |
| REQ-324 | The total occupancy must not be lower than the rooms min occupancy                                |
| REQ-325 | The status must be a type of enum                                                                 |
| REQ-326 | There must only ever be one reservation item in assigned to a room at any instance of time        |
| REQ-327 | There must be an index for the reservation                                                        |
| REQ-328 | There must be an index for the assigned room                                                      |
| REQ-329 | There must be an index for the booked room type                                                   |
| REQ-330 | There must be an index for the rate plan                                                          |
| REQ-331 | There must be an index for the status                                                             |
| REQ-332 | There must be an index for the stay period                                                        |
| REQ-333 | The reservation must be in the same property                                                      |
| REQ-334 | The booked room type must be in the same property                                                 |
| REQ-335 | The assigned room must be in the same property                                                    |
| REQ-336 | The rate plan must be in the same property                                                        |
| REQ-337 | There must be a function for validating a rooms occupancy                                         |
| REQ-338 | Each reservation item must have a booked daily rate                                               |
| REQ-339 | The property must exist                                                                           |
| REQ-340 | There must be reservation-items guests table                                                      |
| REQ-341 | The reservation-items must exist                                                                  |
| REQ-342 | The guest must exist                                                                              |
| REQ-343 | There must be an enum type for the guest role                                                     |
| REQ-344 | There must be an index for the guest                                                              |
| REQ-345 | There must be an index for the reservation item                                                   |
| REQ-346 | There must be an index for the guest role                                                         |
| REQ-347 | The item and guest must exist in the same property                                                |
| REQ-348 | There must be a booked daily rates table                                                          |
| REQ-349 | The reservation item must exist                                                                   |
| REQ-350 | The rate plan must exist                                                                          |
| REQ-351 | The calendar date must be in ISO-87601 format (YYYY-MM-DD)                                        |
| REQ-352 | The calendar date must be in today or in the future                                               |
| REQ-353 | The base price must be an positive integer                                                        |
| REQ-354 | The adjustment must follow {type: 'fixed \| percentage', value: +-X, reason: 'y'}                 |
| REQ-355 | If the user can approve the adjustment, it will automatically be approved                         |
| REQ-356 | The user who approves must exist                                                                  |
| REQ-357 | The final price in pence must be the base price after adjustment if approved                      |
| REQ-358 | The final price must be a positive integer                                                        |
| REQ-359 | Each day must have only one row per reservation item                                              |
| REQ-360 | There must be an index for the booked daily rate's calendar date                                  |
| REQ-361 | There must be an index for the booked daily rate's reservation item                               |
| REQ-362 | There must be an index for the booked daily rate's rate plan                                      |
| REQ-363 | There must be an index for if the rate has been approved                                          |
| REQ-364 | There must be an index for the approver of the rate                                               |
| REQ-365 | There must be an index for the booked daily rate's calendar_date by rate plan                     |
| REQ-366 | There must be a function for calculating the final price                                          |
| REQ-367 | There must be a folios table                                                                      |
| REQ-368 | A folio's property must exist                                                                     |
| REQ-369 | A folio's reservation must exist if set                                                           |
| REQ-370 | A folio's sales ledger must exist if set                                                          |
| REQ-371 | There must be an enum type for folio part                                                         |
| REQ-372 | The folio part is required                                                                        |
| REQ-373 | The balance must be in pence                                                                      |
| REQ-374 | The balance must be a positive integer                                                            |
| REQ-375 | There must be an index on a folio's property                                                      |
| REQ-376 | There must be an index for a folio's reservation                                                  |
| REQ-377 | There must be an index for a folio's sales ledger                                                 |
| REQ-378 | There must be a folio transactions table                                                          |
| REQ-379 | A folio transaction's folio must exist                                                            |
| REQ-380 | A folio transaction's ledger code must exist if set                                               |
| REQ-381 | A folio transaction's description must not exceed 250 characters                                  |
| REQ-382 | A folio transaction's net unit price must be in pence                                             |
| REQ-383 | A folio transaction's net unit price must be a int                                                |
| REQ-384 | A folio transaction's quantity must be greater than 1                                             |
| REQ-385 | A folio transaction's tax rule must exist if set                                                  |
| REQ-386 | A folio transaction's total net price must be generated                                           |
| REQ-387 | A folio transaction's tax rate snapshot must be of the tax rule at time of transaction            |
| REQ-388 | A folio transaction's tax amount must be a positive integer                                       |
| REQ-389 | A folio transaction's tax amount must be generated                                                |
| REQ-390 | A folio transaction's gross amount must be generated                                              |
| REQ-391 | A folio transaction's posted only be updated when the status is changed to 'posted'               |
| REQ-392 | A folio transaction's posted by user must exist if a folio is posted                              |
| REQ-393 | A folio transaction's status must have an enum type                                               |
| REQ-394 | There must be an index for a folio transaction's folio                                            |
| REQ-395 | There must be an index for a folio transaction's ledger code                                      |
| REQ-396 | There must be an index for a folio transaction's tax rule                                         |
| REQ-397 | There must be an index for a folio transaction's posted by user                                   |
| REQ-398 | There must be an index for a folio transaction's status                                           |
| REQ-399 | There must be an invoices table                                                                   |
| REQ-400 | The invoice's property must exist                                                                 |
| REQ-401 | The invoice's folio must exist if set                                                             |
| REQ-402 | The invoice's property code msut be 3/4 characters                                                |
| REQ-403 | The invoice's fiscal year is required                                                             |
| REQ-404 | The invoice's fiscal year must be generated as the current fiscal year                            |
| REQ-405 | The fiscal sequential must be unqiue by year by property                                          |
| REQ-406 | The invoice number must be generated as PROPERTY_CODE-FISCAL_YEAR-FISCAL_SEQUENTIAL               |
| REQ-407 | The billing address is required                                                                   |
| REQ-408 | The billing address must not exceed 300 characters                                                |
| REQ-409 | The issue date must be the current date                                                           |
| REQ-410 | The due date must be on/after the issue date                                                      |
| REQ-411 | There must be an index for the invoice's folio                                                    |
| REQ-412 | There must be an index for the invoice's property                                                 |
| REQ-413 | There must be an index for the invoice's fiscal year by property                                  |
| REQ-414 | There must be an index for the invoice's issue date by property                                   |
| REQ-415 | There must be an index for the invoice's due date by property                                     |
| REQ-416 | There must be a sales ledger transactions table                                                   |
| REQ-417 | The SLTX's ledger account must exist                                                              |
| REQ-418 | The SLTX's source invoice must exist if set                                                       |
| REQ-419 | The SLTX's amount in pence must be an integer & is required                                       |
| REQ-420 | The due date must be on/after the posted at date                                                  |
| REQ-421 | The due date must be defaulted to today + 30 days                                                 |
| REQ-422 | The is fully paid column must be generated as if the amount pence is <= 0                         |
| REQ-423 | The posted by user must exist                                                                     |
| REQ-424 | The type must have an enum type                                                                   |
| REQ-425 | There must be an index on a SLTX's ledger account                                                 |
| REQ-426 | There must be an index on a SLTX's invoice                                                        |
| REQ-427 | There must be an index on a SLTX's posted timestamp                                               |
| REQ-428 | There must be an index on a SLTX's type                                                           |
| REQ-429 | There must be an index on a SLTX's due date                                                       |
| REQ-430 | There must be an index on a SLTX's poster (user)                                                  |
| REQ-431 | There must be an index on a SLTX's paid status                                                    |
| REQ-432 | There must be a checkout sessions table                                                           |
| REQ-433 | The session's property must exist                                                                 |
| REQ-434 | The session's reservation must exist                                                              |
| REQ-435 | The session's payment intent must exist                                                           |
| REQ-436 | The session's expiration time must default to Now + 15 mins                                       |
| REQ-437 | The idemoptency key must be unique and set                                                        |
| REQ-438 | There must be an enum type for a session's status                                                 |
| REQ-439 | There must be an index for a session's property                                                   |
| REQ-440 | There must be an index for a session's reservation                                                |
| REQ-441 | There must be an index for a session's payment intent                                             |
| REQ-442 | There must be an index for a session's expiration time                                            |
| REQ-443 | There must be an index for a session's status                                                     |
| REQ-444 | There must be a room inventory ledger table                                                       |
| REQ-445 | The RIL's room must exist                                                                         |
| REQ-446 | The RIL's checkout session must exist if set                                                      |
| REQ-447 | The RIL's calendar date is required                                                               |
| REQ-448 | The RIL's calendar date must be in ISO 8601 format (YYYY-MM-DD)                                   |
| REQ-449 | There must be an enum type for the RIL's status                                                   |
| REQ-450 | Each row must be unique by room & calendar date                                                   |
| REQ-451 | There must be a constraint to ensure that a sold room has a reservation                           |
| REQ-452 | There must be a constraint to ensure that on hold requires a checkout session                     |
| REQ-453 | There must be an index for an RIL's room is available on a date                                   |
| REQ-454 | There must be an index for an RIL's availability by date                                          |
| REQ-455 | There must be an index for an RIL's rooms by date                                                 |
| REQ-456 | There must be an index for an RIL's by calendar date                                              |
| REQ-457 | There must be an index for an RIL's by status                                                     |
| REQ-458 | There must be an index for an RIL's reservation                                                   |
| REQ-459 | There must be an index for an RIL's checkout session                                              |
| REQ-460 | There must be an index for an RIL's room                                                          |
| REQ-461 | There must be a housekeeping log table                                                            |
| REQ-462 | The HSK log's property must exist                                                                 |
| REQ-463 | The HSK log's user must exist                                                                     |
| REQ-464 | The HSK log's room must exist                                                                     |
| REQ-465 | The status to & from must use the housekeeping_status enum                                        |
| REQ-466 | The notes must not exceed 250 characters                                                          |
| REQ-467 | There must be an index for a HSK log's property                                                   |
| REQ-468 | There must be an index for a HSK log's user                                                       |
| REQ-469 | There must be an index for a HSK log's room                                                       |
| REQ-470 | Must be an API endpoint for CreateReservation                                                     |
| REQ-471 | User must be authorised to create reservation                                                     |
| REQ-472 | If a user is not authorised, return 403                                                           |
| REQ-473 | If the room is not available, return 409                                                          |
| REQ-474 | If the room does not exist, return 404                                                            |
| REQ-475 | If successful, return 209                                                                         |
| REQ-476 | If 209, send to the reservation page                                                              |
| REQ-477 | If 209, return details to JSON                                                                    |
| REQ-478 | Must be an API endpoint for GetPlannerData                                                        |
| REQ-479 | User must be authorised to see the planner                                                        |
| REQ-480 | If user is not authorised, return 403                                                             |
| REQ-481 | If the dates are invalid, return 409                                                              |
| REQ-482 | If there is no data, 404                                                                          |
| REQ-483 | If successful, return 200                                                                         |
| REQ-484 | Return JSON object for the rows                                                                   |
| REQ-485 | Render in Svelte                                                                                  |
| REQ-486 | Implement reactive cache for planner data from start -> end date                                  |
| REQ-487 | If any of the planner data fetched is changed, invalid the cache                                  |
| REQ-488 | Attempt to hit cache first when getting planner data                                              |