package fittings

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sync"

	"google.golang.org/protobuf/proto"
)

//go:generate protoc -I ../../protobuf/ --go_opt=module=github.com/shivas/abyss-blackbox/internal/fittings --go_out=. fittings-cache.proto

const (
	cacheFileName = "fittings.db"
	iconSize      = 32
	version       = 1
)

type FittingsManager struct {
	sync.Mutex
	cache FittingsCache
}

func NewManager() *FittingsManager {
	m := &FittingsManager{}
	m.cache.Version = version
	_ = m.LoadCache()
	return m
}

func (m *FittingsManager) ClearAssignments() {
	m.Lock()
	defer m.Unlock()
	m.cache.CharactersFittings = map[string]*FittingRecord{}
}

func (m *FittingsManager) DeleteFitting(index int) {
	m.Lock()
	defer m.Unlock()

	m.cache.Fittings = append(m.cache.Fittings[:index], m.cache.Fittings[index+1:]...)
}

func (m *FittingsManager) GetFittingForPilot(characterName string) *FittingRecord {
	return m.cache.CharactersFittings[characterName]
}

func (m *FittingsManager) AddFitting(r *FittingRecord) (ID int, fitting *FittingRecord, err error) {
	m.Lock()
	defer m.Unlock()

	ID = -1

	if err = r.Validate(); err != nil {
		return
	}

	r.Icon, err = downloadTypeIDIcon(r.ShipTypeID, iconSize)
	if err != nil {
		return
	}

	m.cache.Fittings = append(m.cache.Fittings, r)

	return len(m.cache.Fittings), r, nil
}

func (m *FittingsManager) GetByID(ID int) *FittingRecord {
	m.Lock()
	defer m.Unlock()

	if ID < 0 || ID > len(m.cache.Fittings)-1 {
		return nil
	}

	return m.cache.Fittings[ID]
}

func (m *FittingsManager) AssignFittingToCharacter(f *FittingRecord, characterName string) {
	if m.cache.CharactersFittings == nil {
		m.cache.CharactersFittings = make(map[string]*FittingRecord)
	}

	m.cache.CharactersFittings[characterName] = f
}

// PersistCache stores current state to cache file
func (m *FittingsManager) PersistCache() error {
	m.Lock()
	defer m.Unlock()

	var err error

	wd := filepath.Dir(os.Args[0])
	file := path.Join(wd, cacheFileName)

	data, err := proto.Marshal(&m.cache)
	if err != nil {
		return err
	}

	err = os.WriteFile(file, data, 0o600)

	return err
}

// LoadCache loads records of fittings from cache file if it exists.
func (m *FittingsManager) LoadCache() error {
	m.Lock()
	defer m.Unlock()

	var err error

	wd := filepath.Dir(os.Args[0])
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

	err = proto.Unmarshal(data, &m.cache)
	if err != nil {
		return err
	}

	return err
}

func downloadTypeIDIcon(typeID int32, size int) ([]byte, error) {
	url := fmt.Sprintf("https://images.evetech.net/types/%d/icon?size=%d", typeID, size)

	resp, err := http.DefaultClient.Get(url)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	return ioutil.ReadAll(resp.Body)
}
