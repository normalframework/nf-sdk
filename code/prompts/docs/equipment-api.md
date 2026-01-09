# Equipment API Examples

This guide provides examples for working with the Equipment Management API.

**Full API reference:** [normalgw.hpl.v1.EquipManager](https://buf.build/normalframework/nf/docs/main:normalgw.hpl.v1#normalgw.hpl.v1.EquipManager)

## Table of Contents

- [Get Equipment Types](#get-equipment-types)
- [Get Equipment by Type](#get-equipment-by-type)
- [Get Equipment Counts](#get-equipment-counts)
- [Query Equipment (Alternative Method)](#query-equipment-alternative-method)

---

## Get Equipment Types

Retrieve all available equipment type definitions:

```javascript
async function getEquipmentTypes() {
  // Note: Must include pageSize parameter to retrieve actual type data
  const data = await apiCall('/api/v1/equipment/types?pageSize=100');
  return data.equipmentTypes;
}
```

**Response format:**

```json
{
  "equipmentTypes": [
    {
      "id": "3ef88548-4048-49ab-90f7-e47effad8a0f",
      "name": "iaqSensor (1)",
      "className": "iaqSensor",
      "description": "A device containing a sensor which measures IAQ",
      "markers": [
        {
          "name": "iaqSensor",
          "description": "",
          "ontologyRequires": true,
          "typeRequires": false,
          "options": [],
          "selectedOption": ""
        }
      ],
      "points": []
    }
  ],
  "totalCount": "7"
}
```

**Example usage:**

```javascript
// Get equipment types and display them
getEquipmentTypes().then((types) => {
  console.log('Available equipment types:', types);
  types.forEach((type) => {
    console.log(`${type.name} (${type.className}): ${type.description}`);
  });
});
```

---

## Get Equipment by Type

Retrieve equipment instances for a specific type:

```javascript
async function getEquipmentByType(typeId) {
  const data = await apiCall(
    `/api/v1/equipment/by-type/${typeId}?pageSize=100&pageOffset=0`
  );
  return {
    equipment: data.equips || [],
    totalCount: parseInt(data.totalCount || '0', 10),
  };
}
```

**Response format:**

```json
{
  "equips": [
    {
      "id": "Floor 23: South Return",
      "point": {
        "uuid": "0b20569d-2868-385f-bbe3-251678d72f15",
        "name": "Floor 23: South Return",
        "layer": "model",
        "attrs": {
          "markers": "iaqSensor,equip",
          "id": "Floor 23: South Return",
          "equipTypeId": "3ef88548-4048-49ab-90f7-e47effad8a0f",
          "class": "iaqSensor"
        },
        "pointType": "EQUIPMENT"
      }
    }
  ],
  "totalCount": "3"
}
```

**Note:** Equipment objects have a nested structure:
- `id` - The equipment identifier/name
- `point` - Nested object containing `uuid`, `name`, `attrs`, etc.

---

## Get Equipment Counts

Get counts of equipment for all types:

```javascript
async function getEquipmentCounts() {
  // First, get all equipment types
  const typesData = await apiCall('/api/v1/equipment/types?pageSize=100');
  const types = typesData.equipmentTypes || [];

  // Then get count for each type
  const countsPromises = types.map(async (type) => {
    const data = await apiCall(
      `/api/v1/equipment/by-type/${type.id}?pageSize=100&pageOffset=0`
    );
    return {
      id: type.id,
      name: type.name,
      className: type.className,
      count: parseInt(data.totalCount || '0', 10),
    };
  });

  const counts = await Promise.all(countsPromises);

  // Filter to only types with equipment
  return counts.filter((c) => c.count > 0);
}

// Usage
getEquipmentCounts().then((counts) => {
  console.log('Equipment by type:');
  counts.forEach((item) => {
    console.log(`${item.className}: ${item.count}`);
  });
});
```

---

## Query Equipment (Alternative Method)

You can also query equipment using the POST endpoint with structured queries:

### Get All Equipment

```javascript
async function getEquips() {
  const data = await fetchData('/api/v1/equip/getEquips', {
    pageSize: 100,
    responseFormat: 'LAYERS_COLLAPSED',
  });
  return data;
}
```

### Query Equipment by Site

```javascript
async function getEquipsBySite(siteRef) {
  const data = await fetchData('/api/v1/equip/getEquips', {
    structuredQuery: {
      field: {
        property: 'siteRef',
        text: siteRef,
      },
    },
    pageSize: 100,
    responseFormat: 'LAYERS_COLLAPSED',
  });
  return data.equips;
}
```

### Pagination

Retrieve all equipment across multiple pages:

```javascript
async function getAllEquips() {
  let allEquips = [];
  let pageOffset = 0;
  let hasMore = true;

  while (hasMore) {
    const result = await fetchData('/api/v1/equip/getEquips', {
      pageSize: 100,
      pageOffset: pageOffset,
      responseFormat: 'LAYERS_COLLAPSED',
    });

    allEquips = allEquips.concat(result.equips);
    hasMore = result.nextPageOffset > 0;
    pageOffset = result.nextPageOffset;
  }

  return allEquips;
}
```

See the [Pagination Guide](../README.md#pagination) for more details on pagination patterns.
