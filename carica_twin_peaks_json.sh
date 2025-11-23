#!/bin/bash

# read quality index from command line argument
# usage: ./carica_twin_peaks_json.sh [quality_index]
QUALITY=${1:-2} # Default to quality index 2 if not provided

# Clear playlist
curl http://localhost:8080/clear
echo ""

# Load Twin Peaks season 1 playlist as JSON and start playback
curl -X POST http://localhost:8080/load \
  -H "Content-Type: application/json" \
  -d '[
    {
      "type": "video",
      "path": "/media/7. serie/twin peaks/Twin.Peaks.S01/Twin.Peaks.S01E01.Pilot.v2.720p.Bluray.AC3.ITA.DTS.ENG.Subs.x264-HDitaly.mkv",
      "quality_index": '$QUALITY',
      "aspect_ratio_4_3": false,
      "text_banner": false
    },
    {
      "type": "video",
      "path": "/media/7. serie/twin peaks/Twin.Peaks.S01/Twin.Peaks.S01E02.Traces.To.Nowhere.720p.Bluray.AC3.ITA.DTS.ENG.Subs.x264-HDitaly.mkv",
      "quality_index": '$QUALITY',
      "aspect_ratio_4_3": false,
      "text_banner": false
    },
    {
      "type": "video",
      "path": "/media/7. serie/twin peaks/Twin.Peaks.S01/Twin.Peaks.S01E03.Zen.Or.The.Skill.To.Catch.A.Kill.720p.Bluray.AC3.ITA.DTS.ENG.Subs.x264-HDitaly.mkv",
      "quality_index": '$QUALITY',
      "aspect_ratio_4_3": false,
      "text_banner": false
    },
    {
      "type": "video",
      "path": "/media/7. serie/twin peaks/Twin.Peaks.S01/Twin.Peaks.S01E04.Rest.In.Pain.720p.Bluray.AC3.ITA.DTS.ENG.Subs.x264-HDitaly.mkv",
      "quality_index": '$QUALITY',
      "aspect_ratio_4_3": false,
      "text_banner": false
    },
    {
      "type": "video",
      "path": "/media/7. serie/twin peaks/Twin.Peaks.S01/Twin.Peaks.S01E05.The.One.Armed.Man.720p.Bluray.AC3.ITA.DTS.ENG.Subs.x264-HDitaly.mkv",
      "quality_index": '$QUALITY',
      "aspect_ratio_4_3": false,
      "text_banner": false
    },
    {
      "type": "video",
      "path": "/media/7. serie/twin peaks/Twin.Peaks.S01/Twin.Peaks.S01E06.Coopers.Dreams.720p.Bluray.AC3.ITA.DTS.ENG.Subs.x264-HDitaly.mkv",
      "quality_index": '$QUALITY',
      "aspect_ratio_4_3": false,
      "text_banner": false
    },
    {
      "type": "video",
      "path": "/media/7. serie/twin peaks/Twin.Peaks.S01/Twin.Peaks.S01E07.Realization.Time.720p.Bluray.AC3.ITA.DTS.ENG.Subs.x264-HDitaly.mkv",
      "quality_index": '$QUALITY',
      "aspect_ratio_4_3": false,
      "text_banner": false
    },
    {
      "type": "video",
      "path": "/media/7. serie/twin peaks/Twin.Peaks.S01/Twin.Peaks.S01E08.The.Last.Evening.720p.Bluray.AC3.ITA.DTS.ENG.Subs.x264-HDitaly.mkv",
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
