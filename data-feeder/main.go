package main

import (
	"context"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/sirupsen/logrus"
)

const (
	es_domain = "ELASTIC_SEARCH_DOMAIN"
	insert    = "INSERT"
	modify    = "MODIFY"
	remove    = "REMOVE"
)

var (
	es          *ESClient
	metricAgent *MetricAgent
)

func main() {
	lambda.Start(handleRequest)
}

func handleRequest(ctx context.Context, e events.DynamoDBEvent) {
	s, err := session.NewSession(aws.NewConfig())
	if err != nil {
		logrus.WithError(err)
	}
	metricAgent = NewMetricAgent(s)
	defer func() {
		if r := recover(); r != nil {
			// emit failure metric
			emitMetric(e, 0)
		}
	}()

	es, err = NewESClient()
	if err != nil {
		logrus.WithError(err).Errorln("Unable to connect to elastic search cluster")
		panic(err)
	}
	for _, record := range e.Records {
		switch event_type := record.EventName; event_type {
		case insert:
			logrus.Println("Inserting a record")
			insertRecord(record)
		case modify:
			logrus.Println("Modifying a record")
			modifyRecord(record)
		case remove:
			logrus.Println("Removing a record")
			removeRecord(record)
		default:
			logrus.Errorln("Event type not recognized: %s ", event_type)
			panic("unknown event")
		}
	}
	// emit success metric
	emitMetric(e, 1)
}

func emitMetric(e events.DynamoDBEvent, val float64) {
	for _, record := range e.Records {
		switch event_type := record.EventName; event_type {
		case insert:
			metricAgent.Emit(Insert, val)
		case modify:
			metricAgent.Emit(Modify, val)
		case remove:
			metricAgent.Emit(Delete, val)
		default:
			metricAgent.Emit(Unknown, val)
		}
	}
}

func insertRecord(record events.DynamoDBEventRecord) {
	rec := NewRecord(record.Change.NewImage)
	es.insert(rec)
}

func modifyRecord(record events.DynamoDBEventRecord) {
	updated, err := locationUpdated(record.Change)
	if err != nil {
		logrus.WithError(err)
		panic(err)
	}
	if updated {
		rec := NewRecord(record.Change.NewImage)
		es.updateLocation(rec)
	}
}

func removeRecord(record events.DynamoDBEventRecord) {
	rec := NewRecord(record.Change.OldImage)
	es.delete(rec)
}

// this method should be updated only when there is a MODIFY event
func locationUpdated(change events.DynamoDBStreamRecord) (bool, error) {
	newLocation, err := NewLocation(change.NewImage)
	if err != nil {
		return false, err
	}
	oldLocation, err := NewLocation(change.OldImage)
	if err != nil {
		return false, err
	}
	return !newLocation.equals(oldLocation), nil
}
