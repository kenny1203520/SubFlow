# SubFlow Agent Documentation

## Project Overview
**SubFlow** is a subscription management and group-buying platform. It allows users to track recurring expenses, split costs with groups, and manage shared subscriptions.

## Tech Stack
- **Frontend**: Vue 3 (Composition API), Vite, TypeScript, Pinia, Vue Router, `vue-i18n`.
- **Backend**: Node.js, Express, Socket.IO (with MessagePack parser), PostgreSQL (`pg` driver), Lucia Auth.
- **Communication**: Binary WebSocket protocol (Socket.IO + MessagePack). No public REST APIs (except Auth endpoints).
- **Styling**: Vanilla CSS with a strict Design System (Variables, Utility Classes).

## Architectural Principles (OOP & SOLID)
Status: *The project is transitioning from procedural to Object-Oriented patterns. All new code MUST follow these principles.*

### Backend Architecture
**Pattern**: `Controller` -> `Service` -> `Repository`

1.  **Controllers (Socket Handlers)**
    *   **Responsibility**: Handle incoming Socket.IO events, validate payloads, and call Services.
    *   **Rule**: NO business logic or SQL queries in controllers.
    *   **Format**: Use Class-based controllers where possible, or clearly separated handler modules.

2.  **Services (Business Logic)**
    *   **Responsibility**: Contain the core business rules (e.g., calculating splits, handling subscription cycles).
    *   **Rule**: Services should be agnostic of the transport layer (HTTP/Socket). They accept data and return results/errors.
    *   **OOP**: Define services as Classes (e.g., `ExpenseService`) to allow dependency injection or easier testing.

3.  **Repositories/Models (Data Access)**
    *   **Responsibility**: Execute SQL queries and map results to domain objects.
    *   **Rule**: All SQL must live here. No `pool.query` in Services or Controllers.
    *   **OOP**: Use Classes (e.g., `UserRepository`) to encapsulate table operations.

### Frontend Architecture
**Pattern**: `View` -> `Composable (Service)` -> `Store`

1.  **Views/Components**:
    *   **Responsibility**: Rendering UI and handling user interaction.
    *   **Rule**: Minimal logic. Delegate complex operations to Composables or Stores.
    *   **i18n**: HARD REQUIREMENT. All text must be tokenized (`$t('key')`). NO hardcoded strings.
    *   **RWD**: ALL layouts must be responsive (Mobile First -> Desktop).

2.  **Composables (`use...`)**:
    *   **Responsibility**: Encapsulate reusable logic and state management (e.g., `useSubscriptionActions`).
    *   **Rule**: Treat these as your "Frontend Services".

3.  **Stores (Pinia)**:
    *   **Responsibility**: Global state (User session, cached data).

## Development Guidelines

### 1. Code Style
*   **TypeScript**: Strict typing required. Avoid `any`. Define interfaces for all socket payloads and database rows.
*   **CSS**: Use the Global Design System in `style.css`.
    *   Use variables: `var(--primary-color)`, `var(--bg-surface)`.
    *   Use utility classes: `grid`, `flex`, `hidden`, `text-danger`.
    *   AVOID: Hardcoded hex colors or arbitrary pixel values.

### 2. Internationalization (i18n)
*   **Default**: English (`en`).
*   **Structure**: Group keys by domain (e.g., `auth.login`, `dashboard.totalOwed`).
*   **Workflow**: When adding a new feature, immediately add keys to BOTH `en/{module}.json` and `zh/{module}.json`.

### 3. Error Handling
*   **Backend**: Catch errors in Controllers. Return standardized error objects `{ status: 'error', message: '...' }` to the client.
*   **Frontend**: handle errors gracefully. Show UI feedback (toasts/alerts) to the user.

## Task Workflow for Agents
1.  **Understand**: Read strict requirements. Check `AGENT.md` (this file).
2.  **Plan**: Create an `implementation_plan.md`. Define the Architecture changes (Controller/Service split).
3.  **Implement**: Write code.
    *   *Backend*: Create Service -> Create Controller -> Register Event.
    *   *Frontend*: Update Locale -> Create Component.
4.  **Verify**:
    *   Check i18n (Switch languages).
    *   Check RWD (Resize window).
    *   Check Logic (Run flows).
5.  **Document**: Update `task.md` and `walkthrough.md`.

## Directory Structure (Refactored Target)
```
backend/src/
  controllers/    # Socket Event Handlers (Class-based)
  services/       # Business Logic (Class-based)
  repositories/   # Data Access (Class-based)
  models/         # TypeScript Interfaces/Types
  socket/         # Event Registration
```
