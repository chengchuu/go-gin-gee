# Retire Docker Tag Lookup with a Reusable Handler

## Summary

Remove the Docker Hub lookup implementation while preserving `GET /api/gee/get-tag-name` as an explicitly retired endpoint. Future retired routes can reuse the same handler.

This document is an implementation plan. Creating it does not implement the retirement.

## Implementation

- Add `controllers.RetiredAPI` as a shared terminal route handler.
- Add `CodeAPIRetired = 41001` to the existing `http_err` constants.
- Use `http_err.Failure` and the existing `http_err.APIResponse` envelope to return HTTP **410 Gone** with exactly these fields:

  ```json
  {
    "code": 41001,
    "message": "This API has been retired.",
    "data": null
  }
  ```

- Set `Cache-Control: no-store`, retaining the recommended default.
- Map the existing GET route directly to `RetiredAPI`; do not retain a Docker-specific wrapper or `tagName` response.
- Remove the Docker-tag controller, repository, dedicated models, and obsolete tag-selection test.
- Keep shared dependencies such as `resty` and `lo`, which remain in use elsewhere.
- Update README and repository guidance to document the response and how to register future retired routes.

## Validation

- Test the exact status, response fields, message, null data, JSON content type, and cache header.
- Test the actual Docker-tag route without database configuration or outbound HTTP requests.
- Verify the shared handler works on another test-only route.
- Preserve existing behavior for unknown routes and unrelated endpoints.
- Search for stale references to the removed Docker-tag implementation.
- Run Go 1.25 repository-wide tests, affected-package race tests, build, vet, and `git diff --check`.

## Boundaries

Only explicitly registered retired routes receive this response. Do not add catch-all retirement middleware, a configurable retirement registry, or a compatibility payload.

Do not change dependencies, databases, Docker workflows, or unrelated APIs. Do not publish or deploy as part of this work.
