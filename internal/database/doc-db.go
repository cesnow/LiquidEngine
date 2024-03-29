package database

import (
	"context"
	"errors"
	"fmt"
	"github.com/cesnow/liquid-engine/logger"
	"github.com/cesnow/liquid-engine/settings"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"

	"net/url"
	"strings"
	"time"
)

const (
	authenticationStringTemplate = "%s:%s@"
	connectionStringTemplate     = "%s://%s%s/%s?"
)

type DocDB struct {
	IDatabase
	client *mongo.Client
	config *settings.DocDbConf
}

func ConnectWithDocDB(config *settings.DocDbConf) (*DocDB, error) {

	docDb := &DocDB{
		config: config,
	}

	authenticationURI := ""
	if config.Username != "" {
		authenticationURI = fmt.Sprintf(
			authenticationStringTemplate,
			config.Username,
			config.Password,
		)
	}

	connectionURI := fmt.Sprintf(
		connectionStringTemplate,
		config.Protocol,
		authenticationURI,
		config.Host,
		config.DefaultDb,
	)

	connectUri, _ := url.Parse(connectionURI)
	connectQuery, _ := url.ParseQuery(connectUri.RawQuery)

	if config.ReplicaSet != "" {
		connectQuery.Add("replicaSet", config.ReplicaSet)
	}

	if config.ReadPreference != "" {
		connectQuery.Add("readpreference", config.ReadPreference)
	}

	connectUri.RawQuery = connectQuery.Encode()

	printConnectUri := connectUri.String()
	findIndexAt := strings.Index(connectUri.String(), "@")
	if findIndexAt > -1 && config.Username != "" {
		prefixIndex := len(config.Protocol) + 3 + len(config.Username)
		connectUriStr := connectUri.String()
		printConnectUri = fmt.Sprintf("%s:*****%s", connectUriStr[:prefixIndex], connectUriStr[findIndexAt:])
	}
	logger.SysLog.Infof("[DocumentDB] Try to connect document db `%s`", printConnectUri)

	clientOptions := options.Client().ApplyURI(connectUri.String())
	client, err := mongo.NewClient(clientOptions)

	if err != nil {
		return nil, errors.New(fmt.Sprintf("Failed to Create New Client, %s", err))
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.ConnectTimeoutMs)*time.Millisecond)
	defer cancel()

	err = client.Connect(ctx)
	if err != nil {
		logger.SysLog.Errorf("Failed to connect to cluster: %v", err)
	}

	// Force a connection to verify our connection string
	err = client.Ping(ctx, readpref.SecondaryPreferred())
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Failed to ping cluster: %s", err))
	}

	logger.SysLog.Info("[DocumentDB] Connected to DocumentDB!")

	docDb.client = client

	return docDb, nil
}

func (db *DocDB) PopulateIndex(database, collection, key string, sort int32, unique bool) {
	c := db.client.Database(database).Collection(collection)
	opts := options.CreateIndexes().SetMaxTime(3 * time.Second)
	index := db.yieldIndexModel(
		[]string{key}, []int32{sort}, unique, nil,
	)
	_, err := c.Indexes().CreateOne(context.Background(), index, opts)
	if err != nil {
		logger.SysLog.Errorf("[DocumentDb] Ensure Index Failed, %s", err)
	}
}

func (db *DocDB) PopulateTTLIndex(database, collection, key string, sort int32, unique bool, ttl int32) {
	c := db.client.Database(database).Collection(collection)
	opts := options.CreateIndexes().SetMaxTime(3 * time.Second)
	index := db.yieldIndexModel(
		[]string{key}, []int32{sort}, unique,
		options.Index().SetExpireAfterSeconds(ttl),
	)
	_, err := c.Indexes().CreateOne(context.Background(), index, opts)
	if err != nil {
		logger.SysLog.Errorf("[DocumentDb] Ensure TTL Index Failed, %s", err)
	}
}

func (db *DocDB) PopulateMultiIndex(database, collection string, keys []string, sorts []int32, unique bool) {
	if len(keys) != len(sorts) {
		logger.SysLog.Warnf("[DocumentDb] Ensure Indexes Failed, %s", "Please provide some item length of keys/sorts")
		return
	}
	c := db.client.Database(database).Collection(collection)
	opts := options.CreateIndexes().SetMaxTime(3 * time.Second)
	index := db.yieldIndexModel(keys, sorts, unique, nil)
	_, err := c.Indexes().CreateOne(context.Background(), index, opts)
	if err != nil {
		logger.SysLog.Errorf("[DocumentDb] Ensure TTL Index Failed, %s", err)
	}
}

func (db *DocDB) yieldIndexModel(keys []string, sorts []int32, unique bool, indexOpt *options.IndexOptions) mongo.IndexModel {
	SetKeysDoc := bson.D{}
	for index, _ := range keys {
		key := keys[index]
		sort := sorts[index]
		SetKeysDoc = append(SetKeysDoc, bson.E{Key: key, Value: sort})
	}
	if indexOpt == nil {
		indexOpt = options.Index()
	}
	indexOpt.SetUnique(unique)
	index := mongo.IndexModel{
		Keys:    SetKeysDoc,
		Options: indexOpt,
	}
	return index
}

func (db *DocDB) ListIndexes(database, collection string) {
	c := db.client.Database(database).Collection(collection)
	duration := 10 * time.Second
	batchSize := int32(10)
	cur, err := c.Indexes().List(context.Background(), &options.ListIndexesOptions{&batchSize, &duration})
	if err != nil {
		logger.SysLog.Fatalf("[DocumentDB] Something went wrong listing %v", err)
	}
	for cur.Next(context.Background()) {
		index := bson.D{}
		_ = cur.Decode(&index)
	}
}

func (db *DocDB) GetClient() *mongo.Client {
	return db.client
}

func (db *DocDB) Database(dbname string) *mongo.Database {
	return db.client.Database(dbname)
}
