package main

import (
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"

	"os"
	"regexp"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/sashabaranov/go-openai"

	"github.com/xuri/excelize/v2"
)

func cleanCNPJ(cnpj string) string {
	re := regexp.MustCompile(`\D`)
	return re.ReplaceAllString(cnpj, "")
}

func formatCNAE(cnae string) string {
	re := regexp.MustCompile(`^(\d{4})(\d)(\d{2})$`)
	return re.ReplaceAllString(cnae, "$1-$2/$3")
}

func main() {
	type Fatura struct {
		Id          int
		DataCompra  time.Time
		NomeCartao  string
		FinalCartao string
		Categoria   string
		Descricao   string
		Parcela     string
		Valor       string
	}

	type Empresa struct {
		CNAEPrincipal string `json:"cnae_principal"`
	}

	type CnpjRes struct {
		Resultados []struct {
			CNPJBase string `json:"cnpj_base"`
			Nome     string `json:"nome_empresarial"`
		} `json:"resultados_paginacao"`
	}

	type finalCnpjRes struct {
		Res []struct {
			CompleteCnpj string `json:"cnpj"`
		} `json:"resultados_paginacao"`
	}

	type Info struct {
		Quantidade string
		Receita    string
		Ano        string
	}

	var list []Fatura
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Erro ao carregar .env")
	}

	apiKey := os.Getenv("openAiKey")

	file, err := os.Open("Fatura_2025-07-20.csv")

	if err != nil {
		fmt.Println(err)
	}

	reader := csv.NewReader(file)
	reader.Comma = ';'

	defer file.Close()
	counter := 0

	connStr := os.Getenv("db")

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}

	for {
		i, err := reader.Read()

		if err == io.EOF {
			break
		}

		if err != nil {
			fmt.Println(err)
		}

		fmt.Println("LINHA", counter)

		if counter != 0 {

			aiClient := openai.NewClient(apiKey)

			ctx := context.Background()

			resp, err := aiClient.CreateChatCompletion(
				ctx,
				openai.ChatCompletionRequest{
					Model: openai.GPT4oMini,
					Messages: []openai.ChatCompletionMessage{

						{
							Role:    "system",
							Content: os.Getenv("AISystemPrompt"),
						},
						{
							Role:    "user",
							Content: os.Getenv("AIUserPrompt") + i[4],
						},
					},
				},
			)

			if err != nil {
				log.Println("Erro OpenAI:", err)
			}
			response := resp.Choices[0].Message.Content

			firtsCnpjNumbers, err := http.Get("https://api.cnpj.pw/razao_social/" + response)

			if err != nil {
				log.Println("Erro ao obter CNPJ:", err)
				continue
			}

			defer firtsCnpjNumbers.Body.Close()

			var data CnpjRes
			var finalCnpj finalCnpjRes

			json.NewDecoder(firtsCnpjNumbers.Body).Decode(&data)

			if len(data.Resultados) > 0 {

				geFinalCnpj, err := http.Get("https://api.cnpj.pw/cnpj_base/" + data.Resultados[0].CNPJBase)

				if err != nil {
					log.Println("Erro ao obter CNPJ completo:", err)
					continue
				}

				json.NewDecoder(geFinalCnpj.Body).Decode(&finalCnpj)

				defer geFinalCnpj.Body.Close()

			}
			fmt.Println("ai.response", response)

			var fixCnpj string
			var cnae string
			var estimatedProfit float64

			if len(finalCnpj.Res) > 0 {
				fmt.Println(finalCnpj.Res[0].CompleteCnpj)

				if len(finalCnpj.Res) == 0 {
					log.Println("CNPJ não encontrado para a empresa:", response)
					continue
				}

				fixCnpj = cleanCNPJ(finalCnpj.Res[0].CompleteCnpj)

				fmt.Println("Cnpj tratado", fixCnpj)

				url := fmt.Sprintf("https://api.opencnpj.org/%s", fixCnpj)

				reqApi, err := http.NewRequest("GET", url, nil)
				if err != nil {
					log.Println("Erro ao criar request API:", err)
					continue
				}

				respJson, err := client.Do(reqApi)
				if err != nil {
					log.Println("Erro na API:", err)
					continue
				}

				jsonRes, err := io.ReadAll(respJson.Body)
				if err != nil {
					log.Fatal(err)
				}

				var empresa Empresa

				err = json.Unmarshal(jsonRes, &empresa)
				if err != nil {
					log.Println("Erro ao fazer unmarshal:", err)
					continue
				}

				f, err := excelize.OpenFile("Tab_07_Subclasse_Ano.xlsx")
				if err != nil {
					fmt.Println(err)
					return
				}
				defer f.Close()

				rows, err := f.GetRows("Tabela_07")
				if err != nil {
					fmt.Println(err)
					return
				}

				cnae = formatCNAE(empresa.CNAEPrincipal)

				for i, row := range rows {
					if i == 0 || len(row) < 6 {
						continue
					}

					codigo := strings.TrimSpace(row[0])
					ano := strings.TrimSpace(row[2])

					if codigo == cnae && ano == "2022" {

						qtdCnpj, err := strconv.ParseFloat(strings.TrimSpace(strings.ReplaceAll(row[3], ",", "")), 64)
						if err != nil {
							fmt.Println("Erro:", err)
							return
						}

						totalProfit, err := strconv.ParseFloat(strings.TrimSpace(strings.ReplaceAll(row[4], ",", "")), 64)
						if err != nil {
							fmt.Println("Erro:", err)
							return
						}

						estimatedProfit = math.Round((totalProfit/qtdCnpj)*100) / 100

						break
					}
				}

			}

			layout := "02/01/2006"

			date, err := time.Parse(layout, i[0])
			if err != nil {
				fmt.Println("Erro ao converter depois colocar dare ali:", err)
				return
			}

			_, err = db.Exec(
				"INSERT INTO userCreditData (lineId, name) VALUES ($1, $2)",
				counter,
				i[1],
			)

			if err != nil {
				fmt.Println("ERRO PRA INSERIR NA USER", err)
				return
			}

			_, err = db.Exec(
				"INSERT INTO card (lineId, lastNumbers) VALUES ($1, $2)",
				counter,
				i[2],
			)

			if err != nil {
				fmt.Println("ERRO PRA INSERIR NA CARD", err)
				return
			}

			fmt.Println()

			_, err = db.Exec(
				"INSERT INTO category (lineId, description) VALUES ($1, $2)",
				counter,
				i[3],
			)

			if err != nil {
				fmt.Println("ERRO PRA INSERIR NA CATEGORY", err)
				return
			}

			_, err = db.Exec(
				"INSERT INTO fornecedor (name, cnpj, cnae, estimatedProfit) VALUES ($1, $2, $3, $4)",
				i[4],
				fixCnpj,
				cnae,
				estimatedProfit,
			)

			if err != nil {
				fmt.Println("ERRO PRA INSERIR NA FORNECEDOR", err)
				return
			}

			_, err = db.Exec(
				"INSERT INTO transactionCreditData (lineId, cnpj, value, transactionDate) VALUES ($1, $2, $3, $4)",
				counter,
				fixCnpj,
				i[len(i)-1],
				date,
			)

			if err != nil {
				fmt.Println("ERRO PRA INSERIR NA transaction", err)
				return
			}

			var typePayment string

			if i[5] == "Única" {
				typePayment = "debit"
			} else {
				typePayment = "credit"
			}

			_, err = db.Exec(
				"INSERT INTO payment (lineId, type, installment) VALUES ($1, $2, $3)",
				counter,
				typePayment,
				i[5],
			)

			if err != nil {
				fmt.Println("ERRO PRA INSERIR NA payment", err)
				return
			}

			fatura := Fatura{
				Id:          counter,
				DataCompra:  date,
				NomeCartao:  i[1],
				FinalCartao: i[2],
				Categoria:   i[3],
				Descricao:   i[4],
				Parcela:     i[5],
				Valor:       i[len(i)-1],
			}

			list = append(list, fatura)
		}

		counter++

	}

}
