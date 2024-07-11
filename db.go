package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

func MustInitDB(root string) {
	err := os.Mkdir(".meta", os.FileMode(0755))
	if err != nil {
		if !errors.Is(err, os.ErrExist) {
			log.Fatalf("failed creating root directory %s -> %s\n", ".meta", err)
		}
	}
}

func CheckEtag(path string) (string, error) {
	path = "../.meta/" + path
	if !strings.HasSuffix(path, "/") {
		path += "/"
	}

	etag, err := os.ReadFile(path + "tag.txt")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", fmt.Errorf("error opening file on read %s -> %v", path, err)
	}

	return string(etag), nil
}

func SaveEtag(path string, etag string) error {
	path = "../.meta/" + path
	err := os.MkdirAll(path, os.FileMode(0755))
	if err != nil {
		if !errors.Is(err, os.ErrExist) {
			return fmt.Errorf("error creating folder on save %s -> %v", path, err)
		}
	}

	if !strings.HasSuffix(path, "/") {
		path += "/"
	}

	file, err := os.OpenFile(path+"tag.txt", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(0644))
	if err != nil {
		return fmt.Errorf("error opening file on save %s -> %v", path, err)
	}

	file.WriteString(etag)
	file.Close()

	file, err = os.OpenFile(path+"meta.txt", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(0644))
	if err != nil {
		return fmt.Errorf("error opening file on save %s -> %v", path, err)
	}

	file.WriteString(time.Now().String())
	file.Close()

	return nil
}
