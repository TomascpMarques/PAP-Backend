package endpointfuncs

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/tomascpmarques/PAP/backend/robinservicoequipamento/loggers"
	"github.com/tomascpmarques/PAP/backend/robinservicoequipamento/mongodbhandle"
	"github.com/tomascpmarques/PAP/backend/robinservicoequipamento/resolvedschema"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var mongoParams = mongodbhandle.MongoConexaoParams{
	URI: "mongodb://0.0.0.0:27018/",
}

// MongoClient cliente com a conexão à instancia mongo
var MongoClient = mongodbhandle.CriarConexaoMongoDB(mongoParams)

// PingServico responde que o serviço está online
func PingServico(name string) map[string]interface{} {
	result := make(map[string]interface{})

	result["status"] = fmt.Sprintf("Hello %s, I'm alive and OK", name)
	return result
}

// AdicionarRegisto Adiciona um registo numa base de dados e coleção especifícada
func AdicionarRegisto(registoMeta map[string]interface{}, item map[string]interface{}, token string) (result map[string]interface{}) {
	result = make(map[string]interface{})

	// if VerificarTokenUser(token) != "OK" {
	// 	fmt.Println("Erro: A token fornecida é inválida ou expirou")
	// 	result["erro"] = "A token fornecida é inválida ou expirou"
	// 	return result
	// }

	/* Defenição do novo registo */
	// Verificação da meta do registo
	if err := VerificarCamposMetaRegisto(registoMeta); err != nil {
		loggers.ServerErrorLogger.Println("Error: ", err)
		result["erro"] = err.Error()
		return
	}
	// Verificação do corpo do novo registo
	if !reflect.ValueOf(item).IsValid() || reflect.ValueOf(item).IsZero() {
		loggers.ServerErrorLogger.Println("Error: ", "O corpo do registo não pode ser nulo ou inválido")
		result["erro"] = "O corpo do registo não pode ser nulo ou inválido"
		return
	}

	metaRegisto := resolvedschema.RegistoMetaParaStruct(&registoMeta)
	registo := resolvedschema.Registo{
		Meta: &metaRegisto,
		Body: item,
	}

	// Get the mongo colection
	mongoCollection := MongoClient.Database("recursos").Collection(registo.Meta.Tipo)

	// Insser um registo na coleção e base de dados especificada
	_, err := mongodbhandle.InserirUmRegisto(registo, mongoCollection, 10)

	if err != nil {
		loggers.ServerErrorLogger.Println("Error: ", err)
		result["erro"] = err
		return
	}

	loggers.MongoDBLogger.Println("Registo adicionado ao sistema!")
	result["sucesso"] = true
	return
}

// QueryRegistoJSON Executa um query nos registos encontrados que satisfazêm o filtro de pesquisa
// Devolve só os campos pedidos dos registos encontrados, no formato { "key1.key2.key3.value1": result1 }
func QueryRegistoJSON(campos map[string]interface{}, colecao string, token string) (result map[string]interface{}) {
	result = make(map[string]interface{})

	// if VerificarTokenUser(token) != "OK" {
	// 	fmt.Println("Erro: A token fornecida é inválida ou expirou")
	// 	result["erro"] = "A token fornecida é inválida ou expirou"
	// 	return result
	// }

	// Define o query a usar nas buscas e a colecao alvo
	query := resolvedschema.QueryParaStruct(&campos)
	colecaoAlvo := GetColecaoFromDB(colecao)

	// Busca os registos da coleção, que igualêm os resultados
	registos, err := GetRegistosDaColecao(query.Campos, colecaoAlvo)
	if err != nil {
		loggers.ServerErrorLogger.Println("Error: ", err)
		result["erro"] = err
		return
	}

	// Extrai os campos pedidos
	var records = RunQuerysOnRecords(query, registos)

	// Verifica se os resultados do query são válidos (!= 0)
	if reflect.ValueOf(records).IsZero() {
		loggers.ServerErrorLogger.Println("Error: ", "Erro ao extrair os campos pedidos")
		result["erro"] = "Erro ao extrair os campos pedidos"
		return
	}

	loggers.ResolverLogger.Println("Sucesso, campos extraidos com sucesso!")
	result["registos"] = map[string]interface{}{
		colecao: records,
	}
	return
}

// BuscarTodosOsRegistosColecao Faz o que o título da função descreve
func BuscarTodosOsRegistosColecao(colecao string, token string) (retorno map[string]interface{}) {
	retorno = make(map[string]interface{})

	// if VerificarTokenUser(token) != "OK" {
	// 	fmt.Println("Erro: A token fornecida é inválida ou expirou")
	// 	result["erro"] = "A token fornecida é inválida ou expirou"
	// 	return result
	// }

	// Define o query a usar nas buscas e a colecao alvo
	colecaoAlvo := GetColecaoFromDB(colecao)
	results, err := colecaoAlvo.Find(context.TODO(), bson.M{}, options.Find())
	if err != nil {
		loggers.ServerErrorLogger.Println("Error: ", err)
		retorno["erro"] = "Erro ao buscar todos os registos"
		return
	}

	registos := make([]map[string]interface{}, 0)
	err = results.All(context.TODO(), &registos)
	if err != nil {
		loggers.ServerErrorLogger.Println("Error: ", err)
		retorno["erro"] = err
		return
	}

	loggers.ResolverLogger.Println("Sucesso, registos encontrados!")
	retorno["registos"] = registos
	return
}

func BuscarTodosRegistosBD(token string) (retorno map[string]interface{}) {
	retorno = make(map[string]interface{})

	// if VerificarTokenUser(token) != "OK" {
	// 	fmt.Println("Erro: A token fornecida é inválida ou expirou")
	// 	result["erro"] = "A token fornecida é inválida ou expirou"
	// 	return result
	// }

	// Busca a base de dados usada pelo sistema
	dataBase := ReturnDB()
	// Extrai todos os nome de coleções existentes
	colecoesNomes, err := dataBase.ListCollectionNames(context.TODO(), bson.M{})
	if err != nil {
		fmt.Println("erro: ", err)
		retorno["erro"] = err.Error()
		return
	}

	// Prepara a variavél onde todos os registos vão ser guardados
	registos := make(map[string]interface{})
	for _, collName := range colecoesNomes {
		// Busca o conteudo da coleção alvo
		colecaoAlvo := GetColecaoFromDB(collName)
		// Busca todos os items presentes na coleção
		results, err := colecaoAlvo.Find(context.TODO(), bson.M{}, options.Find())
		if err != nil {
			loggers.ServerErrorLogger.Println("Error: ", err)
			retorno["erro"] = "Erro ao buscar todos os registos"
			return
		}

		// Variavél temporária para armazenar todos os registos da coleção
		temp := make([]map[string]interface{}, 0)
		// Descodifica todos os registos para a var temp
		err = results.All(context.TODO(), &temp)
		if err != nil {
			loggers.ServerErrorLogger.Println("Error: ", err)
			retorno["erro"] = err.Error()
			return
		}
		// Atribui os registos a um mapa representante da coleção, existente na var registos
		registos[collName] = temp
	}

	// Retorna os registos se tudo correr bêm
	loggers.ResolverLogger.Println("Sucesso, registos encontrados e enviados!")
	retorno["registos"] = registos
	return
}

// ApagarRegistoPorID Apaga um registo pelo seu ObjectID, na bd e coleção fornecida
func ApagarRegistoPorID(colecao string, idItem string, token string) map[string]interface{} {
	result := make(map[string]interface{})

	// if VerificarTokenUser(token) != "OK" {
	// 	fmt.Println("Erro: A token fornecida é inválida ou expirou")
	// 	result["erro"] = "A token fornecida é inválida ou expirou"
	// 	return result
	// }

	// Converte o ID de uma String para um ObjectID
	idOBJ, err := primitive.ObjectIDFromHex(idItem)
	if err != nil {
		loggers.ServerErrorLogger.Println()
		fmt.Println("Error: ID de registo não pode ser convertido")
		result["erro"] = err
		return result
	}

	// Set-up do filtro
	filter := bson.M{"_id": idOBJ}

	// Get collection
	coll := MongoClient.Database("recursos").Collection(colecao)
	cntx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	// Operação de delete de um só registo
	item, err := coll.DeleteOne(cntx, filter, options.Delete())
	defer cancel()
	if err != nil {
		loggers.ServerErrorLogger.Println("Erro: Não foi possível apagar o registo de id: ", idItem)
		result["erro"] = "Error: Não foi possível apagar o item de registo:" + idItem
		return result
	}

	result["registo_apagado"] = item
	return result
}

// AtualizarRegistoDeItem Na bd e coleção escolhida, o registo de id idItem
// vai ser atualizado para os valores especificados em item
func AtualizarRegistoDeItem(colecao string, idItem string, item map[string]interface{}, token string) map[string]interface{} {
	result := make(map[string]interface{})

	// if VerificarTokenUser(token) != "OK" {
	// 	fmt.Println("Erro: A token fornecida é inválida ou expirou")
	// 	result["erro"] = "A token fornecida é inválida ou expirou"
	// 	return result
	// }

	// Converte o ID de uma String para um ObjectID
	idOBJ, err := primitive.ObjectIDFromHex(idItem)
	if err != nil {
		loggers.ServerErrorLogger.Println()
		fmt.Println("Error: ID de registo não pode ser convertido")
		result["erro"] = err
		return result
	}

	// Set-up do filtro
	filter := bson.M{"_id": idOBJ}

	// Get collection
	coll := MongoClient.Database("recursos").Collection(colecao)
	cntx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	// Atualiza o item através do map especificado nos params
	matchCount, err := coll.UpdateOne(cntx, filter, bson.M{"$set": item}, options.MergeUpdateOptions())
	fmt.Println(matchCount)
	defer cancel()
	if err != nil {
		loggers.ServerErrorLogger.Println("Erro: ", err, " | registo de id: ", idItem)
		result["erro"] = "Error: registo:" + idItem
		return result
	}

	result["atualizacoes"] = matchCount
	return result
}
