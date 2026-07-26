# Key-Value API

The key-value store is available under `/api/gee/kv` and exposes only three endpoints:

| Method | Endpoint | API key |
| --- | --- | --- |
| `POST` | `/api/gee/kv/get` | Required for private entries |
| `POST` | `/api/gee/kv/set` | Required |
| `POST` | `/api/gee/kv/increment` | Required |

Supply credentials in `X-API-Key`. Accepted credentials are configured in `Data.KVAPIKeys`. Public entries can be read without credentials. Private entries return `404 key not found` to unauthorized callers so their existence is not disclosed.

## Requests

Keys are accepted only in JSON request bodies. A valid key matches `^[a-z0-9][a-z0-9._:-]{0,127}$`.

Get:

```json
{
  "key": "site.title"
}
```

Set (create or replace):

```json
{
  "key": "site.title",
  "value": "Gee Service",
  "content_type": "text/plain",
  "visibility": "private"
}
```

`content_type` defaults to `text/plain`. `visibility` accepts only `public` or `private` and defaults to `private`.

Increment:

```json
{
  "key": "page.views",
  "delta": 1
}
```

An omitted `delta` defaults to `1`, while an explicit `0` remains zero. Positive and negative deltas are supported. Missing counters are initialized from zero, and updates use atomic database arithmetic. Counters use the dedicated `kv_counters` table; attempting to increment a key already stored as a regular entry returns `409 incompatible value type`.

## Responses

Every response contains exactly `code`, `message`, and `data`:

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "key": "page.views",
    "value": 101,
    "delta": 1
  }
}
```

Failures use a meaningful HTTP status, a non-zero application code, and normally `data: null`:

```json
{
  "code": 40401,
  "message": "key not found",
  "data": null
}
```

| Application code | Meaning | HTTP status |
| ---: | --- | ---: |
| `0` | success | `200` or `201` |
| `40001` | invalid request | `400` |
| `40002` | invalid key | `400` |
| `40003` | invalid value | `400` |
| `40101` | API key required | `401` |
| `40301` | access denied | `403` |
| `40401` | key not found | `404` |
| `40901` | incompatible value type | `409` |
| `50001` | internal server error | `500` |
