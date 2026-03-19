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
	"github.com/playwright-community/playwright-go"
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

	connStr := "user=postgres password=masterKey host=localhost port=5432 dbname=bigo sslmode=disable"

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

		if counter != 0 {

			aiClient := openai.NewClient(apiKey)

			ctx := context.Background()

			fmt.Println("QUALLL A EMPRESAA ", i[4])

			resp, err := aiClient.CreateChatCompletion(
				ctx,
				openai.ChatCompletionRequest{
					Model: openai.GPT4oMini,
					Messages: []openai.ChatCompletionMessage{
						{
							Role:    openai.ChatMessageRoleUser,
							Content: "QUal o nome correto dessa empresa? ME devolva apenas o nome na resposta sem texto adicional SUBSTITUINDO OS ESPAÇOS POR ESTE CARACTERE +" + i[4],
						},
					},
				},
			)
			if err != nil {
				log.Println("Erro OpenAI:", err)
			}
			response := resp.Choices[0].Message.Content

			fmt.Println(response)

			pw, _ := playwright.Run()

			browser, _ := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
				Headless: playwright.Bool(true),
			})

			page, _ := browser.NewPage()

			urlEmployeInfo := fmt.Sprintf("http://cnpj.info/busca?q=%s", response)

			page.Goto(urlEmployeInfo)

			html, _ := page.Content()

			re := regexp.MustCompile(`href="\/(\d{14})"`)
			res := re.FindStringSubmatch(html)

			fixCnpj := cleanCNPJ(res[1])

			fmt.Println("CNP", fixCnpj)

			url := fmt.Sprintf("https://api.opencnpj.org/%s", fixCnpj)

			fmt.Print(url)
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

			fmt.Println("Resposta da API:", string(jsonRes))
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

			cnae := formatCNAE(empresa.CNAEPrincipal)
			var estimatedProfit float64

			for i, row := range rows {
				if i == 0 || len(row) < 6 {
					continue
				}

				codigo := strings.TrimSpace(row[0])
				ano := strings.TrimSpace(row[2])

				if codigo == cnae && ano == "2022" {
					fmt.Println("CNAE:", cnae)
					fmt.Println("Quantidade de CNPJ:", row[3])
					fmt.Println("Receita Bruta:", row[4])

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
					fmt.Printf("Lucro Estimado para CNAE %s: %.2f\n", cnae, estimatedProfit)

					break
				}
			}

			fmt.Println("Lucro estimado", estimatedProfit)
			layout := "02/01/2006"

			date, err := time.Parse(layout, i[0])
			if err != nil {
				fmt.Println("Erro ao converter depois colocar dare ali:", err)
				return
			}

			fmt.Println("O QUE VEMMM NO PARCELA", i[5])

			fmt.Println(counter)
			fmt.Println(i[1])

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
