package loginregistohandlers

import (
	"encoding/json"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/tomascpmarques/PAP/backend/robinservicologin/loggers"
	"github.com/tomascpmarques/PAP/backend/robinservicologin/redishandle"
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
	admin - md5 > 532f1f7e5e4ae1475835c4978696c1e3
			clear-text > @@Robin_Gestao_Admin2#0#2#0!!
*/

// User - Epecifica os dados que definem um utilizador
type User struct {
	Username   string `json:"user,omitempty"`
	Password   string `json:"passwd,omitempty,-"`
	Permissoes int    `json:"perms,omitempty"`
	JWT        string `json:"jwt,omitempty"`
}

// CriarNovoUser através de um username, password e permissões cria e devolve um novo utilizador (struct)
func CriarNovoUser(user string, password string, perms int) User {
	return User{
		Username:   user,
		Password:   password,
		Permissoes: perms,
	}
}

// CriarUserJWT Cria as JWT Token para cada utilisador, a partir dos dados da struct User
func (user User) CriarUserJWT() *jwt.Token {
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user":  user.Username,
		"perms": user.Permissoes,
		"iss":   "Robin-Servico-Auth",
		"exp":   time.Now().Add(time.Minute * 40).Unix(),
	})
	return jwtToken
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
		admin := CriarNovoUser("admin", "532f1f7e5e4ae1475835c4978696c1e3", 2)
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
