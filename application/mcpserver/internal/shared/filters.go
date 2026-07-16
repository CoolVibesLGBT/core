package shared

import (
	"context"
	"core/constants"
	"core/types"
	"fmt"
	"strconv"

	"github.com/google/uuid"
)

func BuildFilter(ctx context.Context, arguments map[string]any) (types.Filter, error) {
	filter := types.Filter{
		Context: ctx,
		Limit:   constants.DEFAULT_LIMIT,
	}

	if len(arguments) == 0 {
		return filter, nil
	}

	if value, ok := LookupString(arguments, "user_id", "userId"); ok {
		if err := applyMixedID(value, &filter.UserUUID, &filter.UserID, "user_id"); err != nil {
			return filter, err
		}
	}

	if value, ok := LookupString(arguments, "post_id", "postId"); ok {
		if err := applyMixedID(value, &filter.PostUUID, &filter.PostID, "post_id"); err != nil {
			return filter, err
		}
	}

	if limit, ok := LookupInt64(arguments, "limit"); ok {
		if limit <= 0 {
			return filter, fmt.Errorf("invalid limit")
		}
		if limit > constants.MAXIMUM_LIMIT {
			limit = constants.MAXIMUM_LIMIT
		}
		filter.Limit = int(limit)
	}

	if cursor, ok := LookupInt64(arguments, "cursor"); ok {
		filter.Cursor = &cursor
	}

	if search, ok := LookupString(arguments, "search"); ok {
		filter.Search = &search
	}

	if category, ok := LookupString(arguments, "category"); ok {
		filter.Category = &category
	}

	if name, ok := LookupString(arguments, "name"); ok {
		filter.Name = &name
	}

	if city, ok := LookupString(arguments, "city"); ok {
		filter.City = &city
	}

	if country, ok := LookupString(arguments, "country"); ok {
		filter.Country = &country
	}

	if latitude, ok := LookupFloat64(arguments, "latitude"); ok {
		filter.Latitude = &latitude
	}

	if longitude, ok := LookupFloat64(arguments, "longitude"); ok {
		filter.Longitude = &longitude
	}

	if distance, ok := LookupFloat64(arguments, "distance"); ok {
		filter.Distance = &distance
	}

	return filter, nil
}

func applyMixedID(value string, uuidTarget *uuid.UUID, intTarget *int64, fieldName string) error {
	if parsedUUID, err := uuid.Parse(value); err == nil {
		*uuidTarget = parsedUUID
		return nil
	}

	parsedID, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid %s: %s", fieldName, value)
	}

	*intTarget = parsedID
	return nil
}
