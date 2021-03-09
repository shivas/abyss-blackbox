package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/shivas/abyss-blackbox/encoding"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: extract recording.abyss")
		os.Exit(1)
	}

	file, err := os.Open(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	abyssFile, err := encoding.Decode(file)
	if err != nil {
		log.Fatal(err)
	}

	gifName := filepath.Base(os.Args[1]) + ".gif"

	gifFile, err := os.Create(gifName)
	if err != nil {
		log.Fatal(err)
	}
	defer gifFile.Close()

	_, err = io.Copy(gifFile, bytes.NewReader(abyssFile.Overview))
	if err != nil {
		log.Fatal(err)
	}

	for _, logRecord := range abyssFile.CombatLog {
		fmt.Printf("combat log record language for character %q: %s\n", logRecord.CharacterName, logRecord.GetLanguageCode().String())
		f, err := os.Create(logRecord.CharacterName + ".combatlog.txt")
		if err != nil {
			log.Println(err)
		}

		for _, l := range logRecord.CombatLogLines {
			_, err = f.WriteString(l + "\n")
			if err != nil {
				log.Println(err)
			}
		}
		f.Close()
	}

	f, err := os.Create(os.Args[1] + ".loot.txt")
	if err != nil {
		log.Println(err)
	}

	fmt.Fprintf(f, "Loot recordings:")
	for _, lootRecord := range abyssFile.Loot {
		fmt.Fprintf(f, "time: %s\n%s\n\n", time.Duration(time.Duration(lootRecord.Frame)*time.Second).String(), lootRecord.Loot)
	}
	f.Close()
}
