# ads-projeto-bi

Projeto para disciplina de business inteligence ADS/26-01
Combina o projeto de exemplo em construção nas aulas e o início do trabalho final 

Tabelas de organização

CREATE TABLE userCreditData (
    id SERIAL PRIMARY KEY,
    lineId INTEGER NOT NULL,
    name VARCHAR
);

CREATE TABLE card (
    id SERIAL PRIMARY KEY,
    lineId INTEGER NOT NULL,
    lastNumbers VARCHAR(10)
);

CREATE TABLE category (
    id SERIAL PRIMARY KEY,
    lineId INTEGER NOT NULL,
    description VARCHAR
);

CREATE TABLE fornecedor (
    id SERIAL PRIMARY KEY,
    name VARCHAR,
	cnpj VARCHAR,
	cnae VARCHAR,
	estimatedProfit NUMERIC(15,2)
	
);


CREATE TABLE transactionCreditData (
    id SERIAL PRIMARY KEY,
    lineId INTEGER NOT NULL,
	cnpj VARCHAR,
    value NUMERIC(10,2),
    transactionDate DATE
   
);

CREATE TABLE payment (
    id SERIAL PRIMARY KEY,
    lineId INTEGER NOT NULL,
    type VARCHAR,
    installment TEXT
);