# fileshare
File sharing service with individual temporary link generation.
Uses a simple basic auth, but blocks any brute force attack.


Usage: `./fileshare -secret 123 -path "./store"`

API:

`GET /file.name?key=KEY` - getting file with name and generated key (must be exists in store cache and not expired)
`GET /newlink/upload` - show page for upload a file with basic auth (admin:secret_param)
`POST /newlink/upload` - upload a file in part "file" of multipart body, and return a link, that expired after 1 day
`GET /newlink/gen?file=file.name` - generate new unique link for file.name (must be exists in store cache), that expired after 1 day
