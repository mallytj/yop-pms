---
name: 🏗 Technical Design / Feature Plan
about: Plan out the schema, logic, and API contract before coding.
title: "[DESIGN]: "
labels: enhancement, design-phase
assignees: ""
---

## 🎯 Objective

_Briefly describe the business goal. What problem are we solving for the hotel?_

## 📝 RTM (Requirements Traceability Matrix)

_Link to the business requirement_

- [ ] REQ-0001:

## 📈 Relevant Diagram

_Provide a diagram_

## 💾 Proposed Schema Changes

_Describe new tables, columns, or modified relationships._

```sql
-- Paste SQL or describe changes here
```

## 🔌 API Contract

| Method | Endpoint      | Description          |
| :----- | :------------ | :------------------- |
| GET    | `/v1/example` | Example API contract |

### Expected Payload / Result

#### 📥 Payload

```json
{
  "example": "data"
}
```

#### ✅ Success

```json
{
  "example": "data"
}
```

#### 🚫 Error Responses

| Status | Message     | Reason |
| :----- | :---------- | :----- |
| 400    | Bad Request |        |

### 🔒 Handler Level Validation

#### Validation Matricies

#### 🔍 **Required Fields:**

-

#### 🔍 Data Types

-

#### 🤖 Logic

-

### 🔒 Service Level Validation

#### 🚫 Permissions

-

#### 💍 Relationship

-

#### 💼 Business Logic

-

### 📏 Potential Edge Cases

-

### 🧪 Testing Plan

| Test | Description | Expected Result |
| :--- | :---------- | :-------------- |
|      |             |                 |

## 📄 Docs

- [ ] Swagger comments added to `...`
- [ ] Readme updated for

### Swagger snippet

```go
// ExampleHandler godoc
// @Summary      Example API
// @Description  Returns a response
// @Tags
// @Param        example   query     string  true
// @Param        check_out  query     string  true
// @Success      200        {array}   models.ExampleResponse
// @Router       /availability [get]
func (h *AvailabilityHandler) GetAvailability(c *gin.Context) { ... }
```
