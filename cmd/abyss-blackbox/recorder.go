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
	"github.com/shivas/abyss-blackbox/internal/config"
)

const (
	RecorderStopped = iota
	RecorderRunning
	RecorderAwaitingInitialLoot
)

type Recorder struct {
	sync.Mutex
	state               int
	frameChan           chan *image.Paletted
	loot                chan string
	config              *config.CaptureConfig
	done                chan bool
	frames              []*image.Paletted
	delays              []int
	recordingName       string
	lootRecords         []*encoding.LootRecord
	notificationChannel chan NotificationMessage
	combatlogReader     *combatlog.Reader
	charactersTracking  map[string]combatlog.CombatLogFile
	weatherStrength     int
}

// NewRecorder constructs Recorder
func NewRecorder(frameChan chan *image.Paletted, c *config.CaptureConfig, nc chan NotificationMessage, clr *combatlog.Reader) *Recorder {
	return &Recorder{
		frameChan:           frameChan,
		loot:                make(chan string, 2),
		state:               RecorderStopped,
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

// GetWeatherStrengthListener returs listener that will set weather strength when invoked. In not running state it is NOOP.
func (r *Recorder) GetWeatherStrengthListener(strength int) func() {
	return func() {
		if r.state != RecorderRunning {
			return
		}

		r.Lock()
		defer r.Unlock()
		r.weatherStrength = strength
		r.notificationChannel <- NotificationMessage{Title: "Abyssal.Space recorder", Message: fmt.Sprintf("Weather strength set to: %d%%", strength)}
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
				case RecorderAwaitingInitialLoot:
					r.notificationChannel <- NotificationMessage{Title: "Abyssal.Space recording started...", Message: "Initial cargo received, awaiting cargo after fillament activation"}
					r.state = RecorderRunning
					r.lootRecords = append(r.lootRecords, &encoding.LootRecord{Frame: 0, Loot: lootSnapshot})
				case RecorderRunning:
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
				if r.state == RecorderRunning { // append to buffer
					r.frames = append(r.frames, frame)
					r.delays = append(r.delays, 10)

					if r.weatherStrength == 0 && (len(r.frames)%180 == 0) { // remind every 3 minutes skipping initial frame
						r.notificationChannel <- NotificationMessage{"Reminder", "Please record weather strength!"}
					}
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
			log.Fatalf("could not create folder: %s", r.config.Recordings) //nolint:gocritic // it's dead jim
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
	r.weatherStrength = 0
	r.state = RecorderAwaitingInitialLoot
	r.notificationChannel <- NotificationMessage{Title: "Recording starting...", Message: "CTRL+A, CTRL+C your inventory"}
}

// Stop stops recording and writes .abyss file if frames captured
func (r *Recorder) Stop() (string, error) {
	r.Lock()
	defer r.Unlock()

	if len(r.frames) == 0 {
		r.state = RecorderStopped
		return r.recordingName, fmt.Errorf("there was no frames captured, skipping recording of abyss run")
	}

	file, _ := os.Create(r.recordingName)
	defer file.Close()

	defer log.Printf("Recording %d frames, written to file: %s", len(r.frames), r.recordingName)

	var buf bytes.Buffer

	r.state = RecorderStopped
	anim := gif.GIF{Delay: r.delays, LoopCount: -1, Image: r.frames}

	err := gif.EncodeAll(&buf, &anim)
	if err != nil {
		return r.recordingName, err
	}

	defer func() {
		r.notificationChannel <- NotificationMessage{Title: "Abyss recorder", Message: fmt.Sprintf("Abyss run successfully recorded to file: %s", r.recordingName)}
	}()

	defer func() {
		// let GC collect memory allocated for recording
		r.frames = []*image.Paletted{}
		r.lootRecords = []*encoding.LootRecord{}
	}()

	abyssFile := encoding.AbyssRecording{
		Overview:                buf.Bytes(),
		Loot:                    r.lootRecords,
		CombatLog:               r.combatlogReader.GetCombatLogRecords(r.charactersTracking),
		TestServer:              r.config.TestServer,
		WeatherStrength:         int32(r.weatherStrength),
		LootRecordDiscriminator: r.config.LootRecordDiscriminator,
	}

	err = abyssFile.Encode(file)
	return r.recordingName, err
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
