// Package mongo fornece uma implementação de um adaptador de conexão MongoDB.
package mongo

import (
	"context"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"time"
)

// MongoDB é uma estrutura que contém um cliente *mongo.Client.
type MongoDB struct {
	client *mongo.Client
}

// NewDB cria uma nova instância do adaptador MongoDB e retorna um ponteiro para ela.
// A função recebe uma string de conexão (URI) como argumento e retorna um erro, se houver algum.
func NewDB(uri string) (*MongoDB, error) {
	client, err := connect(uri)
	if err != nil {
		return nil, err
	}
	return &MongoDB{
		client: client,
	}, nil
}

// connect é uma função auxiliar que estabelece uma conexão com o banco de dados MongoDB.
// Recebe uma string de conexão (URI) como argumento e retorna um cliente *mongo.Client e um erro, se houver algum.
func connect(uri string) (*mongo.Client, error) {
	// Cria um contexto com tempo limite de 10 segundos
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Conecta ao MongoDB usando o contexto e a string de conexão (URI)
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}

	// Verifica a conexão, enviando um ping para o cluster primário
	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		return nil, err
	}

	// Retorna o cliente conectado
	return client, nil
}

// Close fecha a conexão com o banco de dados MongoDB.
// Retorna um erro, se houver algum.
func (db *MongoDB) Close() error {
	// Cria um contexto com tempo limite de 10 segundos
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Desconecta o cliente do MongoDB usando o contexto
	return db.client.Disconnect(ctx)
}

// Collection retorna um ponteiro para uma coleção do banco de dados MongoDB.
// Recebe o nome do banco de dados e o nome da coleção como argumentos.
func (db *MongoDB) Collection(database, collection string) *mongo.Collection {
	// Retorna a coleção do banco de dados especificado
	return db.client.Database(database).Collection(collection)
}
