package main

import (
	"github.com/aws/aws-lambda-go/events"
	"github.com/sirupsen/logrus"
)

const (
	lat             = "lat"
	long            = "lon"
	EPSILON float64 = 0.00000001
)

type Location struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

func NewLocation(attributes map[string]events.DynamoDBAttributeValue) (*Location, error) {
	latitude, err := attributes[lat].Float()
	if err != nil {
		logrus.WithError(err).Errorln("Unable to convert the location latitude")
		return nil, err
	}
	longitude, err := attributes[long].Float()
	if err != nil {
		logrus.WithError(err).Errorln("Unable to convert the location latitude")
		return nil, err
	}
	return &Location{
		Lat: latitude,
		Lon: longitude,
	}, nil
}

func LocationFromLatLong(lat, lon float64) *Location {
	return &Location{
		Lat: lat,
		Lon: lon,
	}
}

// 0 if same
// 1 if only latitude different
// 2 if only longitude different
// 3 if both updated
func (l1 *Location) equals(l2 *Location) bool {
	return floatEquals(l1.Lat, l2.Lat) && floatEquals(l1.Lon, l2.Lon)
}

func floatEquals(a, b float64) bool {
	if (a-b) < EPSILON && (b-a) < EPSILON {
		return true
	}
	return false
}
