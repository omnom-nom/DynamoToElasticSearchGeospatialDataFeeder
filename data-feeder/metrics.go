package main

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/sirupsen/logrus"
)

type Metric struct {
	Name  string
	Unit  string
	Value string
}

func newMetric(name string, unit string) *Metric {
	return &Metric{
		Name: name,
		Unit: unit,
	}
}

func (self *Metric) metricDatum(val float64) *cloudwatch.MetricDatum {
	return &cloudwatch.MetricDatum{
		MetricName: aws.String(self.Name),
		Unit:       aws.String(self.Unit),
		Value:      aws.Float64(val),
	}
}

var (
	// add new metric here
	Insert  = newMetric("Insert", "Count")
	Modify  = newMetric("Modify", "Count")
	Delete  = newMetric("Delete", "Count")
	Unknown = newMetric("Unknown", "Count")
)

const (
	namespace = "CookLocation"
)

type MetricEmitter interface {
	Emit(metric *Metric, value float64)
}

type MetricAgent struct {
	CloudWatchClient *cloudwatch.CloudWatch
}

func NewMetricAgent(s *session.Session) *MetricAgent {
	return &MetricAgent{
		CloudWatchClient: cloudwatch.New(s),
	}
}

func (self *MetricAgent) Emit(metric *Metric, value float64) {
	input := &cloudwatch.PutMetricDataInput{
		Namespace:  aws.String(namespace),
		MetricData: []*cloudwatch.MetricDatum{metric.metricDatum(value)},
	}
	_, err := self.CloudWatchClient.PutMetricData(input)
	if err != nil {
		logrus.WithError(err).Errorln("Unable to emit metric")
	} else {
		logrus.Infof("Emitted metric - Name : %s, Value : %v \n", metric.Name, value)
	}
}
