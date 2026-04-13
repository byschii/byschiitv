#!/bin/bash

# read quality index from command line argument
# usage: ./carica_twin_peaks_json.sh [quality_index]
QUALITY=${1:-3} # Default to quality index 2 if not provided

# Clear playlist
curl http://localhost:8080/clear
echo ""

# Load Twin Peaks season 1 playlist as JSON and start playback
curl -X POST http://localhost:8080/load \
  -H "Content-Type: application/json" \
  -d '[
    {
      "type": "video",
      "path": "/media/7. serie/Neon Genesis Evangelion/NGE_Ep_01_ITA.mp4",
      "quality_index": '$QUALITY',
      "aspect_ratio_4_3": false,
      "text_banner": false
    },
    {
      "type": "video",
      "path": "/media/7. serie/Neon Genesis Evangelion/NGE_Ep_02_ITA.mp4",
      "quality_index": '$QUALITY',
      "aspect_ratio_4_3": false,
      "text_banner": false
    },
    {
      "type": "video",
      "path": "/media/7. serie/Neon Genesis Evangelion/NGE_Ep_03_ITA.mp4",
      "quality_index": '$QUALITY',
      "aspect_ratio_4_3": false,
      "text_banner": false
    },
    {
      "type": "video",
      "path": "/media/7. serie/Neon Genesis Evangelion/NGE_Ep_04_ITA.mp4",
      "quality_index": '$QUALITY',
      "aspect_ratio_4_3": false,
      "text_banner": false
    },
    {
      "type": "video",
      "path": "/media/7. serie/Neon Genesis Evangelion/NGE_Ep_05_ITA.mp4",
      "quality_index": '$QUALITY',
      "aspect_ratio_4_3": false,
      "text_banner": false
    },
    {
      "type": "video",
      "path": "/media/7. serie/Neon Genesis Evangelion/NGE_Ep_06_ITA.mp4",
      "quality_index": '$QUALITY',
      "aspect_ratio_4_3": false,
      "text_banner": false
    },
    {
      "type": "video",
      "path": "/media/7. serie/Neon Genesis Evangelion/NGE_Ep_07_ITA.mp4",
      "quality_index": '$QUALITY',
      "aspect_ratio_4_3": false,
      "text_banner": false
    },
    {
      "type": "video",
      "path": "/media/7. serie/Neon Genesis Evangelion/NGE_Ep_08_ITA.mp4",
      "quality_index": '$QUALITY',
      "aspect_ratio_4_3": false,
      "text_banner": false
    },
    {
      "type": "video",
      "path": "/media/7. serie/Neon Genesis Evangelion/NGE_Ep_09_ITA.mp4",
      "quality_index": '$QUALITY',
      "aspect_ratio_4_3": false,
      "text_banner": false
    },
    {
      "type": "video",
      "path": "/media/7. serie/Neon Genesis Evangelion/NGE_Ep_10_ITA.mp4",
      "quality_index": '$QUALITY',
      "aspect_ratio_4_3": false,
      "text_banner": false
    },
    {
      "type": "video",
      "path": "/media/7. serie/Neon Genesis Evangelion/NGE_Ep_11_ITA.mp4",
      "quality_index": '$QUALITY',
      "aspect_ratio_4_3": false,
      "text_banner": false
    },
    {
      "type": "video",
      "path": "/media/7. serie/Neon Genesis Evangelion/NGE_Ep_12_ITA.mp4",
      "quality_index": '$QUALITY',
      "aspect_ratio_4_3": false,
      "text_banner": false
    },
    {
      "type": "video",
      "path": "/media/7. serie/Neon Genesis Evangelion/NGE_Ep_13_ITA.mp4",
      "quality_index": '$QUALITY',
      "aspect_ratio_4_3": false,
      "text_banner": false
    },
    {
      "type": "video",
      "path": "/media/7. serie/Neon Genesis Evangelion/NGE_Ep_14_ITA.mp4",
      "quality_index": '$QUALITY',
      "aspect_ratio_4_3": false,
      "text_banner": false
    },
    {
      "type": "video",
      "path": "/media/7. serie/Neon Genesis Evangelion/NGE_Ep_15_ITA.mp4",
      "quality_index": '$QUALITY',
      "aspect_ratio_4_3": false,
      "text_banner": false
    },
    {
      "type": "video",
      "path": "/media/7. serie/Neon Genesis Evangelion/NGE_Ep_16_ITA.mp4",
      "quality_index": '$QUALITY',
      "aspect_ratio_4_3": false,
      "text_banner": false
    },
    {
      "type": "video",
      "path": "/media/7. serie/Neon Genesis Evangelion/NGE_Ep_17_ITA.mp4",
      "quality_index": '$QUALITY',
      "aspect_ratio_4_3": false,
      "text_banner": false
    },
    {
      "type": "video",
      "path": "/media/7. serie/Neon Genesis Evangelion/NGE_Ep_18_ITA.mp4",
      "quality_index": '$QUALITY',
      "aspect_ratio_4_3": false,
      "text_banner": false
    },
    {
      "type": "video",
      "path": "/media/7. serie/Neon Genesis Evangelion/NGE_Ep_19_ITA.mp4",
      "quality_index": '$QUALITY',
      "aspect_ratio_4_3": false,
      "text_banner": false
    },
    {
      "type": "video",
      "path": "/media/7. serie/Neon Genesis Evangelion/NGE_Ep_20_ITA.mp4",
      "quality_index": '$QUALITY',
      "aspect_ratio_4_3": false,
      "text_banner": false
    },
    {
      "type": "video",
      "path": "/media/7. serie/Neon Genesis Evangelion/NGE_Ep_21_ITA.mp4",
      "quality_index": '$QUALITY',
      "aspect_ratio_4_3": false,
      "text_banner": false
    },
    {
      "type": "video",
      "path": "/media/7. serie/Neon Genesis Evangelion/NGE_Ep_22_ITA.mp4",
      "quality_index": '$QUALITY',
      "aspect_ratio_4_3": false,
      "text_banner": false
    },
    {
      "type": "video",
      "path": "/media/7. serie/Neon Genesis Evangelion/NGE_Ep_23_ITA.mp4",
      "quality_index": '$QUALITY',
      "aspect_ratio_4_3": false,
      "text_banner": false
    },
    {
      "type": "video",
      "path": "/media/7. serie/Neon Genesis Evangelion/NGE_Ep_24_ITA.mp4",
      "quality_index": '$QUALITY',
      "aspect_ratio_4_3": false,
      "text_banner": false
    },
    {
      "type": "video",
      "path": "/media/7. serie/Neon Genesis Evangelion/NGE_Ep_25_ITA.mp4",
      "quality_index": '$QUALITY',
      "aspect_ratio_4_3": false,
      "text_banner": false
    },
    {
      "type": "video",
      "path": "/media/7. serie/Neon Genesis Evangelion/NGE_Ep_26_ITA.mp4",
      "quality_index": '$QUALITY',
      "aspect_ratio_4_3": false,
      "text_banner": false
    }

  ]'

echo ""

# Start playback
curl http://localhost:8080/start

echo ""

# List current playlist
curl http://localhost:8080/list
