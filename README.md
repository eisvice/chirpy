Chirpy — Simple API

This README explains how to run the server and consume the Chirpy HTTP API.

**Requirements**
- **Go**: Build/run using `go run` or `go build`.
- **Postgres**: Set `DB_URL` to point to your Postgres instance.

**Environment**
- **`DB_URL`**: Postgres connection string used by the app.
- **`PLATFORM`**: set to `DEV` to enable the `/admin/reset` endpoint.
- **`SECRET_KEY`**: HMAC secret used to sign access JWTs.
- **`POLKA_KEY`**: API key used to authenticate Polka webhooks.

**Run**
- Start the server from the project root:

```
DB_URL="postgres://user:pass@localhost:5432/dbname?sslmode=disable" \
SECRET_KEY="supersecret" POLKA_KEY="polka-key" PLATFORM=DEV \
  go run main.go
```

Server defaults to listening on port `8080`.

**Quick overview**
- **File server**: static files are served under `GET /app/`.
- **Health**: `GET /api/healthz` → returns `OK`.
- **Admin metrics**: `GET /admin/metrics` (returns simple HTML with file server hit count).
- **Admin reset**: `POST /admin/reset` (allowed only when `PLATFORM=DEV`).

**Auth basics**
- Access tokens are short-lived JWTs (signed with `SECRET_KEY`). Send them as:

  `Authorization: Bearer <access_token>`

- Refresh tokens are opaque hex strings returned by the login endpoint. They are also sent in the `Authorization` header as a Bearer token when calling refresh/revoke endpoints.

- Polka webhook authentication uses an API key in the `Authorization` header with the `ApiKey ` prefix:

  `Authorization: ApiKey <POLKA_KEY>`

**API Endpoints**

- **Create user**: `POST /api/users`
  - **Body**: `{"email":"you@example.com","password":"your-password"}`
  - **Returns**: `201` with user fields (no tokens).
  - **Example**:

    ```bash
    curl -X POST http://localhost:8080/api/users \
      -H "Content-Type: application/json" \
      -d '{"email":"alice@example.com","password":"s3cret"}'
    ```

- **Login**: `POST /api/login`
  - **Body**: `{"email":"you@example.com","password":"your-password"}`
  - **Returns**: `200` with `token` (access JWT) and `refresh_token` (opaque string).
  - **Example**:

    ```bash
    curl -X POST http://localhost:8080/api/login \
      -H "Content-Type: application/json" \
      -d '{"email":"alice@example.com","password":"s3cret"}'
    ```

- **Refresh access token**: `POST /api/refresh`
  - **Auth**: send the refresh token in `Authorization: Bearer <refresh_token>`
  - **Returns**: `200` with a new `token` (access JWT).
  - **Example**:

    ```bash
    curl -X POST http://localhost:8080/api/refresh \
      -H "Authorization: Bearer <refresh_token>"
    ```

- **Revoke refresh token**: `POST /api/revoke`
  - **Auth**: `Authorization: Bearer <refresh_token>`
  - **Returns**: `204` on success.

- **Update user**: `PUT /api/users`
  - **Auth**: `Authorization: Bearer <access_token>`
  - **Body**: `{"email":"new@example.com","password":"new-pass"}`
  - **Returns**: `200` with updated user info and the provided access token.

- **Create chirp**: `POST /api/chirps`
  - **Auth**: `Authorization: Bearer <access_token>`
  - **Body**: `{"body":"Hello world"}`
  - **Constraints**: `body` max length 140 characters.
  - **Returns**: `201` with created chirp JSON.

- **List chirps**: `GET /api/chirps`
  - **Query**: optional `author_id=<uuid>` to filter by author.
  - **Returns**: `200` with an array of chirps.
  - **Example**:

    ```bash
    curl http://localhost:8080/api/chirps
    curl "http://localhost:8080/api/chirps?author_id=<author-uuid>"
    ```

- **Get chirp**: `GET /api/chirps/{chirpID}`
  - **Returns**: `200` with chirp JSON or `404` if not found.

- **Delete chirp**: `DELETE /api/chirps/{chirpID}`
  - **Auth**: `Authorization: Bearer <access_token>` (must be the chirp owner).
  - **Returns**: `204` on success.

- **Polka webhook**: `POST /api/polka/webhooks`
  - **Auth**: `Authorization: ApiKey <POLKA_KEY>`
  - **Body** example:

    ```json
    {
      "event": "user.upgraded",
      "data": { "user_id": "<uuid>" }
    }
    ```
  - **Notes**: only `user.upgraded` is handled — the endpoint upgrades the user to Chirpy Red.

**Error handling**
- Errors are returned as JSON: `{"error":"message"}` with appropriate HTTP status codes.

**Notes & implementation details**
- Access tokens are JWTs and expire (short-lived). Use the refresh token to obtain new access tokens.
- The `Authorization` header is required for endpoints that need authentication.
- The `POST /admin/reset` endpoint will delete users when `PLATFORM=DEV` — use with care.
