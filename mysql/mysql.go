// Package mysql fornece uma implementação de um adaptador de conexão MySQL.
package mysql

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql" // Importa o driver MySQL sem atribuir um nome de pacote
)

// MysqlDB é uma estrutura que contém um cliente *sql.DB.
type MysqlDB struct {
	client *sql.DB
}

// Client retorna o cliente *sql.DB armazenado na estrutura MysqlDB.
func (db *MysqlDB) Client() *sql.DB {
	return db.client
}

// NewDB cria uma nova instância do adaptador MySQL e retorna um ponteiro para ela.
// A função recebe uma string de conexão (DSN) como argumento e retorna um erro, se houver algum.
func NewDB(dsn string) (*MysqlDB, error) {
	client, err := connect(dsn)
	if err != nil {
		return nil, err
	}
	return &MysqlDB{
		client: client,
	}, nil
}

// connect é uma função auxiliar que estabelece uma conexão com o banco de dados MySQL.
// Recebe uma string de conexão (DSN) como argumento e retorna um cliente *sql.DB e um erro, se houver algum.
func connect(dsn string) (*sql.DB, error) {
	// Abre a conexão com o banco de dados MySQL usando a string DSN
	client, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	// Verifica a conexão, enviando um ping para o servidor MySQL
	err = client.Ping()
	if err != nil {
		return nil, err
	}

	// Retorna o cliente conectado
	return client, nil
}

// Close fecha a conexão com o banco de dados MySQL.
// Retorna um erro, se houver algum.
func (db *MysqlDB) Close() error {
	// Fecha a conexão com o cliente MySQL
	return db.client.Close()
}
