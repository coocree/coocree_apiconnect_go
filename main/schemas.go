package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func findFiles(root string, fileType string) ([]string, error) {
	// Inicia uma lista vazia de arquivos e percorre o diretório root recursivamente em busca de arquivos com a extensão especificada em fileType
	var files []string
	err := filepath.Walk(root, func(
		path string, info os.FileInfo, err error) error {
		if !info.IsDir() && strings.Contains(path, fileType) {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func renderQuery(buffer *bytes.Buffer) *bytes.Buffer {
	// Renderiza as consultas GraphQL em um buffer e retorna o buffer
	_, _ = fmt.Fprint(buffer, "type Query {\n")
	files, _ := findFiles("./", "query.graphqls")
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err == nil {
			text := string(content)
			// Encontra a definição da consulta GraphQL no arquivo e a adiciona ao buffer
			re := regexp.MustCompile(`type\s[\s\S].*\s\{`)
			index := re.FindAllIndex(content, 1)
			pos := index[0][1]
			result := text[pos+1 : len(text)-1]
			_, _ = fmt.Fprint(buffer, result)
		}
	}
	_, _ = fmt.Fprint(buffer, "}\n\r")
	return buffer
}

func renderMutation(buffer *bytes.Buffer) *bytes.Buffer {
	// Renderiza as mutações GraphQL em um buffer e retorna o buffer
	_, _ = fmt.Fprint(buffer, "type Mutation {\n")
	files, _ := findFiles("./", "mutation.graphqls")
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err == nil {
			text := string(content)
			// Encontra a definição da mutação GraphQL no arquivo e a adiciona ao buffer
			re := regexp.MustCompile(`type\s[\s\S].*\s\{`)
			index := re.FindAllIndex(content, 1)
			pos := index[0][1]
			result := text[pos+1 : len(text)-1]
			_, _ = fmt.Fprint(buffer, result)
		}
	}
	_, _ = fmt.Fprint(buffer, "}\n\r")
	return buffer
}

func renderSubscription(buffer *bytes.Buffer) *bytes.Buffer {
	// Renderiza as mutações GraphQL em um buffer e retorna o buffer
	_, _ = fmt.Fprint(buffer, "type Subscription {\n")
	files, _ := findFiles("./", "subscription.graphqls")
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err == nil {
			text := string(content)
			// Encontra a definição da mutação GraphQL no arquivo e a adiciona ao buffer
			re := regexp.MustCompile(`type\s[\s\S].*\s\{`)
			index := re.FindAllIndex(content, 1)
			pos := index[0][1]
			result := text[pos+1 : len(text)-1]
			_, _ = fmt.Fprint(buffer, result)
		}
	}
	_, _ = fmt.Fprint(buffer, "}")
	return buffer
}

func main() {
	// Cria um buffer para armazenar as consultas e as mutações GraphQL
	buffer := bytes.NewBuffer(nil)
	buffer = renderQuery(buffer)
	buffer = renderMutation(buffer)
	buffer = renderSubscription(buffer)

	// Escreve o esquema GraphQL resultante no arquivo schema.graphqls
	if err := os.WriteFile("graph/schema.graphqls", buffer.Bytes(), 0644); err != nil {
		panic(err)
	}

	fmt.Println("RenderSchemas")
}
