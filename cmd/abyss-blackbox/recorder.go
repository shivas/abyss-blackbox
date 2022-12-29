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
	"github.com/shivas/abyss-blackbox/internal/fittings"
	"github.com/shivas/abyss-blackbox/internal/overlay"
	"github.com/shivas/abyss-blackbox/internal/version"
)

const (
	RecorderStopped = iota
	RecorderRunning
	RecorderAwaitingInitialLoot
)

type Recorder struct {
	mutex               sync.Mutex
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
	overlay             *overlay.Overlay
}

// NewRecorder constructs Recorder
func NewRecorder(frameChan chan *image.Paletted, c *config.CaptureConfig, nc chan NotificationMessage, clr *combatlog.Reader, overlay *overlay.Overlay) *Recorder {
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
		overlay:             overlay,
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

		r.mutex.Lock()
		defer r.mutex.Unlock()
		r.weatherStrength = strength
		r.notificationChannel <- NotificationMessage{Title: "Abyssal.Space recorder", Message: fmt.Sprintf("Weather strength set to: %d%%", strength)}
		r.overlay.ChangeProperty(overlay.Weather, fmt.Sprintf("Weather strength set to: %d%%", strength), &overlay.GreenColor)
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
				r.mutex.Lock()

				switch r.state {
				case RecorderAwaitingInitialLoot:
					r.notificationChannel <- NotificationMessage{Title: "Abyssal.Space recording started...", Message: "Initial cargo received, awaiting cargo after fillament activation"}
					r.overlay.ChangeProperty(overlay.TODO, "Activate fillament. Record loot after!", &overlay.RedColor)
					r.overlay.ChangeProperty(overlay.Status, "Recording...", &overlay.GreenColor)
					r.state = RecorderRunning
					r.lootRecords = append(r.lootRecords, &encoding.LootRecord{Frame: 0, Loot: lootSnapshot})
				case RecorderRunning:
					r.notificationChannel <- NotificationMessage{Title: "Abyssal.Space recorder", Message: "Loot captured from clipboard!"}
					r.overlay.ChangeProperty(overlay.Status, "Recording...", &overlay.GreenColor)
					r.overlay.ChangeProperty(overlay.TODO, "Loot captured from clipboard!", &overlay.YellowColor)

					go func() {
						time.Sleep(5 * time.Second)
						r.overlay.ChangeProperty(overlay.TODO, "", nil)
					}()

					lr := &encoding.LootRecord{Frame: int32(len(r.frames) - 1), Loot: lootSnapshot}

					log.Printf("loot appended: %v\n", lr)
					r.lootRecords = append(r.lootRecords, lr)
				default:
					log.Printf("dropped loot record, %v", lootSnapshot)
				}

				r.mutex.Unlock()

			case frame := <-r.frameChan:
				r.mutex.Lock()
				if r.state == RecorderRunning { // append to buffer
					r.frames = append(r.frames, frame)
					r.delays = append(r.delays, 10)

					if r.weatherStrength == 0 && (len(r.frames)%180 == 0) { // remind every 3 minutes skipping initial frame
						r.notificationChannel <- NotificationMessage{"Reminder", "Please record weather strength!"}
						r.overlay.ChangeProperty(overlay.Weather, "Please record weather strength!", &overlay.SecondaryColor)
					}
				}
				r.mutex.Unlock()
			default:
				time.Sleep(1 * time.Millisecond)
			}
		}
	}(r)
}

// Start recording of abyssal run
func (r *Recorder) Start(characters []string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

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
	r.overlay.ChangeProperty(overlay.Status, "Recording starting", &overlay.CyanColor)
	r.overlay.ChangeProperty(overlay.Weather, "TODO: Record weather strength", nil)
	r.overlay.ChangeProperty(overlay.TODO, "TODO: CTRL+A, CTRL+C your inventory", nil)
}

// Stop stops recording and writes .abyss file if frames captured
func (r *Recorder) Stop(fm *fittings.FittingsManager) (string, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if len(r.frames) == 0 {
		r.state = RecorderStopped
		return r.recordingName, fmt.Errorf("there was no frames captured, skipping recording of abyss run")
	}

	runFittings := make(map[string]*encoding.Fit, len(r.charactersTracking))

	for char := range r.charactersTracking {
		fit := fm.GetFittingForPilot(char)
		if fit != nil {
			runFittings[char] = &encoding.Fit{
				Source:      fit.Source,
				ForeignID:   fit.ForeignID,
				FittingName: fit.FittingName,
				EFT:         fit.EFT,
				FFH:         fit.FFH,
				Price:       fit.Price,
				ShipName:    fit.ShipName,
				ShipTypeID:  fit.ShipTypeID,
			}
		}
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
		r.overlay.ChangeProperty(overlay.TODO, "Abyss run successfully recorded to file", &overlay.GreenColor)
		r.overlay.ChangeProperty(overlay.Status, "Recorder on standby", &overlay.YellowColor)
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
		RecorderVersion:         version.RecorderVersion,
		ManualAbyssTypeOverride: r.config.AbyssTypeOverride,
		Fittings:                runFittings,
	}

	if r.config.AbyssTypeOverride {
		abyssFile.AbyssShipType = encoding.AbyssRecording_AbyssShipType(r.config.AbyssShipType)
		abyssFile.AbyssTier = int32(r.config.AbyssTier)
		abyssFile.AbyssWheather = r.config.AbyssWeather
	}

	return r.recordingName, abyssFile.Encode(file)
}

// StopLoop stops main recording loop
func (r *Recorder) StopLoop() {
	r.done <- true
}

// Status returns status of recorder
func (r *Recorder) Status() int {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	return r.state
}
