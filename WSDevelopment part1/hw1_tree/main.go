package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
)

func printSize(size int64) string {
	if size == 0 {
		return "empty"
	} else {
		return fmt.Sprintf("%vb", size)
	}
}

func printDirectory(output io.Writer, dirPath string, printFilesFlag bool, previousPrefix string) {

	listDirs, err := ioutil.ReadDir(dirPath)
	if err != nil {
		fmt.Printf("Cannot read directory %v \n", dirPath)
		return
	}

	if !printFilesFlag {
		listDirsFiltered := make([]os.FileInfo, 0)
		for _, elem := range listDirs {
			if elem.IsDir() {
				listDirsFiltered = append(listDirsFiltered, elem)
			}
		}
		listDirs = listDirsFiltered
	}

	lastDirID := len(listDirs)-1
	for idx, elem := range listDirs {
		switch {
		case printFilesFlag && !elem.IsDir() && idx != lastDirID:
			delimiter := previousPrefix + "├───%v (%v)\n"
			dirGraph := []byte(fmt.Sprintf(delimiter, elem.Name(), printSize(elem.Size())))
			_, err := output.Write(dirGraph)
			if err != nil {
				panic(err)
			}
		case printFilesFlag && !elem.IsDir() && idx == lastDirID:
			delimiter := previousPrefix + "└───%v (%v)\n"
			dirGraph := []byte(fmt.Sprintf(delimiter, elem.Name(), printSize(elem.Size())))
			_, err := output.Write(dirGraph)
			if err != nil {
				panic(err)
			}
		case elem.IsDir() && idx < lastDirID:
			delimiter := previousPrefix + "├───%v\n"
			dirGraph := []byte(fmt.Sprintf(delimiter, elem.Name()))
			_, err := output.Write(dirGraph)
			if err != nil {
				panic(err)
			}
			printDirectory(output, dirPath+string(os.PathSeparator)+elem.Name(), printFilesFlag, previousPrefix+"│\t")

		case elem.IsDir() && idx == lastDirID:
			delimiter := previousPrefix + "└───%v\n"
			dirGraph := []byte(fmt.Sprintf(delimiter, elem.Name()))
			_, err := output.Write(dirGraph)
			if err != nil {
				panic(err)
			}

			printDirectory(output, dirPath+string(os.PathSeparator)+elem.Name(), printFilesFlag, previousPrefix+"\t")

		}
	}
}

func dirTree(output io.Writer, dirPath string, printFilesFlag bool) error {
	pathInfo, err := os.Stat(dirPath)
	if err != nil {
		return err
	}

	if pathInfo.IsDir() {
		printDirectory(output, dirPath, printFilesFlag, "")
	} else {
		return nil
	}
	return nil
}

func main() {
	out := os.Stdout
	if !(len(os.Args) == 2 || len(os.Args) == 3) {
		panic("usage go run main.go . [-f]")
	}
	path := os.Args[1]
	printFiles := len(os.Args) == 3 && os.Args[2] == "-f"

	err := dirTree(out, path, printFiles)
	if err != nil {
		panic(err)
	}
}
