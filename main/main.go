package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"
)

func main() {
	type Fatura struct {
		Id          int
		DataCompra  string
		NomeCartao  string
		FinalCartao string
		Categoria   string
		Descricao   string
		Parcela     string
		Valor       string
	}

	var list []Fatura
	file, err := os.Open("Fatura_2025-07-20.csv")

	if err != nil {
		/* fmt.Print("CAIUUUU NO ERRROOOO")
		fmt.Println(err) */
	}
	reader := csv.NewReader(file)
	reader.Comma = ';'

	defer file.Close()

	header := []string{}
	counter := 0

	for {
		i, err := reader.Read()

		if err == io.EOF {
			break
		}

		if err != nil {
			/* fmt.Print("CAIUUUU NO ERRROOOO") */
			/* fmt.Println(i) */
			/* fmt.Println(err) */
		}

		if len(header) == 0 {
			fmt.Println("O que ta inserindo")
			fmt.Println(strings.Split(i[0], ";")[0])
			for colName := 0; colName < len(strings.Split(i[0], ";")); colName++ {
				res := strings.Split(i[0], ";")[colName]
				header = append(header, strings.TrimSpace(res))
			}

		} else {
			fatura := Fatura{
				Id:          counter,
				DataCompra:  i[0],
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
	fmt.Println(list)

}
