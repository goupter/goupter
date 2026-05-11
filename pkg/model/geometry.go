package model

import (
	"database/sql/driver"
	"fmt"
)

// Geometry represents a MySQL geometry type stored as WKT (Well-Known Text) string
// When querying, use ST_AsText() to convert binary geometry to WKT format
type Geometry string

// Scan implements sql.Scanner interface
func (g *Geometry) Scan(value any) error {
	if value == nil {
		*g = ""
		return nil
	}
	switch v := value.(type) {
	case []byte:
		*g = Geometry(v)
	case string:
		*g = Geometry(v)
	default:
		return fmt.Errorf("cannot scan type %T into Geometry", value)
	}
	return nil
}

// Value implements driver.Valuer interface
// Returns WKT string for insert/update operations
// Note: For insert/update, you may need to use ST_GeomFromText() in raw SQL
func (g Geometry) Value() (driver.Value, error) {
	if g == "" {
		return nil, nil
	}
	return string(g), nil
}

// String returns the WKT string representation
func (g Geometry) String() string {
	return string(g)
}

// IsEmpty returns true if the geometry is empty
func (g Geometry) IsEmpty() bool {
	return g == ""
}

// GeometryColumnsProvider interface for models with geometry columns
// Models implementing this interface will have their geometry columns
// automatically converted using ST_AsText() in queries
type GeometryColumnsProvider interface {
	GeometryColumns() []string
}
