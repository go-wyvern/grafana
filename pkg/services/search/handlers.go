package search

import (
	"sort"

	"github.com/go-wyvern/grafana/pkg/bus"
	m "github.com/go-wyvern/grafana/pkg/models"
)

func Init() {
	bus.AddHandler("search", searchHandler)
}

func searchHandler(query *Query) error {
	dashQuery := FindPersistedDashboardsQuery{
		Title:        query.Title,
		SignedInUser: query.SignedInUser,
		IsStarred:    query.IsStarred,
		DashboardIds: query.DashboardIds,
		Type:         query.Type,
		FolderIds:    query.FolderIds,
		Tags:         query.Tags,
		Limit:        query.Limit,
	}

	if err := bus.Dispatch(&dashQuery); err != nil {
		return err
	}

	hits := make(HitList, 0)
	hits = append(hits, dashQuery.Result...)

	// sort main result array
	sort.Sort(hits)

	if len(hits) > query.Limit {
		hits = hits[0:query.Limit]
	}

	// sort tags
	for _, hit := range hits {
		sort.Strings(hit.Tags)
	}

	// add isStarred info
	if err := setIsStarredFlagOnSearchResults(query.SignedInUser.UserId, hits); err != nil {
		return err
	}

	query.Result = hits
	return nil
}

func setIsStarredFlagOnSearchResults(userId int64, hits []*Hit) error {
	query := m.GetUserStarsQuery{UserId: userId}
	if err := bus.Dispatch(&query); err != nil {
		return err
	}

	for _, dash := range hits {
		if _, exists := query.Result[dash.Id]; exists {
			dash.IsStarred = true
		}
	}

	return nil
}
