# Creating Points and Trending Data

This guide explains how to create points dynamically from external data sources and trend their values in the Normal Framework. This is essential for integrating third-party APIs, IoT devices, and other data sources that aren't directly connected via BACnet or Modbus.

## Table of Contents

- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Step-by-Step Process](#step-by-step-process)
- [Point Structure](#point-structure)
- [Adding Trending Data](#adding-trending-data)
- [Layer Configuration](#layer-configuration)
- [Complete Example](#complete-example)
- [Best Practices](#best-practices)

---

## Overview

When integrating external data sources (weather APIs, IoT sensors, cloud services), you need to:

1. **Create a layer** with indexed attributes for your data source
2. **Generate consistent UUIDs** for points using UUID v5
3. **Upsert points** via the Point API
4. **Add time-series data** for trending

This pattern allows you to dynamically create and update points as new sensors or data sources are added.

---

## Prerequisites

Before creating points programmatically, you must:

1. **Create a layer** in the Normal Framework UI for your data source
2. **Define indexed attributes** that you'll use to search and filter points
3. **Configure your hook** with necessary API keys and parameters

### Example Layer Definition

Create a layer with structured components for the attributes you'll store:

```json
{
    "name": "purpleair",
    "indexed": true,
    "structured_components": [
        {
            "name": "sensor_index",
            "type": "TAG"
        },
        {
            "name": "field_name",
            "type": "TAG"
        },
        {
            "name": "source",
            "type": "TAG"
        },
        {
            "name": "latitude",
            "type": "NUMERIC"
        },
        {
            "name": "longitude",
            "type": "NUMERIC"
        }
    ]
}
```

**Attribute Types:**
- `TAG` - String values for filtering (e.g., sensor IDs, field names)
- `NUMERIC` - Numeric values for range queries (e.g., coordinates, thresholds)

---

## Step-by-Step Process

### 1. Generate Consistent UUIDs

Use UUID v5 to create deterministic UUIDs based on your data source identifier. This ensures the same point always gets the same UUID across runs.

```javascript
const { v5: uuidv5 } = require("uuid");

// Define a namespace UUID (can be any valid UUID)
const NAMESPACE = "6ba7b810-9dad-11d1-80b4-00c04fd430c8";

function generatePointUuid(sensorId, fieldName) {
  const name = `my-source-${sensorId}-${fieldName}`;
  return uuidv5(name, NAMESPACE);
}
```

**Why UUID v5?**
- Deterministic: Same input always produces the same UUID
- Prevents duplicate points when re-running hooks
- Allows updates to existing points via upsert

### 2. Build Point Objects

Create point objects with required fields and custom attributes:

```javascript
const pointsToCreate = [];

// For each data field from your source
for (const field of dataFields) {
  const pointUuid = generatePointUuid(sensorId, field.name);
  
  pointsToCreate.push({
    uuid: pointUuid,                    // Required: UUID v5 generated ID
    name: `${sensorName} - ${field.name}`, // Required: Display name
    layer: "purpleair",                 // Required: Layer name
    protocol_id: `source-${sensorId}-${field.name}`, // Unique protocol identifier
    parent_name: sensorName,            // Parent equipment/device name
    display_units: field.units,         // Units for display (e.g., "°F", "%")
    attrs: {                            // Custom indexed attributes
      source: 'purpleair',
      sensor_index: sensorId.toString(),
      field_name: field.name,
      latitude: sensor.latitude?.toString() || '',
      longitude: sensor.longitude?.toString() || ''
    }
  });
}
```

**Important Notes:**
- **Don't include `class` in `attrs`** - This belongs in the model layer, not custom layers
- **Convert numbers to strings** in `attrs` for TAG fields
- **Use empty strings** for missing optional attributes

### 3. Upsert Points

POST the points array to create or update points:

```javascript
const pointsResponse = await sdk.http.post('/api/v1/point/points', {
  points: pointsToCreate
});

sdk.logEvent(`Created ${pointsToCreate.length} points`);
```

The API will:
- Create new points if they don't exist
- Update existing points if the UUID already exists
- Return the created/updated point data

---

## Point Structure

### Required Fields

| Field | Type | Description | Example |
|-------|------|-------------|---------|
| `uuid` | string | UUID v5 generated identifier | `"a1b2c3d4-..."` |
| `name` | string | Human-readable point name | `"Sensor 123 - temperature"` |
| `layer` | string | Layer name (must exist) | `"purpleair"` |

### Optional Top-Level Fields

| Field | Type | Description | Example |
|-------|------|-------------|---------|
| `protocol_id` | string | Protocol-specific identifier | `"purpleair-123-temp"` |
| `parent_name` | string | Parent equipment/device name | `"Purple Air Sensor 123"` |
| `display_units` | string | Units for display | `"°F"`, `"%"`, `"µg/m³"` |

### Custom Attributes (`attrs`)

Store searchable metadata in the `attrs` object:

```javascript
attrs: {
  source: 'purpleair',           // Data source identifier
  sensor_index: '281180',        // External sensor ID (as string)
  field_name: 'temperature',     // Field/channel name
  latitude: '37.7749',           // Location data (as string)
  longitude: '-122.4194',
  metadata: 'additional info'    // Any other custom data
}
```

---

## Adding Trending Data

After creating points, add time-series data for trending.

### Data Format

Each data request is for a **single point** and contains:

```javascript
{
  uuid: "point-uuid-here",        // Point UUID
  values: [                       // Array of timestamped values
    {
      ts: "2026-01-09T14:24:02.000Z",  // ISO 8601 timestamp
      real: 72.5                        // Value (use correct type)
    }
  ]
}
```

### Value Types

Choose the appropriate value type for your data:

| Type | Use For | Example |
|------|---------|---------|
| `real` | Floating point numbers | `{ real: 72.5 }` |
| `unsigned` | Non-negative integers | `{ unsigned: 100 }` |
| `signed` | Any integer | `{ signed: -10 }` |
| `boolean` | True/false | `{ boolean: true }` |
| `characterString` | Text | `{ characterString: "On" }` |
| `double` | High-precision floats | `{ double: 3.14159265359 }` |

### Posting Data

Make **one POST request per point**:

```javascript
const dataToAdd = [];

for (const field of dataFields) {
  dataToAdd.push({
    uuid: generatePointUuid(sensorId, field.name),
    values: [
      {
        ts: new Date(sensor.timestamp * 1000).toISOString(),
        real: field.value
      }
    ]
  });
}

// Post each point's data individually
for (const dataPoint of dataToAdd) {
  const response = await sdk.http.post('/api/v1/point/data', dataPoint);
  sdk.logEvent(`Added data for ${dataPoint.uuid}`);
}
```

**Why separate requests?**
- The API expects one point's data per request
- Allows granular error handling per point
- Follows the API schema structure

---

## Layer Configuration

### When to Create a Layer

Create a custom layer when:
- You're adding custom attributes to points
- You need to search/filter by those attributes
- You're integrating an external data source

### Layer Requirements

1. **Name** must match the `layer` field in your points
2. **Indexed** should be `true` to enable searching
3. **Structured components** define searchable attributes

### Don't Use Layers For

- Modifying standard Haystack tags (use model layer)
- Adding `class` attributes (belongs in model layer)
- Read-only integrations with existing points

---

## Complete Example

See [get-purpleair-data.js](../get-purpleair-data.js) for a complete working example that:

1. Fetches sensor data from Purple Air API
2. Generates consistent UUIDs using UUID v5
3. Creates points with metadata attributes
4. Trends current values for each sensor reading

### Key Patterns in the Example

```javascript
const { v5: uuidv5 } = require("uuid");
const NAMESPACE = "6ba7b810-9dad-11d1-80b4-00c04fd430c8";

// 1. Generate UUID
const pointUuid = uuidv5(`purpleair-${sensorId}-${fieldName}`, NAMESPACE);

// 2. Create point
const point = {
  uuid: pointUuid,
  name: `${sensorName} - ${fieldName}`,
  layer: "purpleair",
  protocol_id: `purpleair-${sensorId}-${fieldName}`,
  parent_name: sensorName,
  display_units: getUnits(fieldName),
  attrs: {
    source: 'purpleair',
    sensor_index: sensorId.toString(),
    field_name: fieldName
  }
};

// 3. Upsert points
await sdk.http.post('/api/v1/point/points', { points: [point] });

// 4. Add data
await sdk.http.post('/api/v1/point/data', {
  uuid: pointUuid,
  values: [{ ts: timestamp, real: value }]
});
```

---

## Best Practices

### UUID Generation

✅ **DO:**
- Use UUID v5 with a consistent namespace
- Include all identifying information in the name (sensor ID + field)
- Use the same UUID generation logic across all hook runs

❌ **DON'T:**
- Use random UUIDs (creates duplicates)
- Change UUID generation logic (orphans old points)
- Use UUID v4 or timestamp-based IDs

### Point Naming

✅ **DO:**
- Use descriptive names: `"Sensor 123 - Temperature"`
- Include parent device name for context
- Be consistent across similar points

❌ **DON'T:**
- Use just field names: `"temperature"`
- Include redundant info: `"Purple Air Purple Air Sensor 123 Temperature"`
- Use special characters that break queries

### Attributes

✅ **DO:**
- Store searchable metadata in `attrs`
- Convert numeric IDs to strings for TAG fields
- Use empty strings for missing optional data
- Index attributes you'll search on

❌ **DON'T:**
- Put `class` in custom layer attrs (use model layer)
- Store large objects or arrays in attrs
- Use inconsistent attribute names across points

### Data Trending

✅ **DO:**
- Use ISO 8601 timestamps
- Choose appropriate value types (`real`, `unsigned`, etc.)
- Post data immediately after creating points
- Handle missing values gracefully

❌ **DON'T:**
- Use Unix timestamps (convert to ISO 8601)
- Always use `real` (use correct type for your data)
- Batch multiple points in one data request
- Fail the entire hook if one point fails

### Error Handling

✅ **DO:**
- Log point creation success/failure
- Continue processing other points on failure
- Validate data before creating points
- Check API responses for errors

❌ **DON'T:**
- Silently fail without logging
- Stop all processing on first error
- Assume all API calls succeed
- Skip validation of external data

---

## Related Documentation

- [Hook Development Guide](./hooks-guide.md) - Overall hook concepts and patterns
- [Point API Guide](./point-api.md) - Querying and reading points
- [Equipment API Guide](./equipment-api.md) - Working with equipment
- [Purple Air Example](../get-purpleair-data.js) - Complete working implementation

---

## Support

For questions about creating points and trending data:
- Review the [Point API reference](https://buf.build/normalframework/nf/docs/main:normalgw.hpl.v1#normalgw.hpl.v1.PointManager)
- Check the [JavaScript SDK documentation](https://docs.normal.dev/js-sdk)
- Examine working examples in this repository
