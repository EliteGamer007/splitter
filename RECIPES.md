# Developer Recipes & Examples

This document provides code snippets and guides for common tasks when extending Splitter.

## 1. Adding a New ActivityPub Activity Type

If you want to support a new activity like `Invite` or `Flag`:

1. **Update the Actor Implementation**: Add the activity type to your ActivityPub processing logic in `internal/federation/`.
2. **Handle the Inbound Payload**: Update `inbox_handler.go` to recognize the type.
    ```go
    case "Invite":
        return h.handleInvite(ctx, activity)
    ```
3. **Update the Outbox**: Create a helper to wrap the outbound JSON.

## 2. Customize the Frontend Theme

Splitter uses **Tailwind CSS** with CSS Variables for theming. To add a new "Midnight" theme:

1. Open `Splitter-frontend/styles/globals.css`.
2. Add a new theme block:
    ```css
    .theme-midnight {
      --background: 240 10% 3.9%;
      --foreground: 0 0% 98%;
      --primary: 263.4 70% 50.4%;
      /* ... other variables */
    }
    ```
3. Add a toggle in the **Settings** component to apply the class to the `<body>` or root container.

## 3. Creating a Custom Bot

Splitter supports programmatic interaction. You can build a bot using the API:

```javascript
// Example Node.js snippet for a "Daily Fact" bot
const response = await fetch('http://localhost:8000/api/v1/posts', {
  method: 'POST',
  headers: {
    'Authorization': 'Bearer YOUR_TOKEN',
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    content: "Did you know? Splitter is federated! #TodayIFact",
    visibility: "public"
  })
});
```

## 4. Hooking into the Message Pipeline

To add auto-responses or filters to DMs:
- Look at `internal/handlers/message_handler.go`.
- Insert your logic before `repo.CreateMessage(...)`.
- Useful for implementing "Away Messages" or automated safety scans.
