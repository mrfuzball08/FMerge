package main

import (
	"bufio"
	"fmt"
	"os"
)

func ReadMDLines(filepath string) ([]string, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", filepath, err)
	}
	defer file.Close()

	// leemos el archivo linea por linea usando bufio.Scanner
	scanner := bufio.NewScanner(file)
	// establecemos el buffer de tamaño 64KB a 1MB
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var lines []string
	for scanner.Scan() {
		// leyendo linea por linea
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read %s: %w", filepath, err)
	}

	return lines, nil
}
