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
	"github.com/dustin/go-humanize"
	"path/filepath"
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
	var totalSizeFilesDeleted int64
	for _, directory := range directories {
		existsPath, err := existsPath(directory)
		if validateError(err, true) && existsPath {
			tmpFilesDeleted, tmpSizeDeleted := deleteFile(directory, configuration.Pattern, extensions, configuration.DaysOfExpiration, configuration.SearchInSubdirectories)
			totalFilesDeleted += tmpFilesDeleted
			totalSizeFilesDeleted += tmpSizeDeleted
		}
	}
	log.Noticef("Files deleted: %d - Size deleted: %s", totalFilesDeleted, humanize.Bytes(uint64(totalSizeFilesDeleted)))
}

func deleteFile(directory, pattern string, extensions []string, daysOfExpiration uint, searchInSubDirs bool) (uint, int64) {
	var filesDeleted uint
	var totalSizeDeleted int64
	files, err := ioutil.ReadDir(directory)
	if validateError(err, true) {
		for _, file := range files {
			fileCompleteName, err := filepath.Abs(directory)
			validateError(err, true)
			fileCompleteName += string(os.PathSeparator) + file.Name()
			if file.IsDir() && searchInSubDirs {
				filesDeletedSubDir, totalSizeDeletedSubDir := deleteFile(fileCompleteName, pattern, extensions, daysOfExpiration, searchInSubDirs)
				filesDeleted = filesDeleted + filesDeletedSubDir
				totalSizeDeleted = totalSizeDeleted + totalSizeDeletedSubDir
				log.Infof("Directory: %s - Files deleted: %d - Size deleted: %s", fileCompleteName, filesDeletedSubDir, humanize.Bytes(uint64(totalSizeDeletedSubDir)))
			} else if !file.IsDir() {
				if (!isEmpty(pattern) && matchStr(file.Name(), pattern)) || (len(extensions) > 0 && endsWith(file.Name(), extensions)) {
					daysOld := uint(time.Since(file.ModTime()).Hours()) / 24
					log.Infof("File: %s - Last update: %s - Days old: %d", file.Name(), file.ModTime().Format("2006-01-02"), daysOld)
					if daysOld >= daysOfExpiration {
						err := os.Remove(fileCompleteName)
						if validateError(err, true) {
							filesDeleted = filesDeleted + 1
							totalSizeDeleted = totalSizeDeleted + file.Size()
							log.Infof("File deleted: %s - Size: %s", fileCompleteName, humanize.Bytes(uint64(file.Size())))
						}
					}
				}
			}
		}
	}

	return filesDeleted, totalSizeDeleted
}

func endsWith(word string, suffixes []string) bool {
	for _, suffix := range suffixes {
		if !isEmpty(suffix) && strings.HasSuffix(strings.ToUpper(word), strings.ToUpper(suffix)) {
			log.Debugf("Word '%s' has suffix '%s'", word, suffix)

			return true
		}
	}

	return false

}

func matchStr(word, pattern string) bool {
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

func isEmpty(str string) bool {
	return len(strings.TrimSpace(str)) == 0
}