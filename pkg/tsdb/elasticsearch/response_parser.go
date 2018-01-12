package elasticsearch

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/grafana/grafana/pkg/components/null"
	"github.com/grafana/grafana/pkg/tsdb"
)

func joinMaps(left map[string]tsdb.TimeSeriesPoints, right map[string]tsdb.TimeSeriesPoints) map[string]tsdb.TimeSeriesPoints {
	result := map[string]tsdb.TimeSeriesPoints{}
	for key, value := range left {
		result[key] = value
	}

	for key, value := range right {
		if _, ok := result[key]; ok {
			for _, pt := range value {
				result[key] = append(result[key], pt)
			}
		} else {
			result[key] = value
		}
	}

	return result
}

func parseSubQueryResults(parentAggregationKey string, bucketlist BucketList, preferredNames NameMap, resultFilter FilterMap) (map[string]tsdb.TimeSeriesPoints, error) {
	timeSeries := map[string]tsdb.TimeSeriesPoints{}

	for _, bucket := range bucketlist.Buckets {
		rawAggregation, _ := json.Marshal(bucket)

		aggregations := make(map[string]interface{})
		err := json.Unmarshal(rawAggregation, &aggregations)
		if err != nil {
			return timeSeries, err
		}

		metricKey := "elasticsearch"
		var valueRow [2]null.Float
		for key, value := range aggregations {
			switch value.(type) {
			case string:
				if key == "key_as_string" {
					keyf, err := strconv.ParseFloat(value.(string), 64)
					if err == nil {
						valueRow[1] = parseValue(keyf)
					}
				}
			case float64:
				if key == "key" {
					valueRow[1] = parseValue(value.(float64))
				}
				if key == "doc_count" {
					valueRow[0] = parseValue(value.(float64))
				}
			default:
				fmt.Printf("Unknown Type: %v %v\n", key, value)
			}

		}

		timeSeries[metricKey] = append(timeSeries[metricKey], valueRow)
	}

	return timeSeries, nil
}

func parseQueryResult(response []byte, preferredNames NameMap, resultFilter FilterMap) (*tsdb.QueryResult, error) {
	queryRes := tsdb.NewQueryResult()

	esSearchResult := &Response{}
	err := json.Unmarshal(response, esSearchResult)
	if err != nil {
		return nil, err
	}

	timeSeries := map[string]tsdb.TimeSeriesPoints{}
	for aggregationID, buckets := range esSearchResult.Aggregations {
		tSeries, err := parseSubQueryResults(aggregationID, buckets, preferredNames, resultFilter)
		if err != nil {
			return nil, err
		}

		timeSeries = joinMaps(timeSeries, tSeries)
	}

	for id, series := range timeSeries {
		ts := &tsdb.TimeSeries{
			Name:   id,
			Points: series,
		}
		queryRes.Series = append(queryRes.Series, ts)
	}
	return queryRes, nil
}

func parseValue(value float64) null.Float {
	return null.FloatFrom(float64(value))
}
