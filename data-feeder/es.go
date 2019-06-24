package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/sirupsen/logrus"
	"gopkg.in/olivere/elastic.v6"
	"os"
)

const (
	cook_index      = "cook"
	cook_index_type = "location"
	mapping         = `
{
	"settings":{
		"number_of_shards": 1,
		"number_of_replicas": 0
	},
	"mappings":{
		"location":{
			"properties":{
				"location":{
					"type":"geo_point"
				}
			}
		}
	}
}`
)

type ESClient struct {
	client *elastic.Client
}

type Record struct {
	Id       string   `json:"-"`
	Location Location `json:"location"`
}

func NewRecord(rec map[string]events.DynamoDBAttributeValue) *Record {
	loc, _ := NewLocation(rec)
	return &Record{
		Id:       rec["cookId"].String(),
		Location: *loc,
	}
}

func (record *Record) json() string {
	rec, err := json.Marshal(record)
	if err != nil {
		logrus.WithError(err).Errorln("unable to marshal record")
		panic(err)
	}
	return string(rec)
}

func NewESClient() (*ESClient, error) {
	es, err := elastic.NewClient(
		elastic.SetSniff(false),
		elastic.SetURL(os.Getenv(es_domain)),
		//elastic.SetErrorLog(log.New(os.Stderr, "ELASTIC ", log.LstdFlags)),
		//elastic.SetInfoLog(log.New(os.Stdout, "", log.LstdFlags)),
		//elastic.SetTraceLog(log.New(os.Stdout, "", 0)),
	)
	if err != nil {
		return nil, err
	}
	return &ESClient{
		client: es,
	}, nil
}

func (es *ESClient) CreateIndex() {
	logrus.Infoln("Creating new Index")
	client := es.client
	exists, err := client.IndexExists(cook_index).Do(context.Background())
	if err != nil {
		// Handle error
		panic(err)
	}
	if !exists {
		// Create a new index.
		createIndex, err := client.CreateIndex(cook_index).BodyString(mapping).Do(context.Background())
		if err != nil {
			// Handle error
			logrus.WithError(err).Errorln("Failed to create index")
			panic(err)
		}
		if !createIndex.Acknowledged {
			// Not acknowledged
			logrus.Errorln("Not acknowledged")
		}
	}
}

func (es *ESClient) DeleteIndex(index string) {
	client := es.client
	exists, err := client.IndexExists(index).Do(context.Background())
	if err != nil {
		// Handle error
		panic(err)
	}
	if exists {
		logrus.Println("Index exists")
	} else {
		// Create a new index.
		createIndex, err := client.CreateIndex(index).Do(context.Background())
		if err != nil {
			// Handle error
			logrus.WithError(err).Errorln("Failed to delete index")
			panic(err)
		}
		if !createIndex.Acknowledged {
			// Not acknowledged
			logrus.Errorln("Not acknowledged")
		}
	}
}

func (es *ESClient) insert(rec *Record) {
	res, err := es.client.Index().
		Index(cook_index).
		Id(rec.Id).
		Type(cook_index_type).
		BodyString(rec.json()).
		Do(context.Background())
	if err != nil {
		logrus.WithError(err).Errorln("Not acknowledged")
		panic(err)
	}
	fmt.Printf("Indexed cook %s to index %s, type %s\n", res.Id, res.Index, res.Type)
}

func (es *ESClient) delete(rec *Record) {
	res, err := es.client.Delete().
		Index(cook_index).
		Id(rec.Id).
		Type(cook_index_type).
		Do(context.Background())
	if err != nil {
		logrus.WithError(err).Errorln("Not acknowledged")
		panic(err)
	}
	fmt.Printf("Deleted cook %s to index %s, type %s\n", res.Id, res.Index, res.Type)
}

func (es *ESClient) updateLocation(rec *Record) {
	scriptParams := map[string]interface{}{
		"latitude":  rec.Location.Lat,
		"longitude": rec.Location.Lon,
	}
	script := elastic.NewScript(`ctx._source.location.lat = params.latitude; ctx._source.location.lon = params.longitude`).Lang("painless").Params(scriptParams)
	res, err := es.client.Update().
		Index(cook_index).
		Type(cook_index_type).
		Id(rec.Id).
		Script(script).
		Do(context.Background())
	if err != nil {
		logrus.WithError(err).Errorln("Update Failed")
		panic(err)
	}
	fmt.Printf("Updated latitude cook %s to index %s, type %s\n", res.Id, res.Index, res.Type)
}

func (es *ESClient) geo(location *Location, distance string) {
	query := elastic.NewGeoDistanceQuery("location").
		Point(location.Lat, location.Lon).
		Distance(distance)
	res, err := es.client.
		Scroll(cook_index).
		Size(100).
		Query(query).
		Do(context.TODO())
	if err != nil {
		logrus.WithError(err).Errorln("failed the geospatial query")
		panic(err)
	}
	sh := res.Hits.Hits
	for _, hit := range sh {
		var dat Record
		if err := json.Unmarshal(*hit.Source, &dat); err != nil {
			panic(err)
		}
		logrus.Printf("%+v \n", dat)
	}
}
