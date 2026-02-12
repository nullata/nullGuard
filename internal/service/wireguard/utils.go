// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package utils

import (
	"bytes"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"unicode"
)

// writes the given content to a file, overwriting any existing content
func WriteToFile(targetFullFilePath, content string, permissions fs.FileMode) error {
	// check if the file already exists
	if _, err := os.Stat(targetFullFilePath); err == nil {
		log.Printf("Configuration file %s already exists and will be overwritten", targetFullFilePath)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("error accessing file %s: %w", targetFullFilePath, err)
	}

	// write the content to the file, overwriting any existing content
	err := os.WriteFile(targetFullFilePath, []byte(content), permissions)
	if err != nil {
		return fmt.Errorf("error writing to file %s: %w", targetFullFilePath, err)
	}
	log.Printf("Successfully created configuration file: %s", targetFullFilePath)
	return nil
}

// deletes the specified file and returns an error if any occurs
func DeleteFile(filePath string) error {
	err := os.Remove(filePath)
	if err != nil {
		return fmt.Errorf("failed to delete file %s: %v", filePath, err)
	}
	return nil
}

func IsDirectory(path string) (bool, error) {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil // path does not exist, so it's not a directory
	}
	if err != nil {
		return false, err // other error occurred, return it
	}
	return info.IsDir(), nil // check if the path is a directory
}

// checks if the specified file or directory exists
func FileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return false
	}
	return err == nil
}

func capitalizeFirstLetterUnicode(s string) string {
	if len(s) == 0 {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

func GenerateWgKeys(keyType string) (string, string, error) {
	// generate the private key
	cmd := exec.Command("wg", "genkey")
	privateKey, err := cmd.Output()
	if err != nil {
		log.Printf("Error generating %s private key: %v", keyType, err)
		return "", "", err
	}

	// generate the public key using the private key
	cmdPub := exec.Command("wg", "pubkey")

	// pass the private key via stdin buffer
	cmdPub.Stdin = bytes.NewReader(privateKey)

	// capture the public key output
	var publicKey bytes.Buffer
	cmdPub.Stdout = &publicKey

	if err := cmdPub.Run(); err != nil {
		log.Printf("Error generating %s public key: %v", keyType, err)
		return "", "", err
	}

	log.Printf("%s keys generated successfully", capitalizeFirstLetterUnicode(keyType))
	return string(privateKey), publicKey.String(), nil
}
