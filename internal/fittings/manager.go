package fittings

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sync"

	"github.com/shivas/abyss-blackbox/internal/fittings/pb"
	"github.com/shivas/abyss-blackbox/internal/fittings/provider"

	"google.golang.org/protobuf/proto"
)

//go:generate protoc -I ../../protobuf/ --go_opt=module=github.com/shivas/abyss-blackbox/internal/fittings/pb --go_out=./pb fittings-cache.proto

const (
	cacheFileName = "fittings.db"
	iconSize      = 32
	version       = 1
)

type FittingsManager struct {
	sync.Mutex
	cache           pb.FittingsCache
	importProviders []provider.FittingsProvider
	httpClient      *http.Client
}

func NewManager(httpClient *http.Client, providers ...provider.FittingsProvider) *FittingsManager {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	m := &FittingsManager{
		httpClient: httpClient,
	}

	m.cache.Version = version
	m.importProviders = providers
	_ = m.LoadCache()

	return m
}

func (m *FittingsManager) ClearAssignments() {
	m.Lock()
	defer m.Unlock()
	m.cache.CharactersFittings = map[string]*pb.FittingRecord{}
}

func (m *FittingsManager) AvailableProviders(ctx context.Context) map[string]provider.AvailabilityResult {
	available := make(map[string]provider.AvailabilityResult)

	for _, p := range m.importProviders {
		available[p.SourceName()] = p.Available(ctx)
	}

	return available
}

func (m *FittingsManager) FetchFittingIDs(ctx context.Context, provider string) []string {
	for _, p := range m.importProviders {
		if p.SourceName() == provider {
			return p.AvailableFittingIDs(ctx)
		}
	}

	return nil
}

func (m *FittingsManager) ImportFittings(ctx context.Context, source string, callback func(current, max int)) error {
	var provider provider.FittingsProvider

	for _, p := range m.importProviders {
		if p.SourceName() != source {
			continue
		}

		if !p.Available(ctx).Available {
			return fmt.Errorf("source is not available")
		}
		provider = p
		break
	}

	fits := provider.AvailableFittingIDs(ctx)
	current := 1

	for _, fid := range fits {
		fit, err := provider.GetFittingDetails(ctx, fid)
		if err != nil {
			return err
		}

		fID := fid

		f := pb.FittingRecord{
			Source:      source,
			ForeignID:   &fID,
			FittingName: fit.Item.Name,
			EFT:         fit.Item.EFT,
			FFH:         fit.Item.FFH,
			ShipTypeID:  int32(fit.Item.Ship.ID),
			ShipName:    fit.Item.Ship.Name,
		}

		m.AddFitting(&f)
		callback(current, len(fits))
		current++
	}

	return nil
}

func (m *FittingsManager) DeleteFitting(index int) {
	m.Lock()
	defer m.Unlock()

	m.cache.Fittings = append(m.cache.Fittings[:index], m.cache.Fittings[index+1:]...)
}

func (m *FittingsManager) GetFittingForPilot(characterName string) *pb.FittingRecord {
	return m.cache.CharactersFittings[characterName]
}

func (m *FittingsManager) AddFitting(r *pb.FittingRecord) (ID int, fitting *pb.FittingRecord, err error) {
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

func (m *FittingsManager) GetByID(ID int) *pb.FittingRecord {
	m.Lock()
	defer m.Unlock()

	if ID < 0 || ID > len(m.cache.Fittings)-1 {
		return nil
	}

	return m.cache.Fittings[ID]
}

func (m *FittingsManager) AssignFittingToCharacter(f *pb.FittingRecord, characterName string) {
	if m.cache.CharactersFittings == nil {
		m.cache.CharactersFittings = make(map[string]*pb.FittingRecord)
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
