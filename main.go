//Trying to prototype the video upload flow of the user on system like youtube 
//used channels to replace queues


package main

import (
	"fmt"
	"sync"
	"time"
)

type Video struct {
    ID     string
    Title  string
    Chunks int
}

type Clip struct {
    VideoID string
    ClipID  int
    Data    string
}

type VideoMeta struct {
    Title      string
    Chunks     int
    Transcoded map[string][]string
    Status     string
}

var (
    videoUploadCh   = make(chan Video)
    transcode240pCh = make(chan Clip)
    transcode720pCh = make(chan Clip)
    jobs240pCh      = make(chan Clip, 10)
    jobs720pCh      = make(chan Clip, 10)
)

var videoDB = make(map[string]*VideoMeta)
var videoDBMu sync.RWMutex


func uploadVideo(id string, title string, chunks int) {
    video := Video{ID: id, Title: title, Chunks: chunks}
    videoDB[id] = &VideoMeta{
        Title:      title,
        Chunks:     chunks,
        Transcoded: make(map[string][]string),
        Status:     "uploaded",
    }
    fmt.Printf("[Uploader] Uploaded video: %s with %d chunks\n", id, chunks)
    videoUploadCh <- video
}

func chunker(videoUploadCh <-chan Video, trans240pCh chan<- Clip, trans720pCh chan<- Clip) {
    for video := range videoUploadCh {
        fmt.Printf("[Chunker] Received video %s with %d chunks\n", video.ID, video.Chunks)

        if meta, ok := videoDB[video.ID]; ok {
            meta.Status = "chunking"
        }

        for i := 1; i <= video.Chunks; i++ {
            time.Sleep(300 * time.Millisecond)
            clip := Clip{
                VideoID: video.ID,
                ClipID:  i,
                Data:    fmt.Sprintf("clip_data_%d", i),
            }

            fmt.Printf("[Chunker] ➤ Emitting Clip #%d for video %s\n", i, video.ID)
            trans240pCh <- clip
            trans720pCh <- clip
        }

        if meta, ok := videoDB[video.ID]; ok {
            meta.Status = "chunked"
        }

        fmt.Printf("[Chunker] Finished chunking video %s\n", video.ID)
    }
}

func startDispatcher(resolution string, inputCh <-chan Clip, jobsCh chan<- Clip) {
    go func() {
        for clip := range inputCh {
            fmt.Printf("[%s-Dispatcher] Dispatching clip %d of video %s\n", resolution, clip.ClipID, clip.VideoID)
            jobsCh <- clip
        }
    }()
}

func startWorkerPool(resolution string, jobsCh <-chan Clip, numWorkers int) {
    for i := 0; i < numWorkers; i++ {
        go func(workerID int) {
            for clip := range jobsCh {
                fmt.Printf("[%s-Worker-%d] ▶️ Processing clip #%d from %s\n", resolution, workerID, clip.ClipID, clip.VideoID)

                time.Sleep(1 * time.Second) 

                output := fmt.Sprintf("vid_%s_chunk_%d_%s.ts", clip.VideoID, clip.ClipID, resolution)

				videoDBMu.RLock()
				meta, ok := videoDB[clip.VideoID]
				videoDBMu.RUnlock()
                if !ok {
                    fmt.Printf("[%s-Worker-%d] video %s not found in DB\n", resolution, workerID, clip.VideoID)
                    continue
                }

                if meta.Transcoded == nil {
                    meta.Transcoded = make(map[string][]string)
                }
				videoDBMu.Lock()
				meta.Transcoded[resolution] = append(meta.Transcoded[resolution], output)
				videoDBMu.Unlock()


                fmt.Printf("[%s-Worker-%d] Done: %s\n", resolution, workerID, output)
            }
        }(i)
    }
}

func startManifestWatcher() {
    go func() {
        for {
            time.Sleep(2 * time.Second)

            for videoID, meta := range videoDB {
                if meta.Status != "chunked" {
                    continue
                }

                requiredResolutions := []string{"240p", "720p"}
                allComplete := true

                for _, res := range requiredResolutions {
                    clips, ok := meta.Transcoded[res]
                    if !ok || len(clips) != meta.Chunks {
                        allComplete = false
                        break
                    }
                }

                if allComplete {
                    meta.Status = "ready"
                    fmt.Printf("[Manifest] Video %s is ready to serve!\n", videoID)
                    printManifest(videoID, meta)
                }
            }
        }
    }()
}

func printManifest(videoID string, meta *VideoMeta) {
    fmt.Printf("------ Manifest for %s ------\n", videoID)
    for res, clips := range meta.Transcoded {
        fmt.Printf("Resolution: %s\n", res)
        for _, clip := range clips {
            fmt.Println("   •", clip)
        }
    }
    fmt.Println("----------------------------")
}

func main() {
	// Tried Replicating the part when youtube tried to chunk uploaded video after something successfully aded to s3
	// can add some triggers s3 so that chunker service can be notified for thesame
    go chunker(videoUploadCh, transcode240pCh, transcode720pCh)
	

	// Tried replicating the part when after chunking chunker add small files to some pubsub and it got to its subscriber
	// in our case its transcoders 
	startDispatcher("240p", transcode240pCh, jobs240pCh)
    startDispatcher("720p", transcode720pCh, jobs720pCh)
    startWorkerPool("240p", jobs240pCh, 3)
    startWorkerPool("720p", jobs720pCh, 3)


	// the manifest file which is being used by client to render the video 
    startManifestWatcher()
	//upload the photo by user
    go uploadVideo("vid123", "Bike Trip", 5)
    go uploadVideo("vid456", "GoLang Demo", 3)
    go uploadVideo("vid789", "Mumbai Vlog", 4)

    select {}
}
