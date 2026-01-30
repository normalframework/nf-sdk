# Hook Development Guide

This guide explains how to develop hooks (automation functions) for the Normal Framework. Hooks allow you to write custom automation, control algorithms, and data drivers that integrate with building systems.

## Table of Contents

- [Overview](#overview)
- [Hook Structure](#hook-structure)
- [Use Cases](#use-cases)
  - [Writing a Data Driver](#writing-a-data-driver)
  - [Implementing a Control Algorithm](#implementing-a-control-algorithm)
- [Hook Configuration](#hook-configuration)
- [API Reference](#api-reference)

---

## Overview

Hooks are JavaScript functions that run on the Normal Framework to automate building operations. They can:

- Read and write point values (BACnet, Modbus, etc.)
- Fetch data from external REST APIs
- Implement control algorithms
- Trigger based on schedules or on-demand
- Maintain state across executions using variables

**Full JavaScript SDK Reference:** [docs.normal.dev/js-sdk](https://docs.normal.dev/js-sdk)

---

## Hook Structure

Every hook consists of two files:

1. **Configuration file** (JSON) - Defines point queries, grouping, run mode, and variables
2. **Implementation file** (JavaScript) - Contains the automation logic

### Basic Hook Template

**example-hook.json** (Configuration):

```json
{
  "name": "example-hook",
  "entryPoint": "/example-hook.js",
  "points": {
    "query": {
      "field": {
        "property": "class",
        "text": "sensor"
      }
    },
    "labelAttribute": "name"
  },
  "mode": "MODE_ON_SCHEDULE",
  "schedule": {
    "interval": "5m"
  }
}
```

**example-hook.js** (Implementation):

```javascript
const NormalSdk = require("@normalframework/applications-sdk");

/**
 * Hook function
 * @param {NormalSdk.InvokeParams} params
 * @returns {NormalSdk.InvokeResult}
 */
module.exports = async ({ points, sdk, groupVariables, globalVariables, config, args }) => {
  // Your automation logic here
  sdk.logEvent(`Processing ${points.length} points`);
  
  // Access points
  for (const point of points) {
    sdk.logEvent(`Point: ${point.name}, Value: ${point.latestValue?.value}`);
  }
};
```

---

## Use Cases

### Writing a Data Driver

Data drivers fetch data from external APIs and write values to points in the Normal Framework. This is useful for integrating third-party systems, weather services, energy meters, or any REST API.

#### Example: Weather API Driver

This example fetches weather data from an external API and writes it to points.

**Step 1: Create the Hook Configuration**

**weather-driver.json**:

```json
{
  "name": "weather-driver",
  "entryPoint": "/weather-driver.js",
  "points": {
    "query": {
      "and": [
        {
          "field": {
            "property": "class",
            "text": "weather"
          }
        },
        {
          "field": {
            "property": "source",
            "text": "external-api"
          }
        }
      ]
    },
    "groups": {
      "keys": ["siteRef"]
    },
    "labelAttribute": "name"
  },
  "mode": "MODE_ON_SCHEDULE",
  "schedule": {
    "interval": "15m"
  },
  "config": {
    "apiUrl": "https://api.weather.example.com/v1",
    "apiKey": "YOUR_API_KEY"
  }
}
```

**Step 2: Implement the Data Driver**

**weather-driver.js**:

```javascript
const NormalSdk = require("@normalframework/applications-sdk");

/**
 * Weather Data Driver
 * Fetches weather data from external API and writes to points
 * 
 * @param {NormalSdk.InvokeParams} params
 * @returns {NormalSdk.InvokeResult}
 */
module.exports = async ({ points, sdk, config, groupVariables }) => {
  const apiUrl = config["apiUrl"];
  const apiKey = config["apiKey"];
  
  sdk.logEvent(`Weather driver started for ${points.length} points`);
  
  // Find specific points by label
  const tempPoint = points.byLabel("outside-air-temp").first();
  const humidityPoint = points.byLabel("outside-air-humidity").first();
  const pressurePoint = points.byLabel("barometric-pressure").first();
  
  if (!tempPoint) {
    sdk.logEvent("No temperature point found, skipping");
    return;
  }
  
  try {
    // Fetch data from external API
    const response = await sdk.http.get(`${apiUrl}/current`, {
      params: {
        api_key: apiKey,
        location: sdk.groupKey, // Use group key as location identifier
      },
    });
    
    const weatherData = response.data;
    
    sdk.logEvent(`Received weather data: ${JSON.stringify(weatherData)}`);
    
    // Write temperature to point
    if (tempPoint && weatherData.temperature !== undefined) {
      const [success, error] = await tempPoint.write({ 
        real: weatherData.temperature 
      });
      
      if (success) {
        sdk.logEvent(`Wrote temperature: ${weatherData.temperature}°F`);
      } else {
        sdk.logEvent(`Failed to write temperature: ${error}`);
      }
    }
    
    // Write humidity to point
    if (humidityPoint && weatherData.humidity !== undefined) {
      const [success, error] = await humidityPoint.write({ 
        real: weatherData.humidity 
      });
      
      if (success) {
        sdk.logEvent(`Wrote humidity: ${weatherData.humidity}%`);
      } else {
        sdk.logEvent(`Failed to write humidity: ${error}`);
      }
    }
    
    // Write pressure to point
    if (pressurePoint && weatherData.pressure !== undefined) {
      const [success, error] = await pressurePoint.write({ 
        real: weatherData.pressure 
      });
      
      if (success) {
        sdk.logEvent(`Wrote pressure: ${weatherData.pressure} inHg`);
      } else {
        sdk.logEvent(`Failed to write pressure: ${error}`);
      }
    }
    
  } catch (error) {
    sdk.logEvent(`Error fetching weather data: ${error.message}`);
  }
};
```

#### Key Concepts for Data Drivers

1. **External API Calls**: Use `sdk.http` (Axios client) for HTTP requests
2. **Point Selection**: Use `points.byLabel()` to find specific points
3. **Writing Values**: Use `point.write()` to update point values
4. **Error Handling**: Always wrap API calls in try/catch
5. **Logging**: Use `sdk.logEvent()` for debugging and monitoring
6. **Grouping**: Use groups to organize points by site/location
7. **Configuration**: Store API keys and URLs in the `config` object

#### REST API Driver Workflow

```
┌─────────────────┐
│  Hook Triggered │  (On Schedule: every 15 minutes)
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Select Points  │  (Query points with class=weather, source=external-api)
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│   Group Points  │  (Group by siteRef - one API call per site)
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Fetch API Data │  (sdk.http.get() to external weather API)
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Parse Response │  (Extract temperature, humidity, pressure)
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Write to Points│  (point.write() for each value)
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│   Log Results   │  (sdk.logEvent() for monitoring)
└─────────────────┘
```

#### Testing Your Data Driver

1. **Configure points**: Create points with appropriate labels and tags
2. **Set config values**: Add API URL and key to hook configuration
3. **Run hook manually**: Use "MODE_ON_REQUEST" during development
4. **Check logs**: View console output via `sdk.logEvent()`
5. **Verify writes**: Confirm point values are updated
6. **Switch to schedule**: Change to "MODE_ON_SCHEDULE" for production

#### Advanced: Error Recovery and Retries

```javascript
module.exports = async ({ points, sdk, config, globalVariables }) => {
  const maxRetries = 3;
  const retryDelay = 5000; // 5 seconds
  
  // Get retry count from global variable
  let retryCount = globalVariables.byLabel("retry-count");
  if (!retryCount) {
    sdk.logEvent("No retry count variable found");
    retryCount = { latestValue: { value: 0 } };
  }
  
  const currentRetries = Number(retryCount.latestValue?.value || 0);
  
  try {
    // Attempt API call
    const response = await sdk.http.get(config["apiUrl"]);
    
    // Success - reset retry count
    if (retryCount.write) {
      await retryCount.write({ unsigned: 0 });
    }
    
    // Process data...
    
  } catch (error) {
    sdk.logEvent(`API call failed: ${error.message}`);
    
    if (currentRetries < maxRetries) {
      // Increment retry count
      if (retryCount.write) {
        await retryCount.write({ unsigned: currentRetries + 1 });
      }
      
      sdk.logEvent(`Will retry (${currentRetries + 1}/${maxRetries})`);
      
      // Wait and retry
      await sdk.sleep(retryDelay);
      // Note: In production, you'd want to re-invoke the hook or use exponential backoff
    } else {
      sdk.logEvent(`Max retries reached, giving up`);
      // Reset counter for next interval
      if (retryCount.write) {
        await retryCount.write({ unsigned: 0 });
      }
    }
  }
};
```

---

### Implementing a Control Algorithm

*(Coming soon)*

Control algorithms use point data to make decisions and adjust equipment setpoints. Examples include:

- VAV damper control
- Temperature reset strategies
- Demand response
- Optimal start/stop
- Economizer control

---

## Hook Configuration

### Configuration File Structure

```json
{
  "name": "string",              // Hook name
  "entryPoint": "string",        // Path to JS file (e.g., "/my-hook.js")
  "points": {
    "query": {},                 // Point query (structured query)
    "groups": {                  // Optional: group points
      "keys": ["string"]         // Group by these attributes
    },
    "labelAttribute": "string",  // Attribute to use for labels
    "equipmentTypeId": "string"  // Optional: filter by equipment type
  },
  "mode": "string",              // Run mode (see below)
  "schedule": {                  // For MODE_ON_SCHEDULE
    "interval": "string"         // e.g., "5m", "1h", "1d"
  },
  "config": {},                  // Custom config key-value pairs
  "variables": {                 // State variables
    "group": [],                 // Group-scoped variables
    "global": []                 // Hook-scoped variables
  }
}
```

### Run Modes

- **`MODE_ON_SCHEDULE`**: Runs on a fixed interval (e.g., every 5 minutes)
- **`MODE_ON_REQUEST`**: Manual invocation via API (useful for testing)
- **`MODE_ON_CHANGE`**: Runs when point values change
- **`MODE_ON_EVENT`**: Runs on specific events

### Point Query Structure

Point queries use structured query syntax:

```json
{
  "and": [
    {
      "field": {
        "property": "class",
        "text": "sensor"
      }
    },
    {
      "field": {
        "property": "siteRef",
        "text": "Building-A"
      }
    }
  ]
}
```

**Query operators:**
- `and`: All conditions must match
- `or`: Any condition must match
- `not`: Negates a condition
- `field`: Match a property value

---

## API Reference

### Hook Function Parameters

```javascript
module.exports = async ({
  points,           // ResultSet of selected points
  sdk,             // Normal SDK instance
  groupVariables,  // Group-scoped variables
  globalVariables, // Hook-scoped variables
  config,          // Configuration object
  args,            // Arguments (for on-request mode)
}) => {
  // Your code here
};
```

### points (ResultSet)

Query and access points:

```javascript
// Array methods
points.length
points.forEach(point => { })
points.map(point => { })
points.filter(point => { })

// Selection methods
points.byLabel("label-name")      // Find by label
points.where(p => p.attrs.x === y) // Filter by condition
points.first()                     // Get first point
```

### Point Object

```javascript
point.uuid          // Point UUID
point.name          // Point name
point.attrs         // Attributes object
point.latestValue   // Current value
point.latestValue.value  // Value as number
point.latestValue.ts     // Timestamp

// Methods
await point.read()              // Read current value
await point.write({ real: 72 }) // Write value
point.isChanged()               // Check if changed since last run
await point.trueFor("5m", v => v.value > 70) // Check condition duration
```

### sdk Object

```javascript
sdk.groupKey       // Current group key
sdk.logEvent(msg)  // Write to event log
sdk.http           // Axios HTTP client
sdk.sleep(ms)      // Pause execution
```

### HTTP Client (sdk.http)

```javascript
// GET request
const response = await sdk.http.get("/api/endpoint");

// POST request
const response = await sdk.http.post("/api/endpoint", {
  key: "value"
});

// With query parameters
const response = await sdk.http.get("/api/endpoint", {
  params: { key: "value" }
});

// Access response
response.data       // Response body
response.status     // HTTP status code
response.headers    // Response headers
```

### Variables

Variables persist state across hook executions:

```javascript
// Group variables (one per group)
const damperLoop = groupVariables.byLabel("damper-loop");
damperLoop.latestValue.value  // Read value
await damperLoop.write({ real: 50 }) // Write value

// Global variables (shared across all groups)
const minSetpoint = globalVariables.byLabel("min-setpoint");
minSetpoint.latestValue.value  // Read value
await minSetpoint.write({ real: 65 }) // Write value
```

---

## Best Practices

1. **Always handle errors**: Wrap API calls in try/catch blocks
2. **Log meaningful events**: Use `sdk.logEvent()` for debugging
3. **Check point existence**: Verify points exist before accessing
4. **Use groups effectively**: Group points by site/equipment for parallel processing
5. **Store secrets in config**: Never hardcode API keys in code
6. **Test with MODE_ON_REQUEST**: Use manual mode during development
7. **Handle rate limits**: Add delays or backoff for external APIs
8. **Monitor execution time**: Keep hooks fast to avoid blocking
9. **Use variables for state**: Persist data between executions
10. **Document your code**: Add comments explaining logic

---

## Examples

See the following files for working examples:

- [test.js](../test.js) - Basic hook example
- [example.js](../example.js) - Hook template
- [bacnet-write.js](../bacnet-write.js) - BACnet write example

---

## Additional Resources

- **Programming Guide**: [docs.normal.dev/applications/programming-guide](https://docs.normal.dev/applications/programming-guide/)
- **Hooks Overview**: [docs.normal.dev/applications/hooks](https://docs.normal.dev/applications/hooks/)
- **JavaScript SDK**: [docs.normal.dev/js-sdk](https://docs.normal.dev/js-sdk/)
- **Point API**: [Point API Guide](./point-api.md)
- **Equipment API**: [Equipment API Guide](./equipment-api.md)

---

## Support

For questions about hook development, consult the official Normal Framework documentation or reach out to your development team.
