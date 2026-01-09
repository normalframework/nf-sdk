# Plugin Development Guide

This guide explains how to develop plugins for the admin console. Plugins are served in iframes and must follow specific patterns for authentication, static file serving, and API communication.

## Table of Contents

- [Static File Serving](#static-file-serving)
- [UI Framework and Styling](#ui-framework-and-styling)
- [Authentication](#authentication)
- [Backend API Communication](#backend-api-communication)
- [REST API Reference](#rest-api-reference)

## Hook Development Documentation

For developing hooks (automation functions and data integrations):

- **[Hook Development Guide](docs/hooks-guide.md)** - Overview of hooks, configuration, and basic patterns
- **[Creating Points and Trending Data](docs/creating-points-guide.md)** - How to dynamically create points from external APIs and trend data
- **[Point API Guide](docs/point-api.md)** - Querying and reading point data
- **[Equipment API Guide](docs/equipment-api.md)** - Working with equipment

**Example Hooks:**
- [get-purpleair-data.js](get-purpleair-data.js) - Complete example of creating points and trending data from Purple Air sensor API

---

## Static File Serving

### No-Build Workflow (Recommended)

Since plugins are often developed in remote filesystems (e.g., via VSCode remote plugins), the recommended approach is to **avoid build tools entirely**. Write vanilla JavaScript, HTML, and CSS that can be edited and saved directly.

**Key Principles:**

- Write vanilla JavaScript (ES6+ modules supported)
- Load all frameworks from CDN
- No compilation or bundling required
- Edit files directly in VSCode and save to `/static`

### Directory Structure

All static files must be placed in a `/static` folder at the root of your plugin:

```
my-plugin/
├── static/
│   ├── index.html       # Entry point
│   ├── app.js          # Your application logic (vanilla JS)
│   ├── styles.css      # Your custom styles
│   ├── utils.js        # Helper functions (optional)
│   └── assets/
│       └── images/
└── README.md
```

### Development Workflow

1. Create the `/static` folder
2. Add your `index.html` file
3. Write vanilla JavaScript in separate `.js` files
4. Load dependencies from CDN in your HTML
5. Save and test - no build step required!

### Optional: Build Tools (Advanced)

If you prefer to use TypeScript or JSX and have local development access, you can use build tools, but output everything to `/static`:

**Vite Example:**

```javascript
// vite.config.js
export default {
  build: {
    outDir: 'static',
    rollupOptions: {
      external: ['react', 'react-dom'], // Load from CDN
    },
  },
};
```

**Note:** This is optional and not required for most plugins.

---

## UI Framework and Styling

### Recommended UI Framework

For visual consistency, follow the **shadcn/ui** design patterns. Since we're using a no-build workflow, you have two options:

**Option 1: Vanilla CSS/Tailwind (Recommended for no-build)**

- Use Tailwind CSS from CDN
- Manually replicate shadcn/ui component styles using our design tokens
- Copy component HTML structures from shadcn examples

**Option 2: Pre-built Component Libraries (CDN-based)**

- Use libraries like DaisyUI or similar that work with Tailwind CDN
- Customize to match our color palette

**Option 3: Build workflow (if you have local npm access)**

- Install shadcn/ui components: `npx shadcn-ui@latest add [component]`
- Build and output to `/static`

### House Style Guide

To emulate the admin console's design:

#### Color Palette

Use these CSS custom properties in your styles:

```css
:root {
  /* Primary Colors */
  --primary: 237.58 100% 71%; /* Primary purple */
  --primary-foreground: 0 0% 100%;
  --secondary: 188 40% 60%; /* Secondary teal */
  --secondary-foreground: 0 0% 100%;

  /* Neutral Colors */
  --background: 0 0% 100%;
  --foreground: 222.2 84% 4.9%;
  --border: 214.3 31.8% 91.4%;
  --input: 0 0% 80%;

  /* Semantic Colors */
  --success: #287d3c;
  --success-bg: #edf9f0;
  --warning: #eeb102;
  --warning-bg: #fff4ec;
  --error: #da1414;
  --error-bg: #feefef;
  --info: #2e5aac;
  --info-bg: #eef2fa;

  /* Additional Brand Colors */
  --primary-light-purple: #605aea;
  --primary-dark-purple: #20144c;
  --accent-secondary: #abdfe7;
}
```

#### Typography

Use the system font stack or load from CDN:

```html
<!-- In your index.html head -->
<link rel="preconnect" href="https://fonts.googleapis.com" />
<link
  href="https://fonts.googleapis.com/css2?family=Noto+Sans:wght@400;500;600;700&family=Noto+Sans+Mono&display=swap"
  rel="stylesheet"
/>
```

```css
body {
  font-family: 'Noto Sans', sans-serif;
}

code,
pre {
  font-family: 'Noto Sans Mono', monospace;
}
```

#### Component Styling

Use Tailwind CSS (loaded from CDN) with our custom configuration:

```html
<!-- In your index.html head -->
<script src="https://cdn.tailwindcss.com"></script>
<script>
  tailwind.config = {
    theme: {
      extend: {
        colors: {
          'primary-light-purple': '#605AEA',
          'primary-dark-purple': '#20144C',
          'accent-secondary': '#ABDFE7',
        },
      },
    },
  };
</script>
```

### Loading Frameworks from CDN

**Example HTML structure:**

```html
<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>My Plugin</title>

    <!-- Tailwind CSS -->
    <script src="https://cdn.tailwindcss.com"></script>

    <!-- Fonts -->
    <link
      href="https://fonts.googleapis.com/css2?family=Noto+Sans:wght@400;500;600;700&display=swap"
      rel="stylesheet"
    />

    <!-- React (if needed) -->
    <script
      crossorigin
      src="https://unpkg.com/react@18/umd/react.production.min.js"
    ></script>
    <script
      crossorigin
      src="https://unpkg.com/react-dom@18/umd/react-dom.production.min.js"
    ></script>

    <!-- Your bundled styles -->
    <link rel="stylesheet" href="./styles.css" />
  </head>
  <body>
    <div id="root"></div>

    <!-- Your bundled script -->
    <script src="./bundle.js"></script>
  </body>
</html>
```

---

## Authentication

### Token Retrieval from Window Location

Since plugins are served in an iframe, the parent application may pass an authorization token via the URL. Your plugin must:

1. Check the window location for an auth token
2. Store it for API requests
3. Handle token expiration

**JavaScript Example:**

```javascript
// Extract token from URL query parameters or hash
function getAuthToken() {
  const params = new URLSearchParams(window.location.search);
  const token = params.get('token') || params.get('auth_token');

  // Alternatively, check the hash
  if (!token && window.location.hash) {
    const hashParams = new URLSearchParams(window.location.hash.substring(1));
    return hashParams.get('token') || hashParams.get('auth_token');
  }

  return token;
}

// Store token for use in API calls
const authToken = getAuthToken();
```

### Using Token in API Requests

Include the token in the `Authorization` header for all API requests:

```javascript
async function makeAuthenticatedRequest(url, options = {}) {
  const token = getAuthToken();

  const headers = {
    'Content-Type': 'application/json',
    ...options.headers,
  };

  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }

  const response = await fetch(url, {
    ...options,
    headers,
  });

  // Handle permission denied - trigger reload to get new token
  if (response.status === 401 || response.status === 403) {
    console.warn('Authentication failed, reloading to obtain new token...');
    window.location.reload();
    return null;
  }

  return response;
}
```

### Handling Token Expiration

If you receive a `401 Unauthorized` or `403 Forbidden` response:

1. Trigger a page reload to obtain a fresh token
2. The parent application will re-inject the new token on reload

```javascript
function handleAuthError(response) {
  if (response.status === 401 || response.status === 403) {
    // Clear any cached data
    sessionStorage.clear();

    // Reload to get new token from parent
    window.location.reload();
  }
}
```

---

## Backend API Communication

### Using Browser Origin

When making API calls from a plugin, always use the **browser's origin** (not a hardcoded URL):

```javascript
// Correct: Use browser origin
const apiUrl = `${window.location.origin}/api/v1/point/points`;

// Incorrect: Hardcoded URL
// const apiUrl = 'http://localhost:8080/api/v1/point/points';
```

This ensures your plugin works across different deployment environments.

### Complete API Request Example

```javascript
async function fetchData(endpoint, body = null) {
  const token = getAuthToken();
  const url = `${window.location.origin}${endpoint}`;

  const options = {
    method: body ? 'POST' : 'GET',
    headers: {
      'Content-Type': 'application/json',
    },
  };

  if (token) {
    options.headers['Authorization'] = `Bearer ${token}`;
  }

  if (body) {
    options.body = JSON.stringify(body);
  }

  try {
    const response = await fetch(url, options);

    // Handle auth errors
    if (response.status === 401 || response.status === 403) {
      console.warn('Permission denied, reloading...');
      window.location.reload();
      return null;
    }

    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`);
    }

    return await response.json();
  } catch (error) {
    console.error('API request failed:', error);
    throw error;
  }
}
```

---

## Pagination

Most API endpoints that return lists support pagination to handle large datasets efficiently. Understanding the pagination pattern is essential for working with the API.

### Pagination Parameters

Paginated endpoints accept these parameters:

- **`pageSize`** (required): Number of items to return per page (typically 100)
- **`pageOffset`** (optional): Starting offset for the page (default: 0)

### Pagination Response

Paginated responses include:

- **`totalCount`**: Total number of items available (as a string)
- **`nextPageOffset`**: Offset for the next page (0 if no more pages)
- **Result array**: The actual data (e.g., `points`, `equips`, `equipmentTypes`)

### Pagination Pattern

**Single page request:**

```javascript
const data = await apiCall('/api/v1/equipment/types?pageSize=100');
console.log(`Total: ${data.totalCount}, Retrieved: ${data.equipmentTypes.length}`);
```

**Multiple pages (loop until done):**

```javascript
async function getAllItems() {
  let allItems = [];
  let pageOffset = 0;
  let hasMore = true;

  while (hasMore) {
    const result = await fetchData('/api/v1/point/query', {
      pageSize: 100,
      pageOffset: pageOffset,
      responseFormat: 'LAYERS_COLLAPSED',
    });

    allItems = allItems.concat(result.points);
    hasMore = result.nextPageOffset > 0;
    pageOffset = result.nextPageOffset;
  }

  return allItems;
}
```

### REST vs POST Pagination

**GET endpoints** (REST style):
```javascript
// Use query parameters
const data = await apiCall('/api/v1/equipment/types?pageSize=100&pageOffset=0');
```

**POST endpoints** (gRPC style):
```javascript
// Use request body
const data = await fetchData('/api/v1/point/query', {
  pageSize: 100,
  pageOffset: 0,
  responseFormat: 'LAYERS_COLLAPSED',
});
```

### Best Practices

1. **Always specify `pageSize`** - Some endpoints require it to return data
2. **Use reasonable page sizes** - 100 is a good default; adjust based on needs
3. **Check `nextPageOffset`** - A value of 0 means no more pages
4. **Handle partial results** - The last page may have fewer items than `pageSize`
5. **Parse `totalCount` as integer** - It's returned as a string: `parseInt(totalCount, 10)`

---

## REST API Reference

The admin console exposes gRPC services via REST endpoints using the gRPC-Web protocol. All request and response formats follow the Protocol Buffer definitions.

### API Documentation

**Full API reference:** [buf.build/normalframework/nf](https://buf.build/normalframework/nf)

All endpoints follow the pattern:

```
/api/v1/{service}/{method}
```

### Quick Reference Guide

Use this table to quickly find documentation for common tasks:

| Task | Documentation | Key Endpoints |
|------|---------------|---------------|
| Query and filter points | [Point API Guide](docs/point-api.md) | `POST /api/v1/point/query` |
| Get equipment types and instances | [Equipment API Guide](docs/equipment-api.md) | `GET /api/v1/equipment/types`<br>`GET /api/v1/equipment/by-type/{id}` |
| Understand pagination | [Pagination Section](#pagination) | All paginated endpoints |
| Authentication setup | [Authentication Section](#authentication) | - |
| API request patterns | [Backend API Communication](#backend-api-communication) | - |
| House styling guide | [UI Framework and Styling](#ui-framework-and-styling) | - |

### Key APIs for Plugin Development

#### Point Management (`normalgw.hpl.v1`)

**Most commonly used endpoints:**

- **Get Points**: `POST /api/v1/point/query`
  - Use for querying and filtering point data
  - See [GetPointsRequest](https://buf.build/normalframework/nf/docs/main:normalgw.hpl.v1#normalgw.hpl.v1.GetPointsRequest)
- **Get Points by ID**: `POST /api/v1/point/getPointsById`
  - Retrieve specific points by UUID
  - See [GetPointsByIdRequest](https://buf.build/normalframework/nf/docs/main:normalgw.hpl.v1#normalgw.hpl.v1.GetPointsByIdRequest)
- **Update Points**: `POST /api/v1/point/updatePoints`
  - Modify point data
  - See [UpdatePointsRequest](https://buf.build/normalframework/nf/docs/main:normalgw.hpl.v1#normalgw.hpl.v1.UpdatePointsRequest)

**📖 See detailed examples:** [Point API Guide](docs/point-api.md)

**Browse the full Point API:**  
[normalgw.hpl.v1.PointManager](https://buf.build/normalframework/nf/docs/main:normalgw.hpl.v1#normalgw.hpl.v1.PointManager)

#### Equipment Management (`normalgw.hpl.v1`)

**Most commonly used endpoints:**

- **Get Equipment Types**: `GET /api/v1/equipment/types`
  - Retrieve available equipment type definitions
  - Requires `pageSize` parameter
- **Get Equipment by Type**: `GET /api/v1/equipment/by-type/{id}`
  - Retrieve equipment instances for a specific type
  - Requires `pageSize` and `pageOffset` parameters
- **Get Equips**: `POST /api/v1/equip/getEquips`  
  - Query equipment instances with structured queries
  - See [GetEquipsRequest](https://buf.build/normalframework/nf/docs/main:normalgw.hpl.v1#normalgw.hpl.v1.GetEquipsRequest)
- **Update Equips**: `POST /api/v1/equip/updateEquips`
  - Modify equipment data
  - See [UpdateEquipsRequest](https://buf.build/normalframework/nf/docs/main:normalgw.hpl.v1#normalgw.hpl.v1.UpdateEquipsRequest)

**📖 See detailed examples:** [Equipment API Guide](docs/equipment-api.md)

**Browse the full Equipment API:**  
[normalgw.hpl.v1.EquipManager](https://buf.build/normalframework/nf/docs/main:normalgw.hpl.v1#normalgw.hpl.v1.EquipManager)

#### Other Useful Services

- **Time Series Data**: `normalgw.hpl.v1.TimeSeriesManager`
- **Platform Config**: `normalgw.platform.v1.ConfigManager`
- **Sites Management**: `normalgw.hpl.v1.SitesManager`

### Understanding Request/Response Formats

The REST API uses JSON encoding of Protocol Buffer messages. To understand the exact structure:

1. **Browse the API docs** at [buf.build/normalframework/nf](https://buf.build/normalframework/nf)
2. **Find your message type** (e.g., `GetPointsRequest`)
3. **Use the JSON representation** of the protobuf message in your requests

AI agents can read and interpret these protobuf definitions to help you construct proper requests

````

---

## Complete Plugin Example

Here's a minimal working plugin using **vanilla JavaScript** (no build required) that demonstrates all the concepts:

**static/index.html:**

```html
<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>Example Plugin</title>

    <!-- Tailwind CSS from CDN -->
    <script src="https://cdn.tailwindcss.com"></script>

    <!-- Configure Tailwind with our brand colors -->
    <script>
      tailwind.config = {
        theme: {
          extend: {
            colors: {
              'primary-light-purple': '#605AEA',
              'primary-dark-purple': '#20144C',
              'accent-secondary': '#ABDFE7',
            },
          },
        },
      };
    </script>

    <!-- Google Fonts -->
    <link
      href="https://fonts.googleapis.com/css2?family=Noto+Sans:wght@400;600&display=swap"
      rel="stylesheet"
    />

    <!-- Custom styles -->
    <link rel="stylesheet" href="./styles.css" />
  </head>
  <body class="bg-gray-50 p-6">
    <div id="app">
      <h1 class="text-2xl font-semibold text-primary-dark-purple mb-4">
        Point Explorer Plugin
      </h1>
      <button
        id="loadBtn"
        class="bg-primary-light-purple text-white px-4 py-2 rounded hover:opacity-90 transition"
      >
        Load Points
      </button>
      <div id="results" class="mt-4"></div>
    </div>

    <!-- Your vanilla JavaScript - no bundling needed! -->
    <script type="module" src="./app.js"></script>
  </body>
</html>
````

**static/styles.css:**

```css
body {
  font-family: 'Noto Sans', sans-serif;
}

/* Custom CSS variables matching house style */
:root {
  --primary: 237.58 100% 71%;
  --primary-foreground: 0 0% 100%;
  --success: #287d3c;
  --success-bg: #edf9f0;
  --error: #da1414;
  --error-bg: #feefef;
}

.success-message {
  background-color: var(--success-bg);
  color: var(--success);
  padding: 1rem;
  border-radius: 0.5rem;
  border-left: 4px solid var(--success);
}

.error-message {
  background-color: var(--error-bg);
  color: var(--error);
  padding: 1rem;
  border-radius: 0.5rem;
  border-left: 4px solid var(--error);
}
```

**static/app.js:**

```javascript
// Import utilities (optional - can be in same file)
import { apiCall } from './utils.js';

// Load points on button click
async function loadPoints() {
  const resultsDiv = document.getElementById('results');
  resultsDiv.innerHTML = '<p class="text-gray-600">Loading...</p>';

  try {
    const data = await apiCall('/api/v1/point/query', {
      pageSize: 10,
      responseFormat: 'LAYERS_COLLAPSED',
    });

    resultsDiv.innerHTML = `
      <div class="success-message">
        <h2 class="font-semibold mb-2">Loaded ${data.points.length} points</h2>
        <p class="text-sm mb-2">Total: ${data.totalCount || 'unknown'}</p>
        <ul class="list-disc pl-5 space-y-1">
          ${data.points
            .map(
              (p) => `
            <li class="text-sm">
              <span class="font-mono">${p.uuid}</span>
              <span class="text-gray-600 ml-2">${p.type || ''}</span>
            </li>
          `
            )
            .join('')}
        </ul>
      </div>
    `;
  } catch (error) {
    resultsDiv.innerHTML = `
      <div class="error-message">
        <strong>Error:</strong> ${error.message}
      </div>
    `;
  }
}

// Setup event listeners when DOM is ready
document.getElementById('loadBtn').addEventListener('click', loadPoints);
```

**static/utils.js:**

```javascript
// Auth token management
export function getAuthToken() {
  const params = new URLSearchParams(window.location.search);
  let token = params.get('token') || params.get('auth_token');

  // Check hash if not in query params
  if (!token && window.location.hash) {
    const hashParams = new URLSearchParams(window.location.hash.substring(1));
    token = hashParams.get('token') || hashParams.get('auth_token');
  }

  return token;
}

// API call helper with authentication and error handling
export async function apiCall(endpoint, body = null) {
  const token = getAuthToken();
  const url = `${window.location.origin}${endpoint}`;

  const options = {
    method: body ? 'POST' : 'GET',
    headers: {
      'Content-Type': 'application/json',
    },
  };

  // Add auth header if token exists
  if (token) {
    options.headers['Authorization'] = `Bearer ${token}`;
  }

  // Add body for POST requests
  if (body) {
    options.body = JSON.stringify(body);
  }

  const response = await fetch(url, options);

  // Handle authentication errors - reload to get new token
  if (response.status === 401 || response.status === 403) {
    console.warn('Authentication failed, reloading to get new token...');
    window.location.reload();
    throw new Error('Authentication failed, reloading...');
  }

  // Handle other HTTP errors
  if (!response.ok) {
    throw new Error(`HTTP ${response.status}: ${response.statusText}`);
  }

  return response.json();
}
```

**That's it!** No npm, no build tools, no bundling. Just edit and save these files directly in VSCode (even in a remote filesystem).

---

## Best Practices

1. **No-build workflow** - Use vanilla JS and CDN libraries to avoid build complexity
2. **Always use browser origin** for API calls
3. **Handle auth errors gracefully** with automatic reload
4. **Load large frameworks from CDN** to keep files lightweight
5. **Follow the house style** for visual consistency
6. **Test in an iframe** during development
7. **Implement proper error handling** for network failures
8. **Use pagination** for large datasets
9. **Use ES6 modules** (`type="module"`) for better code organization
10. **Edit directly in VSCode** - works perfectly with remote filesystems

---

## Troubleshooting

### Plugin doesn't load

- Verify all files are in the `/static` folder
- Check browser console for errors
- Ensure HTML references correct JS/CSS paths

### API calls fail with CORS errors

- Ensure you're using `window.location.origin`
- Check that auth token is being passed correctly

### 401/403 errors

- Verify token is in URL parameters
- Check token extraction logic
- Ensure reload logic is implemented

### Styling looks different

- Verify CSS custom properties match house style
- Check that Tailwind config is applied
- Ensure fonts are loaded from CDN

---

## Support

For questions or issues with plugin development, please refer to the main project documentation or contact the development team.
