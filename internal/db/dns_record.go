package db

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Record stores a DNS record
type Record struct {
	gorm.Model
	ID    uuid.UUID `gorm:"primaryKey,type:uuid;default:uuid_generate_v4()"`
	Type  string
	Label string
	Value string
	TTL   uint32
	Proxy bool

	ZoneID uuid.UUID
	Zone   Zone
}

// RecordAdd adds a new record to a zone
func RecordAdd(db *gorm.DB, record *Record) error {
	return db.Create(record).Error
}

// RecordList returns a list of DNS records for a zone
func RecordList(db *gorm.DB, zone uuid.UUID) ([]Record, error) {
	var records []Record
	err := db.Where("zone_id = ?", zone).Find(&records).Error
	return records, err
}

// RecordDelete deletes a DNS record from a zone
func RecordDelete(db *gorm.DB, record uuid.UUID) error {
	return db.Where("id = ?", record).Delete(&Record{}).Error
}
