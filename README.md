# YouTube Video Upload & Transcoding Flow In-Memory Simulation (Go)

Hello
Its my first time trying things out in Go (definitely will try more of these out)
This project is a fun prototype where I tried to simulate how YouTube might handle video uploads, chunking, transcoding, and manifest generation

## What it Does

- Simulates a user uploading a video.
- Chunks the video into multiple smaller clips.
- Sends each clip to two transcoders
- Transcoding is handled concurrently using worker pools and goroutines.
- Once all clips are transcoded, a manifest is printed used by client to serve videos

## How to Run
go run main.go
