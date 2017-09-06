package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"regexp"

	"strings"

	"time"

	logging "github.com/op/go-logging"
)

// Config struct for configuration
type Config struct {
	Directories            string `json:"directories"`
	Extensions             string `json:"extensions"`
	Pattern                string `json:"pattern"`
	DaysOfExpiration       uint   `json:"days_of_expiration"`
	SearchInSubdirectories bool   `json:"search_in_subdirectories"`
}

func (config Config) String() string {
	return fmt.Sprintf("{Directories: %s, Extensions: %s, Pattern: %s, DaysOfExpiration: %d, SearchInSubdirectories: %t}", config.Directories, config.Extensions, config.Pattern, config.DaysOfExpiration, config.SearchInSubdirectories)
}

var log = logging.MustGetLogger("main")

func initLog(writer io.Writer) {
	// Example format string. Everything except the message has a custom color
	// which is dependent on the log level. Many fields have a custom output
	// formatting too, eg. the time returns the hour down to the milli second.
	var format = logging.MustStringFormatter(
		`%{time:2006-01-02 15:04:05.000} %{level:.4s} (%{module}-%{pid}) [%{callpath} %{shortfile}] %{message}`,
	)
	// For demo purposes, create two backend for os.Stderr.
	backend1 := logging.NewLogBackend(writer, "", 0)
	backend2 := logging.NewLogBackend(writer, "", 0)

	// For messages written to backend2 we want to add some additional
	// information to the output, including the used log level and the name of
	// the function.
	backend2Formatter := logging.NewBackendFormatter(backend2, format)

	// Only errors and more severe messages should be sent to backend1
	backend1Leveled := logging.AddModuleLevel(backend1)
	backend1Leveled.SetLevel(logging.ERROR, "")

	// Set the backends to be used.
	logging.SetBackend(backend1Leveled, backend2Formatter)
}

func main() {
	var configuration Config
	fileConfig, err := os.Open("config.json")
	validateError(err, false)
	defer fileConfig.Close()
	err = json.NewDecoder(fileConfig).Decode(&configuration)
	validateError(err, false)
	fileLog, err := os.OpenFile(strings.Join([]string{"files-deleted", time.Now().Local().Format("2006-01-02-15"), "log"}, "."), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	validateError(err, false)
	defer fileLog.Close()
	initLog(fileLog)
	log.Debug("Config: ", configuration)
	directories := strings.Split(configuration.Directories, "|")
	extensions := strings.Split(configuration.Extensions, "|")
	var totalFilesDeleted uint
	for _, directory := range directories {
		existsPath, err := existsPath(directory)
		validateError(err, true)
		if existsPath {
			totalFilesDeleted += deleteFile(directory, configuration.Pattern, extensions, configuration.DaysOfExpiration, configuration.SearchInSubdirectories)
		}
	}
	log.Notice("Total files deleted: ", totalFilesDeleted)
}

func deleteFile(directory string, pattern string, extensions []string, daysOfExpiration uint, searchInSubDirs bool) uint {
	var filesDeleted uint
	files, _ := ioutil.ReadDir(directory)
	for _, file := range files {
		fileCompleteName := directory + string(os.PathSeparator) + file.Name()
		if file.IsDir() && searchInSubDirs {
			filesDeletedSubDir := deleteFile(fileCompleteName, pattern, extensions, daysOfExpiration, searchInSubDirs)
			filesDeleted = filesDeleted + filesDeletedSubDir
			log.Infof("Files deleted in %s: %d", fileCompleteName, filesDeletedSubDir)
		} else if !file.IsDir() {
			if matchStr(file.Name(), pattern) || endsWith(file.Name(), extensions) {
				log.Info("File: ", file.Name())
				log.Info("Last update: ", file.ModTime().Format("2006-01-02"))
				daysOld := uint(time.Since(file.ModTime()).Hours()) / 24
				log.Debug("Days old: ", daysOld)
				if daysOld >= daysOfExpiration {
					err := os.Remove(fileCompleteName)
					if validateError(err, true) {
						filesDeleted = filesDeleted + 1
						log.Info("File deleted: ", fileCompleteName)
					}
				}
			}
		}
	}

	return filesDeleted
}

func endsWith(word string, suffixes []string) bool {
	for _, suffix := range suffixes {
		if strings.HasSuffix(word, suffix) {
			log.Debugf("Word '%s' has suffix '%s'", word, suffix)

			return true
		}
	}

	return false

}

func matchStr(word string, pattern string) bool {
	match, _ := regexp.MatchString(pattern, word)
	if match {
		log.Debugf("Word '%s' match with string '%s'", word, pattern)

		return true
	}

	return false
}

func validateError(err error, continueFlow bool) bool {
	if err != nil && continueFlow {
		log.Error(err)

		return false
	} else if err != nil && !continueFlow {
		log.Fatal(err)
	}

	return true
}

// existsPath returns whether the given file or directory exists or not
func existsPath(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}

	return true, err
}
