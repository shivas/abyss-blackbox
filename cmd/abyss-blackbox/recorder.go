package main

import (
	"bytes"
	"fmt"
	"image"
	"image/gif"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/lxn/walk"
	"github.com/shivas/abyss-blackbox/combatlog"
	"github.com/shivas/abyss-blackbox/encoding"
)

const (
	RECORDER_STOPPED = iota
	RECORDER_RUNNING
	RECORDER_AWAITING_INITIAL_LOOT
)

type Recorder struct {
	sync.Mutex
	state               int
	frameChan           chan *image.Paletted
	loot                chan string
	config              *captureConfig
	done                chan bool
	frames              []*image.Paletted
	delays              []int
	recordingName       string
	lootRecords         []*encoding.LootRecord
	notificationChannel chan NotificationMessage
	combatlogReader     *combatlog.Reader
	charactersTracking  map[string]combatlog.CombatLogFile
}

// NewRecorder constructs Recorder
func NewRecorder(frameChan chan *image.Paletted, c *captureConfig, nc chan NotificationMessage, clr *combatlog.Reader) *Recorder {
	return &Recorder{
		frameChan:           frameChan,
		loot:                make(chan string, 2),
		state:               RECORDER_STOPPED,
		config:              c,
		frames:              make([]*image.Paletted, 0),
		delays:              make([]int, 0),
		done:                make(chan bool),
		lootRecords:         make([]*encoding.LootRecord, 0),
		notificationChannel: nc,
		combatlogReader:     clr,
		charactersTracking:  make(map[string]combatlog.CombatLogFile),
	}
}

// ClipboardListener event listener used to capture Clipboard changes for loot recording
func (r *Recorder) ClipboardListener() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	clipboard, err := walk.Clipboard().Text()
	if err != nil {
		return
	}
	select {
	case r.loot <- clipboard:
	default:
		log.Println("not recording, loot capture dropped")
	}
}

// StartLoop starts main recorded loop listening for frames and clipboard changes
func (r *Recorder) StartLoop() {
	go func(r *Recorder) {

		for {
			select {
			case <-r.done:
				return // exit loop

			case lootSnapshot := <-r.loot:
				r.Lock()

				switch r.state {
				case RECORDER_AWAITING_INITIAL_LOOT:
					r.notificationChannel <- NotificationMessage{Title: "Abyssal.Space recording started...", Message: "Initial cargo received, awaiting cargo after fillament activation"}
					r.state = RECORDER_RUNNING
					r.lootRecords = append(r.lootRecords, &encoding.LootRecord{Frame: 0, Loot: lootSnapshot})
				case RECORDER_RUNNING:
					r.notificationChannel <- NotificationMessage{Title: "Abyssal.Space recorder", Message: "Loot captured from clipboard!"}
					lr := &encoding.LootRecord{Frame: int32(len(r.frames) - 1), Loot: lootSnapshot}
					log.Printf("loot appended: %v\n", lr)
					r.lootRecords = append(r.lootRecords, lr)
				default:
					log.Printf("dropped loot record, %v", lootSnapshot)
				}

				r.Unlock()

			case frame := <-r.frameChan:
				r.Lock()
				if r.state == RECORDER_RUNNING { // append to buffer
					r.frames = append(r.frames, frame)
					r.delays = append(r.delays, 10)
				}
				r.Unlock()
			default:
				time.Sleep(1 * time.Millisecond)
			}
		}
	}(r)
}

// Start recording of abyssal run
func (r *Recorder) Start(characters []string) {
	r.Lock()
	defer r.Unlock()

	// make sure recordings folder exists
	_, err := os.Stat(r.config.Recordings)
	if os.IsNotExist(err) {
		err = os.MkdirAll(r.config.Recordings, os.ModeDir)
		if err != nil {
			log.Fatalf("could not create folder: %s", r.config.Recordings)
		}
	}

	combatLogFiles, _ := r.combatlogReader.GetLogFiles(time.Now(), 24*time.Hour)
	r.charactersTracking = r.combatlogReader.MapCharactersToFiles(combatLogFiles)

	// remove log files from tracking
	for char := range r.charactersTracking {
		found := false
		for _, selected := range characters {
			if char == selected {
				found = true
				break
			}
		}
		if !found {
			delete(r.charactersTracking, char)
		}
	}
	log.Printf("recording characters: %+v\n", r.charactersTracking)
	r.combatlogReader.MarkStartOffsets(r.charactersTracking)

	r.recordingName = filepath.Join(r.config.Recordings, fmt.Sprintf("%s.abyss", time.Now().Format("2006-Jan-2-15-04-05")))
	r.frames = make([]*image.Paletted, 0)
	r.delays = make([]int, 0)
	r.lootRecords = make([]*encoding.LootRecord, 0)
	r.state = RECORDER_AWAITING_INITIAL_LOOT
	r.notificationChannel <- NotificationMessage{Title: "Recording starting...", Message: "CTRL+A, CTRL+C your inventory"}
}

// Stop stops recording and writes .abyss file if frames captured
func (r *Recorder) Stop() error {
	r.Lock()
	defer r.Unlock()

	if len(r.frames) == 0 {
		r.state = RECORDER_STOPPED
		return fmt.Errorf("There was no frames captured, skipping recording of abyss run")
	}

	file, _ := os.Create(r.recordingName)
	defer file.Close()

	defer log.Printf("Recording %d frames, written to file: %s", len(r.frames), r.recordingName)

	var buf bytes.Buffer

	r.state = RECORDER_STOPPED
	anim := gif.GIF{Delay: r.delays, LoopCount: -1, Image: r.frames}
	err := gif.EncodeAll(&buf, &anim)
	if err != nil {
		return err
	}

	defer func() {
		r.notificationChannel <- NotificationMessage{Title: "Abyss recorder", Message: fmt.Sprintf("Abyss run succesfully recorded to file: %s", r.recordingName)}
	}()

	defer func() {
		// let GC collect memory allocated for recording
		r.frames = []*image.Paletted{}
		r.lootRecords = []*encoding.LootRecord{}
	}()

	abyssFile := encoding.AbyssRecording{Overview: buf.Bytes(), Loot: r.lootRecords, CombatLog: r.combatlogReader.GetCombatLogRecords(r.charactersTracking)}
	return abyssFile.Encode(file)
}

// StopLoop stops main recording loop
func (r *Recorder) StopLoop() {
	r.done <- true
}

// Status returns status of recorder
func (r *Recorder) Status() int {
	r.Lock()
	defer r.Unlock()
	return r.state
}
