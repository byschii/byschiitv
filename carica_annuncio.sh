#!/bin/bash

# Load Twin Peaks season 1 playlist as JSON and start playback
curl -X POST http://localhost:8080/load \
  -H "Content-Type: application/json" \
  -d '[
    {
      "type": "video",
      "path": "/mnt/c/poopleanno/annuncio.mp4",
      "hi_quality": true,
      "aspect_ratio_4_3": false
    }
  ]'

echo ""

# Start playback
curl http://localhost:8080/start

echo ""

# List current playlist
curl http://localhost:8080/list
