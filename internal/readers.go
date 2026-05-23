package internal

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type FileReader interface {
	Read(path string) (string, error)
	Write(path string, content string) error
	SupportedExtensions() []string
}

type TextFileReader struct{}

func (r *TextFileReader) Read(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("error al leer archivo: %w", err)
	}
	return string(data), nil
}

func (r *TextFileReader) Write(path string, content string) error {
	err := os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		return fmt.Errorf("error al escribir archivo: %w", err)
	}
	return nil
}

func (r *TextFileReader) SupportedExtensions() []string {
	return []string{".txt", ".log", ".md", ".csv"}
}

type JSONFileReader struct{}

func (r *JSONFileReader) Read(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("error al leer archivo JSON: %w", err)
	}
	return string(data), nil
}

func (r *JSONFileReader) Write(path string, content string) error {
	var jsonObj interface{}
	if err := json.Unmarshal([]byte(content), &jsonObj); err != nil {
		return fmt.Errorf("contenido no es JSON válido: %w", err)
	}

	formatted, err := json.MarshalIndent(jsonObj, "", "  ")
	if err != nil {
		return fmt.Errorf("error al formatear JSON: %w", err)
	}

	return os.WriteFile(path, formatted, 0644)
}

func (r *JSONFileReader) SupportedExtensions() []string {
	return []string{".json"}
}

type YAMLFileReader struct{}

func (r *YAMLFileReader) Read(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("error al leer archivo YAML: %w", err)
	}
	return string(data), nil
}

func (r *YAMLFileReader) Write(path string, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}

func (r *YAMLFileReader) SupportedExtensions() []string {
	return []string{".yaml", ".yml"}
}

type ENVFileReader struct{}

func (r *ENVFileReader) Read(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("error al leer archivo .env: %w", err)
	}
	return string(data), nil
}

func (r *ENVFileReader) Write(path string, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}

func (r *ENVFileReader) SupportedExtensions() []string {
	return []string{".env"}
}

type CSVFileReader struct{}

func (r *CSVFileReader) Read(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("error al abrir archivo CSV: %w", err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	if err != nil {
		return "", fmt.Errorf("error al leer CSV: %w", err)
	}

	var sb strings.Builder
	for _, row := range records {
		sb.WriteString(strings.Join(row, ","))
		sb.WriteString("\n")
	}
	return sb.String(), nil
}

func (r *CSVFileReader) Write(path string, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}

func (r *CSVFileReader) SupportedExtensions() []string {
	return []string{".csv"}
}

var readers = []FileReader{
	&TextFileReader{},
	&JSONFileReader{},
	&YAMLFileReader{},
	&ENVFileReader{},
	&CSVFileReader{},
}

func GetReaderForFile(path string) (FileReader, error) {
	ext := strings.ToLower(filepath.Ext(path))

	for _, reader := range readers {
		for _, supported := range reader.SupportedExtensions() {
			if ext == supported {
				return reader, nil
			}
		}
	}

	return nil, fmt.Errorf("formato de archivo no soportado: %s (soportados: .txt, .log, .md, .json, .yaml, .yml, .env, .csv)", ext)
}
