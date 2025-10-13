# BYSCHIITV

It s a television.

There are 3 components:
- a golang server
    - handles the scheduling of shows via a REST API that controlls a subprocess running `ffmpeg`
- a nginx server
    - serves HLS streams and listens for RTMP streams
- a client to plan shows
    - tui client to plan shows via the REST API

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






