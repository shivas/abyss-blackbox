package combatlog

import (
	"bufio"
	"os"
	"path/filepath"
	"time"
)

const DateFormat = "2006.01.02 15:04:05"

type CombatLogFile struct {
	Filename     string
	LanguageCode LanguageCode
}

type Reader struct {
	logDir       string
	startOffsets map[string]os.FileInfo
}

func NewReader(path string) *Reader {
	return &Reader{logDir: path, startOffsets: make(map[string]os.FileInfo)}
}

func (r *Reader) SetLogFolder(folder string) {
	r.logDir = folder
}

// GetLogFiles returns slice of filepaths that logged in in last timeWindow
func (r *Reader) GetLogFiles(end time.Time, timeWindow time.Duration) ([]string, error) {

	prefixes := genPrefixes(end, timeWindow)

	logFiles := []string{}

	for _, prefix := range prefixes {
		matches, err := filepath.Glob(filepath.Join(r.logDir, prefix+"*.txt"))
		if err != nil {
			return nil, err
		}
		logFiles = append(logFiles, matches...)
	}

	return logFiles, nil
}

// GetCombatLogRecords reads combatlog from stored offsets and converts to CombatLogRecord struct
func (r *Reader) GetCombatLogRecords(characters map[string]CombatLogFile) []*CombatLogRecord {
	recordings := make([]*CombatLogRecord, 0)
	for character, logfile := range characters {

		startFileInfo, marked := r.startOffsets[character]
		if !marked {
			continue
		}

		file, err := os.Open(logfile.Filename)
		if err != nil {
			continue
		}
		defer file.Close()

		clr := &CombatLogRecord{CharacterName: character, CombatLogLines: []string{}, LanguageCode: logfile.LanguageCode}

		_, err = file.Seek(startFileInfo.Size(), 0)
		if err != nil {
			continue
		}

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			clr.CombatLogLines = append(clr.CombatLogLines, scanner.Text())
		}
		recordings = append(recordings, clr)
	}

	r.startOffsets = make(map[string]os.FileInfo)

	return recordings
}

// MarkStartOffsets stores offsets of combatlog files
func (r *Reader) MarkStartOffsets(characters map[string]CombatLogFile) {
	for character, logfile := range characters {

		file, err := os.Open(logfile.Filename)
		if err != nil {
			continue
		}
		defer file.Close()

		fi, err := file.Stat()
		if err != nil {
			continue
		}
		r.startOffsets[character] = fi
	}
}

// MapCharactersToFiles maps given paths to characters, detecting combatlog language in process
func (r *Reader) MapCharactersToFiles(files []string) map[string]CombatLogFile {

	type f struct {
		sessionStarted *time.Time
		filename       string
		character      *string
		languageCode   LanguageCode
	}

	tempMap := make(map[string]f)

	for _, filename := range files {
		file, err := os.Open(filename)
		if err != nil {
			continue
		}
		defer file.Close()

		reader := bufio.NewScanner(file)
		var insideBanner bool

		logFile := f{filename: filename}

		for reader.Scan() {
			if reader.Text() == "------------------------------------------------------------" {
				if !insideBanner {
					insideBanner = true
					continue
				}
				if insideBanner {
					break // reading closing header separator, just break
				}
			}

			if insideBanner {
				listenerText := reader.Text()
				sessionText := reader.Text()
				for languageCode, matchers := range LanguageMatchers {
					if matches := matchers.ListenerRe.FindAllStringSubmatch(listenerText, 1); matches != nil {
						logFile.character = &matches[0][1]
						logFile.languageCode = languageCode
					}

					if matches := matchers.SessionStartRe.FindAllStringSubmatch(sessionText, 1); matches != nil {
						sessionStart, err := time.Parse(DateFormat, matches[0][1])
						if err != nil {
							continue
						}
						logFile.sessionStarted = &sessionStart
					}
				}
			}

			// if we have all we need from this file exit scan loop
			if logFile.character != nil && logFile.sessionStarted != nil {
				val, ok := tempMap[*logFile.character]
				if ok {
					if val.sessionStarted.Before(*logFile.sessionStarted) {
						tempMap[*logFile.character] = logFile
					}
				} else {
					tempMap[*logFile.character] = logFile
				}
				break
			}
		}

	}

	result := make(map[string]CombatLogFile, len(tempMap))
	for character, logfile := range tempMap {
		result[character] = CombatLogFile{Filename: logfile.filename, LanguageCode: logfile.languageCode}
	}

	return result
}

func genPrefixes(end time.Time, timeWindow time.Duration) []string {
	begin := end.Add(-1*timeWindow - time.Second)

	filePrefixes := []string{}

	for end.After(begin) {
		filePrefixes = append(filePrefixes, begin.UTC().Format("20060102"))
		begin = begin.Add(24 * time.Hour)
	}

	return filePrefixes
}
