# Running Locally

## Prerequisites

- Go installed (see `go.mod` for version)
- A `.env` file in the project root with the following variables set:

| Variable | Required | Notes |
|---|---|---|
| `MONGO_URL` | Yes | MongoDB Atlas connection string (includes DB name in path) |
| `JWT_SECRET` | Yes | |
| `SECRET` | Yes | |
| `RESET_TOKEN_SECRET` | Yes | |
| `STREAM_API_KEY` | Yes | |
| `STREAM_API_SECRET` | Yes | |
| `EMAIL_USER` | No | |
| `EMAIL_PASS` | No | |
| `PORT` | No | Defaults to `3000` |
| `DEBUG` | No | Defaults to `false`. When `true`, OTP responses (register/forgot-password/forgot-pin) include the OTP directly for local testing against Mailtrap's sandbox. Must stay unset/`false` in any deployed environment. |

No local database is needed — the app connects directly to the MongoDB Atlas cluster specified in `MONGO_URL`.

## Run

From the project root:

```bash
go run ./cmd/api
```

This runs in the foreground; logs print directly to the terminal. Stop with `Ctrl+C`.

Or build a binary once and run it repeatedly:

```bash
go build -o bin/carenav_api ./cmd/api
./bin/carenav_api
```

To use a different port:

```bash
PORT=4000 go run ./cmd/api
```

## Verify it's up

```bash
curl http://localhost:3000/
# → Welcome to my server
```

## Swagger / API docs

The API docs are generated from comment annotations on the handlers using
[swaggo/swag](https://github.com/swaggo/swag).

Once the server is running, view the docs at:

```
http://localhost:3000/swagger/index.html
```

Raw spec: `http://localhost:3000/swagger/doc.json`

### Regenerating the docs

After changing or adding `@...` annotations on a handler (or adding a new
route), regenerate `docs/`:

```bash
go install github.com/swaggo/swag/cmd/swag@latest   # one-time
swag init -g cmd/api/main.go -o docs
```

Then rebuild/restart the server — `docs/docs.go` is imported by the router
and compiled into the binary.

## Routes

- `POST /register`
- `POST /verify-registration-Otp`
- `POST /login`
- `POST /forgot-password`
- `POST /verify-otp`
- `POST /reset-password`
- `POST /create-pin`
- `POST /verify-pin`
- `POST /forgot-pin`
- `POST /reset-pin`
- `POST /create-doctor`
- `POST /doctor-login`
- `GET /api/users/:id`
- `DELETE /api/users/:id`
