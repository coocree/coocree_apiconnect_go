package main

import (
	"bytes"
	"fmt"
	"github.com/fatih/camelcase"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

var listMutationQueryFile []MutationQueryFileModel
var listResponseModel = map[string]ResultModel{}

type ActionModel struct {
	Name     string
	Args     []ArgModel
	Response string
	Package  string
	Project  string
	Type     string
}

type ArgModel struct {
	Name           string
	Type           string
	isRequerid     bool
	isList         bool
	isListRequerid bool
}

type MutationQueryFileModel struct {
	Name      string
	Type      string
	Actions   []ActionModel
	Path      string
	Package   string
	Project   string
	hasUpload bool
}

type ResultModel struct {
	Name           string
	isList         bool
	isRequerid     bool
	isListRequerid bool
}

var re = regexp.MustCompile(`type\s[\s\S].*\s\{`)
var re1 = regexp.MustCompile(`#.*|#\s.*|#-*|"""[\s\S].+"""|,`)
var re2 = regexp.MustCompile(`\s`)
var re3 = regexp.MustCompile(`\#+`)
var re4 = regexp.MustCompile(`\(\#+`)
var re5 = regexp.MustCompile(`\#\)+`)
var re6 = regexp.MustCompile(`\:#`)
var re9 = regexp.MustCompile(`!`)
var re12 = regexp.MustCompile(`[\[\]!]`)
var reIsRequired = regexp.MustCompile(`\w+!`)
var reIsList = regexp.MustCompile(`\[\w+!\]|\[\w+\]`)
var reIsListRequired = regexp.MustCompile(`\[\w+!\]!|\[\w+\]!`)
var reApi = regexp.MustCompile(`Api`)
var regexRemoveSchemas = regexp.MustCompile(`\\schemas`)
var regexIsImplemented = regexp.MustCompile(`not implemented`)

func contentClean(content []byte) string {
	// Converte o conteúdo de bytes para string
	text := string(content)

	// Encontra a definição da consulta/mutação GraphQL no arquivo e a separa do resto do conteúdo
	index := re.FindAllIndex(content, 1)
	pos := index[0][1]
	result := text[pos+1 : len(text)-1]

	// Remove comentários e caracteres especiais do resultado
	result = re1.ReplaceAllString(result, "")
	result = re2.ReplaceAllString(result, "#")
	result = re3.ReplaceAllString(result, "#")
	result = re4.ReplaceAllString(result, "(")
	result = re5.ReplaceAllString(result, ")")
	return re6.ReplaceAllString(result, ":")
}

func createActions(pathFile string, content string) (listActions []ActionModel) {
	// Define expressões regulares para encontrar as definições de ações (com e sem parâmetros)
	var regexGetActionParams = regexp.MustCompile(`(\w+)(\([\s\S]+?\)):(\w+!|\w+)`)
	var regexGetActionNoParams = regexp.MustCompile(`(\w+):(\w+!|\w+)`)

	// Encontra todas as definições de ações com parâmetros e cria modelos de ação correspondentes
	actionsParams := regexGetActionParams.FindAllString(content, -1)
	for _, action := range actionsParams {
		item := regexGetActionParams.FindStringSubmatch(action)
		if len(item) > 1 {
			actionModel := ActionModel{}
			actionModel.Name = item[1]
			actionModel.Args = createArgs(actionModel.Name, pathFile, item[2])
			actionModel.Response = createResponse(actionModel.Name, pathFile, item[3])
			listActions = append(listActions, actionModel)
		}
	}

	// Remove os parênteses dos parâmetros das ações sem parâmetros e cria modelos de ação correspondentes
	content = regexGetActionParams.ReplaceAllString(content, ")")
	actionsNoParams := regexGetActionNoParams.FindAllString(content, -1)

	for _, action := range actionsNoParams {
		item := regexGetActionNoParams.FindStringSubmatch(action)
		if len(item) > 1 {
			actionModel := ActionModel{}
			actionModel.Name = item[1]
			actionModel.Args = []ArgModel{}
			actionModel.Response = createResponse(actionModel.Name, pathFile, item[2])
			listActions = append(listActions, actionModel)
		}
	}

	// Retorna a lista de modelos de ação criados
	return listActions
}

func createArgs(actionName string, pathFile string, value string) (result []ArgModel) {
	// Define expressões regulares para remover parênteses, interrogações e colchetes das definições de argumentos
	var regexRemoveParents = regexp.MustCompile(`\((\w.+)\)`)
	var regexRemoveInterrogation = regexp.MustCompile(`!|\[|\]`)
	var regexInput = regexp.MustCompile(`Input`)

	// Encontra os valores dos argumentos na string de definição de ação
	values := regexRemoveParents.FindStringSubmatch(value)
	args := strings.Split(values[1], "#")
	sort.Strings(args)

	// Cria um modelo de argumento para cada item encontrado e adiciona à lista de argumentos (result)
	for _, item := range args {
		itemsArg := strings.Split(item, ":")
		arg := ArgModel{}
		arg.Name = itemsArg[0]
		arg.Type = regexRemoveInterrogation.ReplaceAllString(itemsArg[1], "")
		arg.isRequerid = reIsRequired.MatchString(itemsArg[1])
		arg.isList = reIsList.MatchString(itemsArg[1])
		arg.isListRequerid = reIsListRequired.MatchString(itemsArg[1])

		// Verifica se o argumento é do tipo 'input' e se o nome do tipo segue a convenção 'ArgInput'
		if arg.Name == "input" && !regexInput.MatchString(arg.Type) {
			fmt.Println("INPUT", actionName, arg.Type)
			fmt.Println("--------------------------- Error Args Input ---------------------------")
			fmt.Println("Error na configuração do argumento input da Action '" + actionName + "', localizado em " + pathFile)
			fmt.Println("'" + arg.Type + "' precisa ter 'Input' presente na formação do nome, Exemplo: 'ArgInput'")
			log.Fatal("\n----------------------------------------------------------------------\n\n")
		}
		result = append(result, arg)
	}

	return result
}

func createFileModel(fileType string) {
	// Encontra todos os arquivos com o tipo especificado na pasta atual e subpastas.
	files, _ := findFilesType("./", fileType+".graphqls")

	// Expressão regular para encontrar a palavra-chave "Upload" nos arquivos de esquema.
	var regexHasUpload = regexp.MustCompile(`Upload`)

	// Itera sobre cada arquivo encontrado.
	for _, pathFile := range files {
		// Divide o caminho do arquivo em três partes: pasta raiz, projeto e pacote.
		splitPath := strings.Split(pathFile, "\\")
		// Lê o conteúdo do arquivo.
		content, err := os.ReadFile(pathFile)
		if err == nil {
			// Cria um novo modelo de arquivo de mutação/consulta GraphQL.
			mqModel := MutationQueryFileModel{}
			mqModel.Type = fileType
			mqModel.Name = "service_" + fileType + ".go"
			mqModel.Project = splitPath[1]
			mqModel.Package = splitPath[2]
			// Define o caminho do arquivo (sem a pasta "schemas").
			path := filepath.Dir(pathFile)
			mqModel.Path = regexRemoveSchemas.ReplaceAllString(path, "")
			// Remove comentários e formata o conteúdo do arquivo.
			contentCleaned := contentClean(content)
			// Verifica se o arquivo contém uploads.
			mqModel.hasUpload = regexHasUpload.MatchString(contentCleaned)
			// Cria os modelos de ação para o arquivo de esquema.
			mqModel.Actions = createActions(pathFile, contentCleaned)
			// Adiciona o modelo do arquivo à lista de modelos de arquivos.
			listMutationQueryFile = append(listMutationQueryFile, mqModel)
		}
	}
}

func createListResult() {
	// Expressão regular para encontrar a definição de tipo de resultado
	var regexFindResult = regexp.MustCompile(`type[\s|\S].+\{[\s|\S]+?\}`)

	// Encontra todos os arquivos de resposta e itera sobre eles
	files, _ := findFilesType("./", "response.graphqls")
	for _, file := range files {
		// Lê o conteúdo do arquivo
		content, err := os.ReadFile(file)
		if err == nil {
			// Encontra todas as definições de tipo de resultado no arquivo e passa para a função processResponse()
			matchResults := regexFindResult.FindAllString(string(content), -1)
			processResponse(matchResults, file)
		}
	}
}

func createParamClean(param string) string {
	// Divide o nome do parâmetro em partes, usando ':' como delimitador
	params := strings.Split(param, ":")
	nameSplit := camelcase.Split(params[0])
	result := ""
	// Itera sobre cada parte do nome do parâmetro e forma o nome final
	for _, namePart := range nameSplit {
		if ((namePart == "id" || namePart == "Id") || (namePart == "api" || namePart == "Api")) && len(nameSplit) > 1 {
			// Se o nome da parte for 'id' ou 'api' e a quantidade de partes for maior que 1, adiciona em maiúsculas
			result += strings.ToUpper(namePart)
		} else {
			result += namePart
		}
	}
	return result
}

func createParamsResponse(response string) string {
	// Remove caracteres desnecessários do nome do tipo de resposta
	response = re9.ReplaceAllString(response, "")
	// Divide o nome do tipo de resposta em partes
	splitted := camelcase.Split(response)
	result := ""
	// Itera sobre cada parte do nome do tipo de resposta e forma o nome final
	for index, k := range splitted {
		if k == "api" || k == "Api" && index == 0 {
			// Se a parte for 'api' e for a primeira, adiciona em maiúsculas
			result += strings.ToUpper(k)
		} else {
			result += k
		}
	}
	return "model." + result
}

// createResponse cria o nome do tipo de resposta para uma determinada ação, validando se o nome está correto
func createResponse(actionName string, pathFile string, value string) string {
	// regex para buscar apenas as letras e números na string
	var getWord = regexp.MustCompile(`\w+`)
	// regex para validar se o nome do tipo de resposta contém o sufixo "Response!"
	var regexIsValid = regexp.MustCompile(`Response!`)
	// extrai o nome do tipo de resposta da string completa
	name := strings.TrimSpace(getWord.FindString(value))

	// verifica se o nome do tipo de resposta contém o sufixo "Response!"
	if !regexIsValid.MatchString(value) {
		// caso o nome esteja incorreto, exibe uma mensagem de erro com o nome esperado
		fmt.Println("--------------------------- Error Response ---------------------------")
		fmt.Println("Error na configuração do response da action '" + actionName + "', localizado em " + pathFile)
		fmt.Println("'" + value + "' precisa ter 'Response!' presente na formação do nome, Exemplo: 'ActionResponse!'")
		log.Fatal("\n----------------------------------------------------------------------\n\n")
	}

	return name
}

// createResolversImport cria as importações necessárias para os arquivos de resolver
func createResoversImport(hasUpload bool) *bytes.Buffer {
	// regex para substituir as barras invertidas por barras normais nos caminhos dos arquivos
	var regexRepleceSlash = regexp.MustCompile(`\\`)
	// buffer para armazenar as importações
	buffer := bytes.NewBuffer(nil)

	_, _ = fmt.Fprint(buffer, "package graph\n\n")
	_, _ = fmt.Fprint(buffer, "import (\n")
	_, _ = fmt.Fprint(buffer, "\t\"coocree_kdl_go_apiconnect/graph/model\"\n")
	_, _ = fmt.Fprint(buffer, "\t\"context\"\n")

	var listPaths []string
	for _, fileModel := range listMutationQueryFile {
		path := regexRepleceSlash.ReplaceAllString(fileModel.Path, "/")
		listPaths = append(listPaths, path)
	}

	// remove os caminhos duplicados
	listPaths = removeDuplicateStr(listPaths)
	// ordena os caminhos em ordem alfabética
	sort.Strings(listPaths)

	for _, path := range listPaths {
		// adiciona as importações dos arquivos de modelo de cada caminho
		_, _ = fmt.Fprint(buffer, "\t\"coocree_kdl_go_apiconnect/"+path+"\"\n")
	}

	if hasUpload {
		// adiciona a importação necessária para o uso de upload de arquivos
		_, _ = fmt.Fprint(buffer, "\t\"github.com/99designs/gqlgen/graphql\"\n")
	}

	_, _ = fmt.Fprint(buffer, ")\n")
	return buffer
}

// findFilesType encontra todos os arquivos com uma determinada extensão em um diretório e seus subdiretórios
func findFilesType(root string, fileType string) ([]string, error) {
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

// Função que recebe uma string e retorna a mesma com a primeira letra em maiúscula
func fistUpperCase(value string) string {
	// Pega a primeira letra da string
	fistLetter := value[:1]
	// Converte a primeira letra para maiúscula e concatena com o restante da string
	return strings.ToUpper(fistLetter) + value[1:]
}

// Função que recebe um modelo de argumento e retorna uma string representando o tipo de lista correspondente
func isList(model ArgModel) string {
	// Cria uma string com o tipo requerido e adicione o prefixo "[]" se for uma lista
	result := "[]" + typeRequerid(model)
	// Adiciona o prefixo "*" se o tipo de lista não for requerido e não for ID ou String
	if !model.isListRequerid && model.Type != "ID" && model.Type != "String" {
		result = "*" + result
	}
	return result
}

// Função que processa uma lista de strings contendo o conteúdo de arquivos e busca por padrões específicos usando regex
func processResponse(values []string, pathFile string) {
	// Cria expressões regulares para encontrar padrões específicos no texto
	var regexFindResult = regexp.MustCompile(`type\s(\w+)\s\{[\s|\S]+?result:\s([\s\S].*|!)`)
	var regexIsValid = regexp.MustCompile(`Result`)
	var regexIsValidSuccess = regexp.MustCompile(`success`)
	var regexIsValidElapsedTime = regexp.MustCompile(`elapsedTime`)

	// Itera sobre cada item da lista de strings
	for _, item := range values {
		// Procura por padrões específicos no item usando a expressão regular regexFindResult
		listMatch := regexFindResult.FindStringSubmatch(item)

		// Se o padrão for encontrado
		if len(listMatch) > 1 {
			// Pega o nome do response
			responseName := listMatch[1]
			responseName = re9.ReplaceAllString(responseName, "")
			// Pega o modelo de resultado do response
			itemResult := listMatch[2]

			// Cria um novo modelo de resultado e configura suas propriedades
			resultModel := ResultModel{}
			resultModel.isList = reIsList.MatchString(itemResult)
			resultModel.isListRequerid = reIsListRequired.MatchString(itemResult)
			resultModel.isRequerid = reIsRequired.MatchString(itemResult)
			resultModel.Name = strings.TrimSpace(re12.ReplaceAllString(itemResult, ""))
			listResponseModel[responseName] = resultModel

			// Verifica se o modelo de resultado possui a palavra "Result" no nome
			if !regexIsValid.MatchString(resultModel.Name) && resultModel.Name != "Int" && resultModel.Name != "Boolean" && resultModel.Name != "String" {
				fmt.Println("--------------------------- Error Response Result ---------------------------")
				fmt.Println("Error na configuração do result do response '" + responseName + "', localizado em " + pathFile)
				fmt.Println("'" + resultModel.Name + "' precisa ter 'Result' presente na formação do nome, Exemplo: 'ActionResult'")
				log.Fatal("\n---------------------------------------------------------------------------\n\n")
			}

			// Verifica se o response possui a palavra "success" no nome
			if !regexIsValidSuccess.MatchString(item) {
				fmt.Println("--------------------------- Error Response Success ---------------------------")
				fmt.Println("Error na configuração do response '" + responseName + "', localizado em " + pathFile)
				fmt.Println("'" + responseName + "' precisa ter 'success' presente na formação")
				log.Fatal("\n----------------------------------------------------------------------------\n\n")
			}

			// Verifica se o response possui a palavra "elapsedTime" no nome
			if !regexIsValidElapsedTime.MatchString(item) {
				fmt.Println("--------------------------- Error Response ElapsedTime ---------------------------")
				fmt.Println("Error na configuração do response '" + responseName + "', localizado em " + pathFile)
				fmt.Println("'" + responseName + "' precisa ter 'elapsedTime' presente na formação")
				log.Fatal("\n--------------------------------------------------------------------------------\n\n")
			}
		}
	}
}

// Função que recebe uma lista de modelos de argumentos e retorna uma string representando a lista de argumentos para uma função de ação
func renderActionArgs(args []ArgModel) string {
	// Cria um slice vazio para armazenar os resultados
	var result []string
	// Itera sobre cada modelo de argumento
	for _, arg := range args {
		// Chama a função renderParamModel para renderizar o modelo de argumento como uma string
		result = append(result, renderParamModel(arg))
	}
	// Junta todas as strings geradas com uma vírgula e retorna o resultado final
	return strings.Join(result, ", ")
}

// Função que recebe uma lista de modelos de argumentos e retorna uma string representando a lista de argumentos limpos para uma função de ação
func renderArgsClean(args []ArgModel) string {
	// Cria um slice com o valor "ctx" como primeiro elemento
	result := []string{"ctx"}
	// Itera sobre cada modelo de argumento
	for _, arg := range args {
		// Chama a função createParamClean para criar um parâmetro limpo a partir do nome do argumento
		result = append(result, createParamClean(arg.Name))
	}
	// Junta todas as strings geradas com uma vírgula e retorna o resultado final
	return strings.Join(result, ", ")
}

// Função que recebe um modelo de argumento e retorna uma string representando o parâmetro formatado
func renderParamModel(arg ArgModel) string {
	// Divide o nome do argumento pelo caractere ":" para separar o nome da tag
	name := strings.Split(arg.Name, ":")
	// Divide o nome do argumento usando a função camelcase.Split para obter as palavras
	nameSplit := camelcase.Split(name[0])
	result := ""
	// Itera sobre cada palavra do nome do argumento
	for _, namePart := range nameSplit {
		// Verifica se a palavra é "id" ou "api" e se a lista de palavras tem mais de um elemento, para colocar essas palavras em maiúsculas
		if ((namePart == "id" || namePart == "Id") || (namePart == "api" || namePart == "Api")) && len(nameSplit) > 1 {
			result += strings.ToUpper(namePart)
		} else {
			result += namePart
		}
	}
	// Concatena a palavra resultante com o tipo formatado do argumento
	return result + " " + renderTypeModel(arg)
}

// Função que renderiza o arquivo "schema.resolvers.go" com as implementações das resolvers geradas para as mutations e queries
func renderResolver() {
	// Cria um mapa vazio para armazenar as mutations e queries encontradas
	listMutationQuery := map[string]ActionModel{}
	// Cria um slice vazio para armazenar as chaves do mapa em ordem alfabética
	var listKeys []string
	// Cria uma variável para indicar se há uploads em alguma mutation
	hasUpload := false
	// Itera sobre cada modelo de mutation/query encontrado
	for _, fileModel := range listMutationQueryFile {
		// Itera sobre cada modelo de ação (mutation/query) do arquivo
		for _, actionModel := range fileModel.Actions {
			// Adiciona informações adicionais à actionModel, como o projeto, pacote e tipo (mutation/query)
			actionModel.Project = fileModel.Project
			actionModel.Package = fileModel.Package
			actionModel.Type = fileModel.Type
			// Adiciona a actionModel ao mapa de mutations/queries, com o nome da ação como chave
			listMutationQuery[actionModel.Name] = actionModel
			// Adiciona o nome da ação à lista de chaves, para ordenar posteriormente
			listKeys = append(listKeys, actionModel.Name)
		}
		// Verifica se há uploads nesse arquivo, para adicionar a importação correspondente na saída
		if !hasUpload {
			hasUpload = fileModel.hasUpload
		}
	}
	// Ordena a lista de chaves em ordem alfabética
	sort.Strings(listKeys)

	// Cria um buffer para armazenar a saída
	buffer := createResoversImport(hasUpload)
	// Itera sobre cada chave (nome de mutation/query) da lista de chaves
	for _, name := range listKeys {
		// Obtém a actionModel correspondente ao nome
		actionModel := listMutationQuery[name]
		// Transforma o nome em PascalCase (com a primeira letra maiúscula)
		name = fistUpperCase(actionModel.Name)
		// Verifica se o tipo da ação é query ou mutation
		if actionModel.Type == "query" {
			// Renderiza a implementação da resolver para a query
			_, _ = fmt.Fprint(buffer, "func (r *queryResolver) "+name+" (ctx context.Context, "+renderActionArgs(actionModel.Args)+") ("+renderResponse(actionModel.Response)+", error) {\n")
			_, _ = fmt.Fprint(buffer, "\treturn "+actionModel.Package+"."+name+"Query("+renderArgsClean(actionModel.Args)+")\n")
		} else if actionModel.Type == "mutation" {
			// Renderiza a implementação da resolver para a mutation
			_, _ = fmt.Fprint(buffer, "func (r *mutationResolver) "+name+" (ctx context.Context, "+renderActionArgs(actionModel.Args)+") ("+renderResponse(actionModel.Response)+", error) {\n")
			_, _ = fmt.Fprint(buffer, "\treturn "+actionModel.Package+"."+name+"Mutation("+renderArgsClean(actionModel.Args)+")\n")
		}
		// Fecha a função
		_, _ = fmt.Fprint(buffer, "}\n")
	}

	// Renderiza as funções Mutation e Query, que retornam implementações das resolvers para os tipos MutationResolver e QueryResolver, respectivamente
	_, _ = fmt.Fprint(buffer, "// Mutation returns MutationResolver implementation.\n")
	_, _ = fmt.Fprint(buffer, "func (r *Resolver) Mutation() MutationResolver { return &mutationResolver{r} }\n\n")
	_, _ = fmt.Fprint(buffer, "// Query returns QueryResolver implementation.\n")
	_, _ = fmt.Fprint(buffer, "func (r *Resolver) Query() QueryResolver { return &queryResolver{r} }\n\n")
	// Renderiza as structs mutationResolver e queryResolver, que armazenam um ponteiro para o Resolver
	_, _ = fmt.Fprint(buffer, "type mutationResolver struct{ *Resolver }\n")
	_, _ = fmt.Fprint(buffer, "type queryResolver struct{ *Resolver }\n")

	// Escreve o conteúdo do buffer no arquivo "schema.resolvers.go"
	if err := os.WriteFile("graph/schema.resolvers.go", buffer.Bytes(), 0644); err != nil {
		panic(err)
	}

	// Imprime no console uma mensagem indicando que a renderização das resolvers foi concluída
	fmt.Println("RenderResolver")
}

// A função `renderResponse` recebe uma string contendo o nome do tipo de resposta de uma função GraphQL e retorna uma string formatada
// que representa o tipo de resposta no código Go.
func renderResponse(response string) string {
	// Remove caracteres especiais do nome do tipo de resposta
	response = re9.ReplaceAllString(response, "")
	// Divide o nome do tipo em uma lista de palavras
	nameSplit := camelcase.Split(response)
	result := ""
	// Itera sobre as palavras do nome do tipo
	for _, namePart := range nameSplit {
		// Verifica se a palavra é "id" ou "api" e se o nome do tipo tem mais de uma palavra. Se for o caso, converte a palavra para maiúscula
		if ((namePart == "id" || namePart == "Id") || (namePart == "api" || namePart == "Api")) && len(nameSplit) > 1 {
			result += strings.ToUpper(namePart)
		} else {
			result += namePart
		}
	}
	// Retorna o tipo de resposta formatado com o prefixo "*model."
	return "*model." + result
}

// A função `renderService` é responsável por renderizar o arquivo de serviço de cada ação no formato correto.
func renderService() {
	// Itera sobre cada arquivo de ação na lista de arquivos
	for _, item := range listMutationQueryFile {
		// Cria o caminho para o arquivo de serviço
		pathFilename := "modules/" + item.Project + "/" + item.Package + "/service_" + item.Type
		// Lê o conteúdo do arquivo de serviço em bytes
		fileByte, _ := os.ReadFile(pathFilename + ".go")
		// Se o arquivo de serviço existe, renderiza o conteúdo existente. Caso contrário, cria um novo arquivo de serviço.
		if len(fileByte) > 0 {
			renderServiceExist(fileByte, item, pathFilename)
		} else {
			renderServiceNotExist(item, pathFilename)
		}
	}
	// Imprime no console uma mensagem indicando que a renderização dos serviços foi concluída
	fmt.Println("RenderService")
}

// renderServiceItem é responsável por gerar o código para um serviço específico que realiza uma ação.
// Recebe um buffer de bytes, um modelo de ação e o nome do método.
// Retorna um buffer de bytes com o código gerado.
func renderServiceItem(buffer *bytes.Buffer, actionModel ActionModel, nameMethod string) *bytes.Buffer {
	// Obtém o modelo de resultado a partir do nome do modelo de resposta
	resultModel := listResponseModel[actionModel.Response]
	// Gera a declaração do método e os valores de retorno
	_, _ = fmt.Fprint(buffer, "func "+nameMethod+"(ctx context.Context, "+renderActionArgs(actionModel.Args)+") ("+renderResponse(actionModel.Response)+", error) {\n")
	// Adiciona um panic para indicar que o método ainda não foi implementado
	_, _ = fmt.Fprint(buffer, "\tpanic(fmt.Errorf(\"not implemented\"))\n\n")
	// Adiciona um timer para medir o tempo de execução
	_, _ = fmt.Fprint(buffer, "\ttimeStart := time.Now()\n")
	// Inicializa a variável que indicará se a ação foi executada com sucesso
	_, _ = fmt.Fprint(buffer, "\tsuccess := false\n")
	// Verifica o tipo de resultado a partir do modelo de resultado obtido
	_, _ = fmt.Fprint(buffer, resultCheckType(resultModel))
	// Gera a declaração do objeto de resposta
	_, _ = fmt.Fprint(buffer, "\tresponse := "+createParamsResponse(actionModel.Response)+"{\n")
	// Adiciona o resultado da ação ao objeto de resposta
	if resultModel.isList {
		_, _ = fmt.Fprint(buffer, "\t\tResult:      result,\n")
	} else {
		_, _ = fmt.Fprint(buffer, "\t\tResult:      &result,\n")
	}
	// Adiciona a informação sobre o sucesso da ação ao objeto de resposta
	_, _ = fmt.Fprint(buffer, "\t\tSuccess:     success,\n")
	// Adiciona a informação sobre o tempo de execução da ação ao objeto de resposta
	_, _ = fmt.Fprint(buffer, "\t\tElapsedTime: time.Since(timeStart).String(),\n")
	_, _ = fmt.Fprint(buffer, "\t}\n")
	// Retorna o objeto de resposta e um erro, caso ocorra
	_, _ = fmt.Fprint(buffer, "\treturn &response, nil\n")
	_, _ = fmt.Fprint(buffer, "}\n\n")

	return buffer
}

// renderServiceExist é responsável por renderizar o serviço existente com as alterações necessárias para uma nova versão.
func renderServiceExist(fileByte []byte, item MutationQueryFileModel, pathFilename string) {
	// Expressão regular para capturar a declaração de import do pacote.
	var regexPackageImport = regexp.MustCompile(`package[\s\S]*?\)`)

	// Expressão regular para capturar todos os métodos da struct que retornam o tipo "response".
	var regexGetAllMethods = regexp.MustCompile(`(func\s(\w.+?)\([\s\S].*{)([\s\S]+?return\s&response[\s\S]+?)}`)

	// Cria um buffer para armazenar o conteúdo do arquivo.
	buffer := bytes.NewBuffer(nil)

	// Converte o conteúdo do arquivo em uma string.
	content := string(fileByte)

	// Captura a declaração do pacote do arquivo original.
	packageImport := regexPackageImport.FindString(content)

	// Adiciona a declaração do pacote no buffer.
	_, _ = fmt.Fprint(buffer, packageImport+"\n\n")

	// Captura todos os métodos do arquivo original que retornam o tipo "response".
	listMethodsExist := regexGetAllMethods.FindAllString(content, -1)
	listMethodsExistTemp := map[string][]string{}

	// Percorre todos os métodos existentes e adiciona no mapa temporário.
	for _, methodItem := range listMethodsExist {
		listMethodItem := regexGetAllMethods.FindStringSubmatch(methodItem)
		listMethodsExistTemp[(listMethodItem[2])] = listMethodItem
	}

	// Ordena as ações em ordem alfabética pelo nome.
	sort.SliceStable(item.Actions, func(i, j int) bool {
		return item.Actions[i].Name < item.Actions[j].Name
	})

	// Percorre todas as ações e gera o código correspondente para cada uma delas.
	for _, actionModel := range item.Actions {
		name := fistUpperCase(actionModel.Name)
		nameType := fistUpperCase(item.Type)
		nameMethod := name + nameType

		// Expressão regular para capturar o método correspondente à ação atual.
		var regexGetMethods = regexp.MustCompile(`(func\s(` + nameMethod + `)[\s\S]+?{)([\s\S]+?return\s&response[\s\S]+?)}`)

		// Captura o método correspondente à ação atual.
		listMatchMethods := regexGetMethods.FindStringSubmatch(content)

		// Gera a declaração da função.
		method := "func " + nameMethod + "(ctx context.Context, " + renderActionArgs(actionModel.Args) + ") (" + renderResponse(actionModel.Response) + ", error) {\n"

		// Verifica se o método correspondente à ação atual já está implementado.
		isSimilarMethod := false
		isNotImplemented := true

		if len(listMatchMethods) >= 1 {
			isNotImplemented = regexIsImplemented.MatchString(listMatchMethods[0])
			if strings.TrimSpace(method) == strings.TrimSpace(listMatchMethods[1]) {
				isSimilarMethod = true
			}
		}

		// Renderiza a ação atual no buffer, caso o método correspondente não esteja implementado.
		if isNotImplemented {
			buffer = renderServiceItem(buffer, actionModel, nameMethod)
		} else if isSimilarMethod {
			// Renderiza o método correspondente no buffer, caso o método correspondente seja semelhante à declaração da função atual.
			_, _ = fmt.Fprint(buffer, listMatchMethods[0]+"\n\n")
		} else if !isSimilarMethod {
			_, _ = fmt.Fprint(buffer, "/**------------------------------------------------------------\n")
			_, _ = fmt.Fprint(buffer, "// !!! AVISO !!!\n")
			_, _ = fmt.Fprint(buffer, "// O código abaixo foi alterado e precisa ser avaliado:\n")
			_, _ = fmt.Fprint(buffer, "// - Ao alterar parâmetros da query ou mutation graphql, precisa ser revisado.\n")
			_, _ = fmt.Fprint(buffer, "------------------------------------------------------------**/\n")
			_, _ = fmt.Fprint(buffer, "func "+nameMethod+"(ctx context.Context, "+renderActionArgs(actionModel.Args)+") ("+renderResponse(actionModel.Response)+", error) {\n")
			_, _ = fmt.Fprint(buffer, listMatchMethods[3]+"\n")
			_, _ = fmt.Fprint(buffer, "}\n\n")
		}
		delete(listMethodsExistTemp, nameMethod)
	}

	for _, bkpItem := range listMethodsExistTemp {
		saveBackup(item.Package, pathFilename, bkpItem)
	}

	if err := os.WriteFile(pathFilename+".go", buffer.Bytes(), 0644); err != nil {
		panic(err)
	}
}

func renderServiceNotExist(item MutationQueryFileModel, pathFilename string) {
	buffer := bytes.NewBuffer(nil)

	_, _ = fmt.Fprint(buffer, "package "+item.Package+"\n\n")
	_, _ = fmt.Fprint(buffer, "import (\n")
	_, _ = fmt.Fprint(buffer, "\t\"coocree_kdl_go_apiconnect/graph/model\"\n")
	_, _ = fmt.Fprint(buffer, "\t\"coocree_kdl_go_apiconnect/modules/api\"\n")
	if item.hasUpload {
		_, _ = fmt.Fprint(buffer, "\t\"github.com/99designs/gqlgen/graphql\"\n")
	}
	_, _ = fmt.Fprint(buffer, "\t\"context\"\n")
	_, _ = fmt.Fprint(buffer, "\t\"fmt\"\n")
	_, _ = fmt.Fprint(buffer, "\t\"time\"\n")
	_, _ = fmt.Fprint(buffer, "\t)\n\n")

	sort.SliceStable(item.Actions, func(i, j int) bool {
		return item.Actions[i].Name < item.Actions[j].Name
	})

	for _, actionModel := range item.Actions {
		name := fistUpperCase(actionModel.Name)
		nameType := fistUpperCase(item.Type)
		resultModel := listResponseModel[actionModel.Response]

		_, _ = fmt.Fprint(buffer, "func "+name+nameType+"(ctx context.Context, "+renderActionArgs(actionModel.Args)+") ("+renderResponse(actionModel.Response)+", error) {\n")
		_, _ = fmt.Fprint(buffer, "\tpanic(fmt.Errorf(\"not implemented\"))\n\n")
		_, _ = fmt.Fprint(buffer, "\ttimeStart := time.Now()\n")
		_, _ = fmt.Fprint(buffer, "\tsuccess := false\n")

		_, _ = fmt.Fprint(buffer, resultCheckType(resultModel))
		_, _ = fmt.Fprint(buffer, "\tresponse := "+createParamsResponse(actionModel.Response)+"{\n")

		if resultModel.isList {
			_, _ = fmt.Fprint(buffer, "\t\tResult:      result,\n")
		} else {
			_, _ = fmt.Fprint(buffer, "\t\tResult:      &result,\n")
		}

		_, _ = fmt.Fprint(buffer, "\t\tSuccess:     success,\n")
		_, _ = fmt.Fprint(buffer, "\t\tElapsedTime: time.Since(timeStart).String(),\n")
		_, _ = fmt.Fprint(buffer, "\t}\n")
		_, _ = fmt.Fprint(buffer, "\treturn &response, nil\n")
		_, _ = fmt.Fprint(buffer, "}\n\n")
	}

	if err := os.WriteFile(pathFilename+".go", buffer.Bytes(), 0644); err != nil {
		panic(err)
	}
}

func renderTypeModel(model ArgModel) string {
	var result string
	if model.isList {
		result = isList(model)
	} else {
		result = typeRequerid(model)
	}
	return result
}

func resultCheckType(resultModel ResultModel) string {
	var defaultType string
	switch resultModel.Name {
	case "Int":
		defaultType = "int"
		if resultModel.isList {
			defaultType = "[]int"
		}
	case "String":
	case "ID":
		defaultType = "string"
		if resultModel.isList {
			defaultType = "[]string"
		}
	case "Upload":
		defaultType = "graphql.Upload"
		if resultModel.isList {
			defaultType = "[]graphql.Upload"
		}
	default:
		resultModel.Name = reApi.ReplaceAllString(resultModel.Name, "API")
		if resultModel.isList {
			defaultType = "[]*model." + resultModel.Name
		} else {
			defaultType = "model." + resultModel.Name
		}
	}
	result := "\tresult := " + defaultType + "{}\n"
	return result
}

func removeDuplicateStr(strSlice []string) []string {
	allKeys := make(map[string]bool)
	var list []string
	for _, item := range strSlice {
		if _, value := allKeys[item]; !value {
			allKeys[item] = true
			list = append(list, item)
		}
	}
	return list
}

func saveBackup(actionPackage string, pathFilename string, bkpItem []string) {
	fmt.Println("Backup", pathFilename+"_bkp.go")

	var regexhasPackage = regexp.MustCompile(`package`)

	isNotImplemented := regexIsImplemented.MatchString(bkpItem[0])
	if !isNotImplemented {
		timeNow := time.Now()
		timeNowStr := timeNow.Format("02012006_150405")

		f, err := os.OpenFile(pathFilename+"_bkp.go", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			panic(err)
		}
		defer func(f *os.File) {
			_ = f.Close()
		}(f)

		actionName := bkpItem[2]
		var regexFindAction = regexp.MustCompile(actionName)
		action := regexFindAction.ReplaceAllString(bkpItem[0]+"\n\n", "BKP__"+timeNowStr+"_"+actionName)

		readFile, err := os.ReadFile(pathFilename + "_bkp.go") // just pass the file name
		if err != nil {
			fmt.Print(err)
		}

		fileStr := string(readFile) // convert content to a 'string'
		render := action
		if !regexhasPackage.MatchString(fileStr) {
			render = "package " + actionPackage + "\n\n"
			render = render + "import (\n\"fmt\"\n)\n\n" + action
		}

		if _, err = f.WriteString(render); err != nil {
			panic(err)
		}
	}
}

func typeRequerid(model ArgModel) string {
	var result string
	switch model.Type {
	case "Int":
		result = "int"
	case "String":
		result = "string"
	case "ID":
		result = "string"
	case "Upload":
		result = "graphql.Upload"
	default:
		result = "model." + model.Type
	}
	if !model.isRequerid {
		result = "*" + result
	}

	nameSplit := camelcase.Split(result)
	result = ""
	for _, namePart := range nameSplit {
		if (namePart == "id" || namePart == "Id") || (namePart == "api" || namePart == "Api") {
			result += strings.ToUpper(namePart)
		} else {
			result += namePart
		}
	}
	return result
}

func main() {
	createFileModel("query")
	createFileModel("mutation")
	createListResult()
	renderResolver()
	renderService()
}
