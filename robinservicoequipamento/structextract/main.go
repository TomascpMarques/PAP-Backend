package structextract

import (
	"reflect"
	"strings"
)

// CustomExtractSchema -
type CustomExtractSchema map[string][]string

/*
	var custom = CustomExtractSchema{
		"PC":         {"PC", "Nome,ID"},
		"Info":       {"PC", "Sala"},
		"Manutencao": {"Info", "Status"},
		"Ultima":     {"Manutencao", "ID"},
		"AAAA":       {"Ultima", "ID"},
	}

	Key do mapa ex: «"PC":» representa o objeto a utilizar na extração,
	{"PC", "Nome,ID"} -> o 1º elemento representa o "parent" do elemento atual,
						 a string seguinte refer os campos a extrair.
*/

/*
ExtrairCamposEspecificosStruct :
	Extrai os campos especificados da struct fornecida,
	utiliza um map[string][]string para especificar os campos a tirar da struct.

Params:
	-> estrutura interface{}, struct src para extrair os dados
	-> listaCampos CustomExtractSchema (map[string][]string), especifica os acmpos a extrair da struct

Notas:
	Utiliza-se uma interface{} como src dos valores para se poder utilisar qualquer struct passada como parametro.
	A função é recurssiva para chegar a todas as structs presentes, até ás mais profundas.
	Os nomes dos campos da struct devem ser iguais aos defenidos nos campos das structs i.e.: campo struct PC é ifual no query «"PC":»
*/
func ExtrairCamposEspecificosStruct(estrutura interface{}, listaCampos map[string][]string) (retorno map[string]interface{}) {
	retorno = make(map[string]interface{})

	// Valore refletido da estrutura passada
	estruturaReflectValue := reflect.ValueOf(estrutura)
	// Tipo refletido da estrutura passada
	estruturaReflectType := reflect.TypeOf(estrutura)

	// Iterar por todos os campos (à superficíe) da struct
	for i := 0; i < estruturaReflectValue.NumField(); i++ {
		// Verifica se o campo foi especificado na lista com os valores a extrair
		if _, existe := listaCampos[estruturaReflectType.Name()]; existe {
			// Verifica se o campo atual da estrutura foi indicado para extrasão
			if strings.Contains(listaCampos[estruturaReflectValue.Type().Name()][1], estruturaReflectType.Field(i).Name) {
				// Adiciona o valor extraido ao retorno da função
				retorno[estruturaReflectType.Field(i).Name] = estruturaReflectValue.Field(i).Interface()
			}

			// Verifica se o campo atual é mais uma estrutura, (parte recurssiva da função)
			if estruturaReflectType.Field(i).Type.Kind().String() == "struct" {
				// Reflexão dos valores presentes na estrutura embutida
				estruturaEmbutida := estruturaReflectValue.Field(i)
				nomeEstruturaEmbutida := estruturaReflectType.Field(i)

				// Indica o index onde o parent da struct começa na string ex: main.PC, ignora tudo até ao 1º "." +1
				pkgNameEndIndex := strings.Index(estruturaReflectValue.Type().String(), ".") + 1
				structParent := estruturaReflectValue.Type().String()[pkgNameEndIndex:]

				// Se a estrutura não estiver vazia, não null, e for um campo defenido na lista através dos parents de cada campo
				// Chama esta função de novo para ler e extrair os valores da struct
				if !estruturaEmbutida.IsZero() && structParent == listaCampos[nomeEstruturaEmbutida.Name][0] {
					retorno[estruturaEmbutida.Type().Name()] = ExtrairCamposEspecificosStruct(estruturaEmbutida.Interface(), listaCampos)
				}
			}
		}
	}

	return
}
