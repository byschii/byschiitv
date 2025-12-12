# BYSCHIITV

It s a television.

There are 3 components:
- a golang server
    - handles the scheduling of shows via a REST API that controlls a subprocess running `ffmpeg`
- a nginx server
    - serves HLS streams and listens for RTMP streams
- a client to plan shows
    - tui client to plan shows via the REST API

## Ports (when Docker is running)

**`localhost:80`** → nginx
- HLS player (web UI): `http://localhost/`
- Stream: `http://localhost/hls/stream.m3u8`
- Stats: `http://localhost/stat`

**`localhost:8080`** → golang API
- Control endpoints: `/start`, `/stop`, `/next`, `/enque/*`, `/list`, `/load`, `/clear`
- For scripts, TUI, curl - not for web client

**`localhost:1935`** → nginx RTMP (internal use)
- ffmpeg publishes here: `rtmp://localhost:1935/live/stream`
- Not accessed by browser

**Flow:** External tool → golang API (8080) → ffmpeg → nginx RTMP (1935) → nginx HLS (80) → browser

## Running Multiple Instances

To run multiple instances in parallel, use environment variables:

```bash
# Instance 1 (default ports)
docker-compose up -d

# Instance 2 (different ports)
CONTAINER_PREFIX=iptvsim2 HOST_PORT_HTTP=81 HOST_PORT_API=8081 HOST_PORT_RTMP=1936 \
  docker-compose up -d

# Or use env file
docker-compose --env-file .env.instance2 up -d
```

**Available env vars:**
- `CONTAINER_PREFIX` - Container name prefix (default: `iptvsim`)
- `HOST_PORT_HTTP` - nginx HTTP port (default: `80`)
- `HOST_PORT_API` - golang API port (default: `8080`)
- `HOST_PORT_RTMP` - nginx RTMP port (default: `1935`)
- `HOST_MEDIA_PATH` - Media directory (default: `./byschiitv/media`)

## Some of the phylosophy

- on the golang server
    - no database, all data is in mem
    - no authentication, it is assumed that the server is run in a trusted environment
    - no persistence, all data is lost on restart
    - it a tv, no ondemand viewing
        - the player can be on = as soon as a show ends -> the next one starts, as soon as a show is scheduled -> it starts
        - the player can be off = pause to debug, no shows are played
    - controlls quality via ffmpeg
        - so that the client can send any stream, the server will transcode it to a fixed quality
        - so that user connection quality does not matter, the server will serve a fixed quality (controlling bandwidth is important)

- on the nginx server
    - no authentication, it is assumed that the server is run in a trusted environment
    - no persistence, all data is lost on restart
    - no transcoding, the input stream is directly served as HLS
    - no recording, the input stream is not recorded
    - nginx should be on same machine as ffmpeg
        - so that ffmpeg can stream to nginx via localhost (no bandidth consumed) and nginx can serve the HLS only when requested

- on the client
    - no authentication, it is assumed that the client is run in a trusted environment
    - no persistence, all data is lost on restart
    - no fancy UI, it is a TUI, so that it can be run in a terminal
    - share utils with server, so that the client can validate input before sending it to the server
    - the idea is to build a planning day by day

## Env

- everythind on a raspberry pi 4b with 4gb ram
    - inside docker containers with rpi image that downloads the appropriate ffmpeg build 
- ideally: on a vpn (netmaker) to aws instance
    - netmaker has caddy integration, so that the nginx server can be accessed via a domain name with https






