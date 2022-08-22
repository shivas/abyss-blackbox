package main

import (
	"fmt"
	"log"

	"golang.org/x/sys/windows/registry"
)

func main() {

	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion`, registry.QUERY_VALUE)
	if err != nil {
		log.Fatal(err)
	}
	defer k.Close()

	cv, _, err := k.GetStringValue("CurrentVersion")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("CurrentVersion: %s\n", cv)

	pn, _, err := k.GetStringValue("ProductName")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("ProductName: %s\n", pn)

	maj, _, err := k.GetIntegerValue("CurrentMajorVersionNumber")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("CurrentMajorVersionNumber: %d\n", maj)

	min, _, err := k.GetIntegerValue("CurrentMinorVersionNumber")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("CurrentMinorVersionNumber: %d\n", min)

	cb, _, err := k.GetStringValue("CurrentBuild")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("CurrentVersion: %s\n", cb)
}
