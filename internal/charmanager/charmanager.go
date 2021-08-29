package charmanager

//go:generate protoc -I ../../protobuf/ --go_opt=module=github.com/shivas/abyss-blackbox --go_out=.. characters-cache.proto

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/png"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"sync"

	"github.com/lxn/walk"
	"github.com/pkg/browser"
	"github.com/shivas/abyss-blackbox/internal/mainwindow"
	"google.golang.org/protobuf/proto"
)

const addCharacterActionIndex = 2

type Character struct {
	CharacterID           int32  `json:"characterID"`
	CharacterName         string `json:"characterName"`
	CharacterToken        string
	CharacterPortraitIcon []byte
}

type CharManager struct {
	sync.RWMutex
	tokenFetcher        backgroundTokenFetcher
	characters          map[int32]Character
	activeCharacter     *Character
	mainWindow          *mainwindow.AbyssRecorderWindow
	notify              func(string, string)
	OnActivateCharacter OnCharacterActivatedHandler
}

type OnCharacterActivatedHandler = func(Character)

func New(notifier func(string, string)) *CharManager {
	return &CharManager{tokenFetcher: backgroundTokenFetcher{}, characters: map[int32]Character{}, activeCharacter: nil, mainWindow: nil, notify: notifier, OnActivateCharacter: func(c Character) {}}
}

func (c *CharManager) MainWindow(window *mainwindow.AbyssRecorderWindow) *CharManager {
	c.Lock()
	defer c.Unlock()
	c.mainWindow = window

	return c
}

func (c *CharManager) SetActiveCharacter(charID int32) error {
	c.Lock()
	defer c.Unlock()

	char, ok := c.characters[charID]
	if !ok {
		c.activeCharacter = nil
		c.OnActivateCharacter(Character{})

		return errors.New("character not linked")
	}

	c.activeCharacter = &char
	c.OnActivateCharacter(char)

	return nil
}

const cacheFileName = "characters.cache"

// PersistCache stores current state to cache file
func (c *CharManager) PersistCache() error {
	c.Lock()
	defer c.Unlock()

	var err error

	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	file := path.Join(wd, cacheFileName)

	var cache CharacterCache
	cache.Characters = make([]*CharacterRecord, 0)

	for _, c := range c.characters {
		cache.Characters = append(cache.Characters, &CharacterRecord{
			CharacterID:   c.CharacterID,
			CharacterName: c.CharacterName,
			Token:         c.CharacterToken,
			Portrait:      c.CharacterPortraitIcon,
		})
	}

	data, err := proto.Marshal(&cache)
	if err != nil {
		return err
	}

	err = os.WriteFile(file, data, 0600)

	return err
}

// LoadCache loads records of characters from cache file if it exists.
func (c *CharManager) LoadCache() error {
	c.Lock()
	defer c.Unlock()

	var err error

	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	file := path.Join(wd, cacheFileName)

	f, err := os.Open(file)
	if err != nil {
		return err
	}

	defer f.Close()

	data, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}

	var cache CharacterCache

	err = proto.Unmarshal(data, &cache)
	if err != nil {
		return err
	}

	for _, char := range cache.Characters {
		c.characters[char.CharacterID] = Character{
			CharacterID:           char.CharacterID,
			CharacterName:         char.CharacterName,
			CharacterToken:        char.Token,
			CharacterPortraitIcon: char.Portrait,
		}
	}

	return err
}

// AddCharacterFromToken used as callback to add character to state from JWT token.
func (c *CharManager) AddCharacterFromToken(token string) {
	_ = c.mainWindow.Toolbar.Actions().At(addCharacterActionIndex).SetEnabled(true)

	if token == "" { // empty token nothing to add
		return
	}

	req, err := http.NewRequest("GET", "https://abyssal.space/auth/ping", nil)
	if err != nil {
		log.Printf("failed to construct ping request: %v\n", err)
	}

	req.Header.Add("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("failed to ping auth: %v\n", err)
	}

	defer resp.Body.Close()

	var char Character

	err = json.NewDecoder(resp.Body).Decode(&char)
	if err != nil {
		log.Printf("failed to unmarshal ping response: %v\n", err)
	}

	if char.CharacterID == 0 || char.CharacterName == "" {
		return
	}

	char.CharacterToken = token
	portrait, _ := downloadCharacterPortrait(char.CharacterID, 32)
	char.CharacterPortraitIcon = portrait

	c.Lock()
	c.characters[char.CharacterID] = char
	c.Unlock()

	_ = c.SetActiveCharacter(char.CharacterID)
	c.RefreshUI()
	_ = c.PersistCache()
}

// removeCharacter removes character from state, sets random character remaining as active.
func (c *CharManager) removeCharacter(charID int32) {
	c.Lock()

	if charID != 0 {
		char, ok := c.characters[charID]
		if ok {
			c.notify("Character removed:", char.CharacterName)
		}
	}

	delete(c.characters, charID)
	c.Unlock()

	_ = c.SetActiveCharacter(0)
	_ = c.PersistCache()

	if len(c.characters) > 0 {
		for a := range c.characters {
			_ = c.SetActiveCharacter(a)
		}
	}

	c.RefreshUI()
}

// RefreshUI resyncs UI from state.
func (c *CharManager) RefreshUI() {
	c.Lock()
	defer c.Unlock()

	if c.mainWindow == nil {
		return
	}

	_ = c.mainWindow.CharacterSwitcherMenu.Actions().Clear()

	for _, char := range c.characters {
		var char = char

		action := walk.NewAction()
		_ = action.SetText(char.CharacterName)
		_ = action.SetImage(char.PortraitIcon())
		_ = action.Triggered().Attach(func() {
			modifiers := walk.ModifiersDown()
			if modifiers == walk.ModControl {
				c.removeCharacter(char.CharacterID)
			} else {
				_ = c.SetActiveCharacter(char.CharacterID)
			}
		})

		_ = c.mainWindow.CharacterSwitcherMenu.Actions().Add(action)
	}

	if len(c.characters) > 0 {
		_ = c.mainWindow.Toolbar.Actions().At(0).SetEnabled(true)
	} else {
		_ = c.mainWindow.Toolbar.Actions().At(0).SetEnabled(false)
	}
}

func (c *Character) PortraitIcon() *walk.Icon {
	img, _, err := image.Decode(bytes.NewReader(c.CharacterPortraitIcon))
	if err != nil {
		return &walk.Icon{}
	}

	icon, err := walk.NewIconFromImageForDPI(img, 92)
	if err != nil {
		return &walk.Icon{}
	}

	return icon
}

func downloadCharacterPortrait(charID int32, size int) ([]byte, error) {
	url := fmt.Sprintf("https://images.evetech.net/characters/%d/portrait?size=%d", charID, size)

	resp, err := http.DefaultClient.Get(url)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	return ioutil.ReadAll(resp.Body)
}

func (c *CharManager) ActiveCharacter() *Character {
	c.Lock()
	defer c.Unlock()

	return c.activeCharacter
}

func (c *CharManager) EventHandlerCharAdd() {
	log.Print("clicked add character")

	_ = c.mainWindow.Toolbar.Actions().At(addCharacterActionIndex).SetEnabled(false)

	req, err := http.NewRequest("GET", "https://abyssal.space/auth/session", nil)
	if err != nil {
		log.Printf("failed creating request: %v\n", err)
		return
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("failed response for session: %v\n", err)
		return
	}

	defer resp.Body.Close()

	result := struct {
		SessionID   string `json:"sessionID"`
		RedirectURL string `json:"redirectURL"`
	}{}

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		log.Printf("session response is not unserializable: %v\n", err)
		return
	}

	err = browser.OpenURL(result.RedirectURL)
	if err != nil {
		log.Printf("failed opening browser: %v\n", err)
		return
	}

	c.tokenFetcher.run(context.Background(), result.SessionID, c.AddCharacterFromToken)

	log.Printf("session link: %v\n", result.RedirectURL)
}
