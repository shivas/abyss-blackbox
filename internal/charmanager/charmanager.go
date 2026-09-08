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
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sync"

	"github.com/pkg/browser"
	"google.golang.org/protobuf/proto"
)

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
	notify              func(string, string)
	OnActivateCharacter OnCharacterActivatedHandler
	OnRefresh           func()
}

type OnCharacterActivatedHandler = func(Character)

func New(notifier func(string, string)) *CharManager {
	return &CharManager{
		tokenFetcher:        backgroundTokenFetcher{},
		characters:          map[int32]Character{},
		notify:              notifier,
		OnActivateCharacter: func(Character) {},
		OnRefresh:           func() {},
	}
}

func (c *CharManager) SetActiveCharacter(charID int32) error {
	c.Lock()
	char, ok := c.characters[charID]
	if !ok {
		c.activeCharacter = nil
		c.Unlock()
		c.OnActivateCharacter(Character{})

		return errors.New("character not linked")
	}

	c.activeCharacter = &char
	activated := char
	c.Unlock()
	c.OnActivateCharacter(activated)

	return nil
}

const cacheFileName = "characters.cache"

func (c *CharManager) PersistCache() error {
	c.Lock()
	defer c.Unlock()

	wd := filepath.Dir(os.Args[0])
	file := path.Join(wd, cacheFileName)

	var cache CharacterCache
	cache.Characters = make([]*CharacterRecord, 0)

	for _, ch := range c.characters {
		cache.Characters = append(cache.Characters, &CharacterRecord{
			CharacterID:   ch.CharacterID,
			CharacterName: ch.CharacterName,
			Token:         ch.CharacterToken,
			Portrait:      ch.CharacterPortraitIcon,
		})
	}

	data, err := proto.Marshal(&cache)
	if err != nil {
		return err
	}

	return os.WriteFile(file, data, 0o600)
}

func (c *CharManager) LoadCache() error {
	c.Lock()
	defer c.Unlock()

	wd := filepath.Dir(os.Args[0])
	file := path.Join(wd, cacheFileName)

	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return err
	}

	var cache CharacterCache
	if err = proto.Unmarshal(data, &cache); err != nil {
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

	return nil
}

func (c *CharManager) AddCharacterFromToken(token string) {
	defer c.refresh()

	if token == "" {
		return
	}

	req, err := http.NewRequest(http.MethodGet, "https://abyssal.space/auth/ping", nil)
	if err != nil {
		log.Printf("failed to construct ping request: %v\n", err)
		return
	}

	req.Header.Add("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("failed to ping auth: %v\n", err)
		return
	}
	defer resp.Body.Close()

	var char Character
	if err = json.NewDecoder(resp.Body).Decode(&char); err != nil {
		log.Printf("failed to unmarshal ping response: %v\n", err)
		return
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
	_ = c.PersistCache()
}

func (c *CharManager) RemoveCharacter(charID int32) {
	c.Lock()
	if charID != 0 {
		if char, ok := c.characters[charID]; ok {
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
			break
		}
	}

	c.refresh()
}

func (c *CharManager) Characters() []Character {
	c.RLock()
	defer c.RUnlock()

	out := make([]Character, 0, len(c.characters))
	for _, ch := range c.characters {
		out = append(out, ch)
	}

	return out
}

func (c *Character) PortraitImage() image.Image {
	img, _, err := image.Decode(bytes.NewReader(c.CharacterPortraitIcon))
	if err != nil {
		return image.NewRGBA(image.Rect(0, 0, 32, 32))
	}

	return img
}

func downloadCharacterPortrait(charID int32, size int) ([]byte, error) {
	url := fmt.Sprintf("https://images.evetech.net/characters/%d/portrait?size=%d", charID, size)

	resp, err := http.DefaultClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

func (c *CharManager) ActiveCharacter() *Character {
	c.Lock()
	defer c.Unlock()

	return c.activeCharacter
}

func (c *CharManager) GetActiveCharacterToken(_ context.Context) string {
	c.Lock()
	defer c.Unlock()

	if c.activeCharacter == nil {
		return ""
	}

	return c.activeCharacter.CharacterToken
}

func (c *CharManager) EventHandlerCharAdd() {
	log.Print("clicked add character")

	req, err := http.NewRequest(http.MethodGet, "https://abyssal.space/auth/session", nil)
	if err != nil {
		log.Printf("failed creating request: %v\n", err)
		c.refresh()
		return
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("failed response for session: %v\n", err)
		c.refresh()
		return
	}
	defer resp.Body.Close()

	result := struct {
		SessionID   string `json:"sessionID"`
		RedirectURL string `json:"redirectURL"`
	}{}

	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("session response is not unserializable: %v\n", err)
		c.refresh()
		return
	}

	if err = browser.OpenURL(result.RedirectURL); err != nil {
		log.Printf("failed opening browser: %v\n", err)
	}

	c.tokenFetcher.run(context.Background(), result.SessionID, c.AddCharacterFromToken)
	log.Printf("session link: %v\n", result.RedirectURL)
}

func (c *CharManager) refresh() {
	if c.OnRefresh != nil {
		c.OnRefresh()
	}
}
