package main

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/rwcarlsen/goexif/exif"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	InfoLogger    *log.Logger
	WarningLogger *log.Logger
	ErrorLogger   *log.Logger
)

func main() {
	InfoLogger = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime)
	WarningLogger = log.New(os.Stdout, "WARNING: ", log.Ldate|log.Ltime)
	ErrorLogger = log.New(os.Stderr, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)

	if len(os.Args[1:]) != 2 {
		ErrorLogger.Fatal(errors.New("Need 2 args"))
	}

	InfoLogger.Printf("Lets do this %s", os.Args[1])

	iterateSourcePath(os.Args[1], os.Args[2])
}

func createDirIfNotExist(dir string) (err error) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			return err
		}
	}
	return nil
}

func copyFileIfRequired(srcFilepath string, dstFilepath string) (err error) {

	//Safety
	if srcFilepath == dstFilepath {
		return errors.New("Not copying - file already exists")
	}

	err = createDirIfNotExist(filepath.Dir(dstFilepath))

	if err != nil {
		return err
	}

	// Open original file
	original, err := os.Open(srcFilepath)
	if err != nil {
		ErrorLogger.Printf(err.Error())
	}
	defer original.Close()

	// Create new file
	new, err := os.Create(dstFilepath)
	if err != nil {
		ErrorLogger.Printf(err.Error())
	}
	defer new.Close()

	//This will copy the file
	bytesWritten, err := io.Copy(new, original)
	if err != nil {
		return err
	}
	InfoLogger.Printf("Bytes Written: %d\n", bytesWritten)
	return nil
}

func getTargetFilepath(filePath string, baseDir string) (targetFilepath string, err error) {

	targetDir, err := getTargetDirectoryForFile(filePath)

	if err != nil {
		return targetFilepath, err
	}

	attempt := 0
	foundUniqueFilename := false
	foundDuplicateFile := false
	extension := filepath.Ext(filePath)
	baseFilename := strings.TrimSuffix(filepath.Base(filePath), extension)
	md5OfOriginal, _ := hashFileMD5(filePath)
	InfoLogger.Printf("filePath %s", filePath)
	InfoLogger.Printf("baseFilename %s", baseFilename)
	InfoLogger.Printf("extension %s", extension)

	for attempt < 99 && !foundUniqueFilename && !foundDuplicateFile {

		attemptFilename := baseFilename
		if attempt > 0 {
			attemptFilename = fmt.Sprintf("%s-%d%s", baseFilename, attempt, extension)
		} else {
			attemptFilename = fmt.Sprintf("%s%s", baseFilename, extension)
		}
		InfoLogger.Printf("Attempt %s", attemptFilename)

		targetFilepath = filepath.Join(baseDir, targetDir, attemptFilename)

		if _, err = os.Stat(targetFilepath); err == nil {
			InfoLogger.Printf("%s exists - checking MD5", targetFilepath)
			hash, _ := hashFileMD5(targetFilepath)
			if hash == md5OfOriginal {
				InfoLogger.Printf("%s has same md5 as %s (%s) - duplicate file", targetFilepath, filePath, md5OfOriginal)
				foundDuplicateFile = true
			}
		} else {
			InfoLogger.Printf("%s does not exist - this is the target filename", targetFilepath)
			foundUniqueFilename = true
		}

		attempt++

	}

	if foundUniqueFilename {
		return targetFilepath, nil
	}

	if foundDuplicateFile {
		return "", errors.New("Found duplicate file")
	}

	return "", err

}

/*
 Gets the relative target directory for the file based on its timestamp
*/
func getTargetDirectoryForFile(filePath string) (targetDirectory string, err error) {

	t, err := getTimeFromMediaFile(filePath)

	if err != nil {
		return targetDirectory, err
	}

	if t.IsZero() {
		return targetDirectory, errors.New("Failed to get time of photo")
	}

	targetDirectory = t.Format("2006/01/02")

	return targetDirectory, err
}

func getTimeFromMediaFile(filename string) (t time.Time, err error) {

	imgFile, err := os.Open(filename)
	if err != nil {
		return t, err
	}

	x, err := exif.Decode(imgFile)
	if err != nil {
		//warn that we failed to decode exif, but it's not really an error?
		return t, nil
	}

	t, err = x.DateTime()
	if err != nil {
		return t, err
	}

	return t, nil

}

func iterateSourcePath(sourcePath string, targetDir string) {
	filepath.Walk(sourcePath, func(filePath string, fileInfo os.FileInfo, err error) error {
		if err != nil {
			ErrorLogger.Printf(err.Error())
			return nil
		}

		fileStat, err := os.Stat(filePath)

		if err != nil {
			ErrorLogger.Printf(err.Error())
			return nil
		}

		//Ignore any directories
		if fileStat.IsDir() == true {
			InfoLogger.Printf("%s is a directory", filePath)
			return nil
		}

		fmt.Printf("File Name: %s\n", fileInfo.Name())

		// Get the target filepath
		targetFilepath, err := getTargetFilepath(filePath, targetDir)

		if err != nil {
			WarningLogger.Printf("Failed to get target filepath for %s", filePath)
			return nil
		}

		if targetFilepath != "" {
			//targetFilepath := filepath.Join(targetDir, targetFilename)
			copyFileIfRequired(filePath, targetFilepath)
		}

		return nil
	})
}

func hashFileMD5(filePath string) (string, error) {
	var returnMD5String string
	file, err := os.Open(filePath)
	if err != nil {
		return returnMD5String, err
	}
	defer file.Close()
	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return returnMD5String, err
	}
	hashInBytes := hash.Sum(nil)[:16]
	returnMD5String = hex.EncodeToString(hashInBytes)
	return returnMD5String, nil
}
