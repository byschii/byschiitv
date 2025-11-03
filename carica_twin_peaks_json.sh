#!/bin/bash

# Load Twin Peaks season 1 playlist as JSON and start playback
curl -X POST http://localhost:8080/load \
  -H "Content-Type: application/json" \
  -d '[
    {
      "type": "video",
      "path": "/media/7. serie/twin peaks/Twin.Peaks.S01/Twin.Peaks.S01E01.Pilot.v2.720p.Bluray.AC3.ITA.DTS.ENG.Subs.x264-HDitaly.mkv",
      "hi_quality": true,
      "aspect_ratio_4_3": false
    },
    {
      "type": "video",
      "path": "/media/7. serie/twin peaks/Twin.Peaks.S01/Twin.Peaks.S01E02.Traces.To.Nowhere.720p.Bluray.AC3.ITA.DTS.ENG.Subs.x264-HDitaly.mkv",
      "hi_quality": true,
      "aspect_ratio_4_3": false
    },
    {
      "type": "video",
      "path": "/media/7. serie/twin peaks/Twin.Peaks.S01/Twin.Peaks.S01E03.Zen.Or.The.Skill.To.Catch.A.Kill.720p.Bluray.AC3.ITA.DTS.ENG.Subs.x264-HDitaly.mkv",
      "hi_quality": true,
      "aspect_ratio_4_3": false
    },
    {
      "type": "video",
      "path": "/media/7. serie/twin peaks/Twin.Peaks.S01/Twin.Peaks.S01E04.Rest.In.Pain.720p.Bluray.AC3.ITA.DTS.ENG.Subs.x264-HDitaly.mkv",
      "hi_quality": true,
      "aspect_ratio_4_3": false
    },
    {
      "type": "video",
      "path": "/media/7. serie/twin peaks/Twin.Peaks.S01/Twin.Peaks.S01E05.The.One.Armed.Man.720p.Bluray.AC3.ITA.DTS.ENG.Subs.x264-HDitaly.mkv",
      "hi_quality": true,
      "aspect_ratio_4_3": false
    },
    {
      "type": "video",
      "path": "/media/7. serie/twin peaks/Twin.Peaks.S01/Twin.Peaks.S01E06.Coopers.Dreams.720p.Bluray.AC3.ITA.DTS.ENG.Subs.x264-HDitaly.mkv",
      "hi_quality": true,
      "aspect_ratio_4_3": false
    },
    {
      "type": "video",
      "path": "/media/7. serie/twin peaks/Twin.Peaks.S01/Twin.Peaks.S01E07.Realization.Time.720p.Bluray.AC3.ITA.DTS.ENG.Subs.x264-HDitaly.mkv",
      "hi_quality": true,
      "aspect_ratio_4_3": false
    },
    {
      "type": "video",
      "path": "/media/7. serie/twin peaks/Twin.Peaks.S01/Twin.Peaks.S01E08.The.Last.Evening.720p.Bluray.AC3.ITA.DTS.ENG.Subs.x264-HDitaly.mkv",
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
