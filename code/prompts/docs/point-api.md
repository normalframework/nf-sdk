# Point API Examples

This guide provides examples for working with the Point Management API (`normalgw.hpl.v1.PointManager`).

**Full API reference:** [normalgw.hpl.v1.PointManager](https://buf.build/normalframework/nf/docs/main:normalgw.hpl.v1#normalgw.hpl.v1.PointManager)

## Table of Contents

- [Simple Query](#simple-query)
- [Query with Filters](#query-with-filters)
- [Filter by Equipment (equipRef)](#filter-by-equipment-equipref)
- [Complex Queries (AND/OR Logic)](#complex-queries-andor-logic)
- [Pagination](#pagination)
- [Get Time Series Data (GetData)](#get-time-series-data-getdata)

---

## Simple Query

Get all points with pagination:

```javascript
async function getPoints() {
  const data = await fetchData('/api/v1/point/query', {
    pageSize: 100,
    responseFormat: 'LAYERS_COLLAPSED',
  });
  return data;
}
```

**Response format:**

```json
{
  "points": [
    {
      "uuid": "point-uuid-123",
      "name": "Temperature Sensor 1",
      "attrs": {
        "siteRef": "site-456",
        "equipRef": "equip-789"
      }
    }
  ],
  "totalCount": "1234"
}
```

---

## Query with Filters

Filter points by a specific property:

```javascript
async function queryPointsBySite(siteRef) {
  const data = await fetchData('/api/v1/point/query', {
    structuredQuery: {
      field: {
        property: 'siteRef',
        text: siteRef,
      },
    },
    pageSize: 100,
    responseFormat: 'LAYERS_COLLAPSED',
  });
  return data;
}
```

---

## Filter by Equipment (equipRef)

Query all points associated with a specific equipment. The `equipRef` property should be the equipment's display name or ID (not the UUID):

```javascript
async function getPointsForEquipment(equipName) {
  const data = await fetchData('/api/v1/point/query', {
    structuredQuery: {
      field: {
        property: 'equipRef',
        text: equipName,  // e.g., "Floor 23: South Return"
      },
    },
    pageSize: 100,
    responseFormat: 'LAYERS_COLLAPSED',
  });
  return data;
}
```

**Important:** Use the equipment's display name (e.g., `"Floor 23: South Return"`) for the `equipRef` filter, not the UUID. This matches how equipment is referenced in the Haystack data model.

---

## Complex Queries (AND/OR Logic)

Combine multiple filters with AND/OR logic:

```javascript
async function queryPointsAdvanced() {
  const data = await fetchData('/api/v1/point/query', {
    structuredQuery: {
      and: [
        { field: { property: 'siteRef', text: 'site-123' } },
        { field: { property: 'equipRef', text: 'equip-456' } },
      ],
    },
    pageSize: 100,
    responseFormat: 'LAYERS_COLLAPSED',
  });
  return data;
}
```

**OR logic example:**

```javascript
async function queryPointsWithOr() {
  const data = await fetchData('/api/v1/point/query', {
    structuredQuery: {
      or: [
        { field: { property: 'type', text: 'temperature' } },
        { field: { property: 'type', text: 'humidity' } },
      ],
    },
    pageSize: 100,
    responseFormat: 'LAYERS_COLLAPSED',
  });
  return data;
}
```

---

## Pagination

Retrieve all points across multiple pages:

```javascript
async function getAllPoints() {
  let allPoints = [];
  let pageOffset = 0;
  let hasMore = true;

  while (hasMore) {
    const result = await fetchData('/api/v1/point/query', {
      pageSize: 100,
      pageOffset: pageOffset,
      responseFormat: 'LAYERS_COLLAPSED',
    });

    allPoints = allPoints.concat(result.points);
    hasMore = result.nextPageOffset > 0;
    pageOffset = result.nextPageOffset;
  }

  return allPoints;
}
```

See the [Pagination Guide](../README.md#pagination) for more details on pagination patterns.

---

## Get Time Series Data (GetData)

Retrieve historical time series data for one or more points using the GetData endpoint. This is a **GET request** with query parameters.

**Full API reference:** [normalgw.hpl.v1.PointManager.GetData](https://buf.build/normalframework/nf/docs/main:normalgw.hpl.v1#normalgw.hpl.v1.PointManager.GetData)

### Basic Example

Fetch the last 24 hours of data for multiple points:

```javascript
async function getRecentData(pointUuids) {
  const now = new Date();
  const oneDayAgo = new Date(now.getTime() - 24 * 60 * 60 * 1000);
  
  const params = new URLSearchParams({
    from: oneDayAgo.toISOString(),
    to: now.toISOString(),
  });
  
  // Append each UUID as a separate 'uuids' parameter
  pointUuids.forEach(uuid => params.append('uuids', uuid));
  
  const response = await fetch(`/api/v1/point/data?${params.toString()}`, {
    headers: {
      'Authorization': `Bearer ${token}`,
    },
  });
  
  const data = await response.json();
  return data.data || [];  // Array of DataVector objects
}
```

### Request Parameters

- **`uuids`** (required): One or more point UUIDs. Repeat the parameter for each UUID (e.g., `?uuids=uuid1&uuids=uuid2`)
- **`from`** (required): Start time in ISO 8601 format (e.g., `2026-01-01T00:00:00Z`)
- **`to`** (required): End time in ISO 8601 format
- **`resampleInterval`** (optional): Resampling interval (not covered in this example)

### Response Format

```json
{
  "data": [
    {
      "uuid": "point-uuid-123",
      "values": [
        {
          "ts": "2026-01-02T10:00:00Z",
          "real": 72.5
        },
        {
          "ts": "2026-01-02T11:00:00Z",
          "real": 73.2
        }
      ]
    }
  ]
}
```

Each `DataVector` contains:
- **`uuid`**: Point UUID
- **`values`**: Array of timestamped values
  - **`ts`**: Timestamp in ISO 8601 format
  - Value field (one of): `boolean`, `unsigned`, `signed`, `real`, `double`, `characterString`, `null`

### Practical Usage: Sparklines

Generate sparkline visualizations from time series data:

```javascript
function generateSparkline(values) {
  const numericValues = values.map(v => {
    if (v.real !== undefined) return v.real;
    if (v.double !== undefined) return v.double;
    if (v.unsigned !== undefined) return v.unsigned;
    if (v.signed !== undefined) return v.signed;
    return null;
  }).filter(v => v !== null);
  
  if (numericValues.length === 0) return '';
  
  const min = Math.min(...numericValues);
  const max = Math.max(...numericValues);
  const range = max - min || 1;
  
  const points = numericValues.map((value, index) => {
    const x = (index / (numericValues.length - 1 || 1)) * 120;
    const y = 30 - ((value - min) / range) * 30;
    return `${x},${y}`;
  }).join(' ');
  
  return `<svg width="120" height="30">
    <polyline fill="none" stroke="#605AEA" stroke-width="1.5" points="${points}" />
  </svg>`;
}
```
