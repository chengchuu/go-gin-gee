**Frontend Guide: Send Discord Message API**

Use this endpoint to send a message to the configured Discord webhook:

```http
POST /api/gee/webhook-message
```

**Important Security Rule**

Do not call this endpoint directly from public browser code with the API key.

The `X-Webhook-API-Key` is secret. If it is placed in frontend JavaScript, users can inspect it and reuse it. Call this API only from trusted server-side code, such as:

- backend service
- Next.js API route / server action
- internal admin server
- CI/CD script
- private operations tool

**Required Config**

Backend must have:

```json
{
  "Data": {
    "WEBHOOK_ID": "your_discord_webhook_id",
    "WEBHOOK_TOKEN": "your_discord_webhook_token",
    "EnableWebhookAPI": "on",
    "WebhookAPIKeys": [
      "gee_webhook_12345"
    ]
  }
}
```

If `EnableWebhookAPI` is not `"on"`, the endpoint returns `404`.

**Request**

Headers:

```http
Content-Type: application/json
X-Webhook-API-Key: gee_webhook_12345
```

Body:

```json
{
  "content": "Hello from frontend service"
}
```

`content` is required and cannot be empty.

**Success Response**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "content": "Hello from frontend service"
  }
}
```

**Error Responses**

| Status | Meaning |
|---:|---|
| `400` | Invalid JSON body or empty `content` |
| `401` | Missing or invalid `X-Webhook-API-Key` |
| `404` | Webhook API is disabled |
| `502` | Failed to send message to Discord |
| `503` | Discord webhook credentials are not configured |

**Server-Side Fetch Example**

```ts
export async function sendDiscordMessage(content: string) {
  const res = await fetch("BASE_URL/api/gee/webhook-message", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "X-Webhook-API-Key": process.env.GEE_WEBHOOK_API_KEY!,
    },
    body: JSON.stringify({ content }),
  });

  const data = await res.json();

  if (!res.ok) {
    throw new Error(data?.message || "Failed to send Discord message");
  }

  return data;
}
```

**curl Test**

```bash
curl -i -X POST "BASE_URL/api/gee/webhook-message" \
  -H "Content-Type: application/json" \
  -H "X-Webhook-API-Key: gee_webhook_12345" \
  -d '{"content":"Hello from API"}'
```
