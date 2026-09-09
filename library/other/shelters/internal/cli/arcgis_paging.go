// Copyright 2026 Abe Diaz (@abe238) and contributors. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

// arcGISPageCap prevents a broken service from keeping a side-feed request in
// an unbounded paging loop. Even the smallest verified layer limit reaches
// 100,000 records before this guard fires.
const arcGISPageCap = 100

type arcGISQuery struct {
	URL                string
	Referer            string
	OIDField           string
	PageSize           int
	SupportsPagination bool
}

// getArcGISPage is the HTTP seam for side-feed paging tests.
var getArcGISPage = httpGet

// fetchArcGISPages follows ArcGIS transfer-limit markers and returns one
// combined feature collection so existing parsers keep their whole-feed
// validation and duplicate detection. OID ordering makes resultOffset stable
// while the query is paged.
func fetchArcGISPages(ctx context.Context, query arcGISQuery, base url.Values) ([]byte, error) {
	features := make([]json.RawMessage, 0)
	offset := 0
	for page := 0; page < arcGISPageCap; page++ {
		params := make(url.Values, len(base)+3)
		for key, values := range base {
			params[key] = append([]string(nil), values...)
		}
		params.Set("resultOffset", strconv.Itoa(offset))
		params.Set("orderByFields", query.OIDField+" ASC")
		if query.PageSize > 0 {
			params.Set("resultRecordCount", strconv.Itoa(query.PageSize))
		}

		body, err := getArcGISPage(ctx, query.URL+"?"+params.Encode(), query.Referer)
		if err != nil {
			return nil, fmt.Errorf("fetching ArcGIS page at offset %d: %w", offset, err)
		}
		var response struct {
			Error *struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
			Features              []json.RawMessage `json:"features"`
			ExceededTransferLimit bool              `json:"exceededTransferLimit"`
		}
		if err := json.Unmarshal(body, &response); err != nil {
			return nil, fmt.Errorf("parsing ArcGIS page at offset %d: %w", offset, err)
		}
		if response.Error != nil {
			return nil, fmt.Errorf("ArcGIS service error %d at offset %d: %s", response.Error.Code, offset, response.Error.Message)
		}
		if response.Features == nil {
			return nil, fmt.Errorf("unrecognized ArcGIS response at offset %d: no 'features' array and no 'error'", offset)
		}
		features = append(features, response.Features...)
		if !response.ExceededTransferLimit {
			return json.Marshal(struct {
				Features []json.RawMessage `json:"features"`
			}{Features: features})
		}
		if !query.SupportsPagination {
			return nil, fmt.Errorf("ArcGIS response was truncated at %d record(s) and the layer does not support pagination", len(features))
		}
		if len(response.Features) == 0 {
			return nil, fmt.Errorf("ArcGIS pagination made no progress at offset %d", offset)
		}
		offset += len(response.Features)
	}
	return nil, fmt.Errorf("ArcGIS pagination exceeded the %d-page cap", arcGISPageCap)
}
