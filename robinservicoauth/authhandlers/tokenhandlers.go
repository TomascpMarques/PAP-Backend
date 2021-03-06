package authhandlers

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/tomascpmarques/PAP/backend/robinservicoauth/loggers"
	"github.com/tomascpmarques/PAP/backend/robinservicoauth/redishandle"
)

const (
	// ROOT utilizador com as permissões mais elevadas
	ROOT = iota + 1 // ! ainda não foi implementado este utilizador, valor 1
	// ADMIN neste momento o tipo de utilizador com mais previlégios, valor 2
	ADMIN
	// USER previlégios básicos, valor 3
	USER
)

/*
	Credeenciais default do admin robin:
	admin - md5 > 		 532f1f7e5e4ae1475835c4978696c1e3
			sha256 > 	 027aede4e00bfe45724dc54c740fa6d57109dc1ba661edf99f93728f6c7371e4
			clear-text > @@Robin_Gestao_Admin2#0#2#0!!
*/

// User - Epecifica os dados que definem um utilizador
type User struct {
	JWT        string `json:"jwt,omitempty"`
	Username   string `json:"user,omitempty"`
	Password   string `json:"passwd,omitempty"`
	Permissoes int    `json:"perms,omitempty"`
}

// CriarNovoUser através de um username, password e permissões cria e devolve um novo utilizador (struct)
func CriarNovoUser(user string, password string, perms int) User {
	return User{
		Username:   user,
		Password:   password,
		Permissoes: perms,
	}
}

type UserFuncs interface {
	CriarJWTAuth() *jwt.Token
}

// CriarJWTAuth Cria as JWT Token para cada utilisador, a partir dos dados da struct User
func (user User) CriarJWTAuth() *jwt.Token {
	jwtAuth := jwt.NewWithClaims(jwt.SigningMethodHS512, jwt.MapClaims{
		"user":  user.Username,
		"perms": user.Permissoes,
		"iss":   "Robin-Servico-Auth",
		"exp": time.Now().Add(time.Hour * 9).Unix(),
	})
	return jwtAuth
}

// GetUserParaValorStruct Busca um utilisador pelo nome na base de dados
func GetUserParaValorStruct(username string) (User, error) {
	// Busca o registo correspondente ao user passado nos parametros
	userCompare, err := redishandle.GetRegistoBD(&RedisClientDB, username, 0)
	if err != nil {
		loggers.LoginRedisLogger.Println("Erro: ", err)
		return User{}, err
	}

	// Cria a estrutura User para o registo, descodifica o conteúdo de valores json
	var registo User
	err = json.Unmarshal([]byte(userCompare), &registo)
	if err != nil {
		loggers.LoginRedisLogger.Println("Erro: ", err)
		return User{}, err
	}

	return registo, nil
}

// VerificarAdminFirstBoot verifica se o utilizador admin da backend robin existe, se não existir cria esse user
// com as credenciais default
func VerificarAdminFirstBoot() bool {
	// Tenta encontrar o registo do admin, se não o encontrar cria-o
	_, err := redishandle.GetRegistoBD(&RedisClientDB, "admin", 0)
	if err != nil {
		loggers.LoginAuthLogger.Println("O utilizador administrador não existe...")
		// Cria a struct de utilisador para o admin
		admin := CriarNovoUser("admin", "027aede4e00bfe45724dc54c740fa6d57109dc1ba661edf99f93728f6c7371e4", 2)
		registoUserJSON, err := json.Marshal(&admin)
		if err != nil {
			loggers.LoginRedisLogger.Println("Erro: ", err)
			return false
		}
		// Inssere o administrador
		redishandle.SetRegistoBD(&RedisClientDB, redishandle.RegistoRedisDB{
			Key:    admin.Username,
			Valor:  registoUserJSON,
			Expira: 0,
		}, 0)

		return true
	}
	return false
}

// VerificarTokenUser verifica se a token passada é válida, logo vê se já expirou
// se o modo de assinatura é o correto, e se o emissor é o servidor de autenticação
func VerificarTokenUser(userToken string) string {
	token, err := jwt.Parse(userToken, func(token *jwt.Token) (interface{}, error) {
		// valida o metodo de assinatura da key
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("metodo de assinatura inesperado: %v", token.Header["alg"])
		}
		// assinaturaSecretaServer é o []byte que contem o segredo de assinatura
		return assinaturaSecretaServer, nil
	})
	// Se a token for assinada por outro metodo ou a key for diferente dá erro
	if err != nil {
		return fmt.Sprint(err)
	}

	// Verifica que a token é válida e assinada pelo servidor de login
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid && claims["iss"] == "Robin-Servico-Auth" {
		return "OK"
	}
	return "Token inválida ou expirada"
}

// VerificarTokenAdmin verifica tudo o que a função VerificarTokenUser verifica,
// e ainda verifica se o utilisador é o administrador
func VerificarTokenAdmin(userToken string) string {
	token, err := jwt.Parse(userToken, func(token *jwt.Token) (interface{}, error) {
		// valida o metodo de assinatura da key
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("metodo de assinatura inesperado: %v", token.Header["alg"])
		}

		// hmacSampleSecret é o []byte que contem o segredo de assinatura
		return assinaturaSecretaServer, nil
	})
	// Se a token for assinada por outro metodo ou a key for diferente dá erro
	if err != nil {
		return fmt.Sprint(err)
	}

	// Verifica que a token é válida e assinada pelo servidor de login
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid && claims["iss"] == "Robin-Servico-Auth" &&
		claims["perms"].(float64) == 2 {
		return "OK"
	}
	return "Token inválida ou expirada"
}

// DevolveTokenClaims Devolve os valores do body da token (claims)
func DevolveTokenClaims(userToken string) map[string]interface{} {
	token, err := jwt.Parse(userToken, func(token *jwt.Token) (interface{}, error) {
		// valida o metodo de assinatura da key
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, nil
		}
		// hmacSampleSecret é o []byte que contem o segredo de assinatura
		return assinaturaSecretaServer, nil
	})
	// Se a token for assinada por outro metodo ou a key for diferente dá erro
	if err != nil {
		return nil
	}

	return token.Claims.(jwt.MapClaims)
}

// VerificarTokenReAuth Verifica a token de reload de autenticação do user
func VerificarTokenReAuth(reAuthToken string, tokenAuth string) string {
	token, err := jwt.Parse(reAuthToken, func(token *jwt.Token) (interface{}, error) {
		// valida o metodo de assinatura da key
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("metodo de assinatura inesperado: %v", token.Header["alg"])
		}

		// hmacSampleSecret é o []byte que contem o segredo de assinatura
		return assinaturaSecretaServer, nil
	})
	// Se a token for assinada por outro metodo ou a key for diferente dá erro
	if err != nil {
		return fmt.Sprint(err)
	}

	// Verifica que a token é válida e assinada pelo servidor de login
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid && claims["iss"] == "Robin-Servico-Auth" &&
		claims["typ"] == "reauth" {
		return "OK"
	}
	return "Token inválida ou expirada"
}
