# i

i is a minimal program for uploading files to your server and getting a "i.example.com/file.png" link back, similar to imgur.com. It is based on [fourtf/i](https://github.com/fourtf/i) but dockerized with configurable environment variables.

## what you need to build

- docker

## how to use

- Open `.env` and edit the configuration
- Generate password hash with `make hash`
- Build with `docker compose build`
- Generate base64 auth string with `make auth USER=user PASSWORD=password` for authorization header

## environment variables

- `UPLOADS_DIRECTORY`: directory where files are stored (default: uploads)
- `MAX_FILE_AGE`: maximum age of a file in hours
- `AUTH_USER`: username for basic auth
- `AUTH_HASHED_PASSWORD`: hashed password for basic auth from
- `DISABLE_CLEANUP`: disable cleanup of old files
