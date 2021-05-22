package endpointfuncs

import (
	"context"
	"errors"
	"reflect"
	"regexp"
	"time"

	"github.com/tomascpmarques/PAP/backend/robinservicovideoshare/loggers"
	"github.com/tomascpmarques/PAP/backend/robinservicovideoshare/resolvedschema"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoDBOperation Struct com o setup minímo para fazer uma oepração na BDs
type MongoDBOperation struct {
	Colecao    *mongo.Collection
	Cntxt      context.Context
	CancelFunc context.CancelFunc
	Filter     interface{}
}

// Setup Evita mais lihas desnecessárias e repetitivas para poder-se usar a coleção necessaria
func SetupColecao(dbName, collName string) (defs MongoDBOperation) {
	defs.Colecao = MongoClient.Database(dbName).Collection(collName)
	defs.Cntxt, defs.CancelFunc = context.WithTimeout(context.Background(), time.Second*10)
	return
}

func VerificarVideoShareMetaData(videoShare map[string]interface{}) error {
	camposObrgt := []string{
		"titulo",
		"url",
		"criador",
	}

	// Verifica se têm os campos obrigatórios defenidos
	for _, v := range camposObrgt {
		if _, existe := videoShare[v]; !existe {
			return errors.New("o campo <" + v + ">, não está presente na info da videoshare")
		}
	}

	// Verifica o tamanho do título
	if len(videoShare["titulo"].(string)) < 4 {
		return errors.New("o título do video é demasiado curto, deve ter no minímo 4 caracteres")
	}

	// Verifica que o url é válido
	url := (videoShare["url"].(string))
	regex := regexp.MustCompile(`(?m)https://youtu\.be/[a-zA-Z0-9_]+`).FindAllString(url, -1)
	if reflect.ValueOf(regex).IsZero() {
		return errors.New("o link fornecido não é válido")
	}

	return nil
}

// TrimURL Larga a parte desnecessária do url, >https://youtu\.be/<
func TrimURL(url string) string {
	return url[18:]
}

// Adiciona os dados da video-share à base de dados
func AdicionarVideoShareDB(videoShare *resolvedschema.Video) error {
	colecao := SetupColecao("videoshares", "videos")

	// Inserção do registo na BD
	result, err := colecao.Colecao.InsertOne(colecao.Cntxt, videoShare, options.InsertOne())
	defer colecao.CancelFunc()
	if err != nil {
		loggers.DbFuncsLogger.Println("Não foi possivél adicionar a video-share na base de dados: ", err.Error())
		return err
	}

	// Verifica se o id inserido é != de nil, para verificar a inserção do registo
	if result.InsertedID == "" {
		loggers.DbFuncsLogger.Println("Não foi possivél criar o registo na base de dados")
		return errors.New("não foi possivél criar o registo da video-share na base de dados")
	}

	return nil
}