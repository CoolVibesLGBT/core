package extensions

import (
	"database/sql/driver"
	"fmt"

	"github.com/cridenour/go-postgis"
)

type PostGISPoint struct {
	Lng float64 `json:"lng"`
	Lat float64 `json:"lat"`
}

func (p *PostGISPoint) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	var point postgis.PointS
	switch v := value.(type) {
	case []byte:
		if err := point.Scan(v); err != nil {
			return err
		}
	case string:
		if err := point.Scan([]byte(v)); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported type %T for PostGISPoint", v)
	}

	p.Lng = point.X
	p.Lat = point.Y
	return nil
}

func (p PostGISPoint) Value() (driver.Value, error) {
	return fmt.Sprintf("POINT(%f %f)", p.Lng, p.Lat), nil
}
