package db

import (
	"time"

	"gorm.io/gorm"
)

// Record stores a DNS record
type Record struct {
	ID     string `gorm:"primaryKey,type:uuid;default:uuid_generate_v4()" json:"id"`
	Type   string `json:"type" validate:"required,dns-rrtype"`
	Label  string `json:"label" validate:"required"`
	Value  string `json:"value"`
	TTL    uint32 `json:"ttl" validate:"gte=300,lte=2147483647"`
	Proxy  bool   `json:"proxy"`
	ZoneID string `json:"zone"`

	Zone      Zone      `json:"-" validate:"-"` // Zone is populated by the database so will be zero value at record creation time
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
}

// RecordAdd adds a new record to a zone
func RecordAdd(db *gorm.DB, record *Record) error {
	if err := ZoneIncrementSerial(db, record.ZoneID); err != nil {
		return err
	}

	return db.Create(record).Error
}

// RecordList returns a list of DNS records for a zone
func RecordList(db *gorm.DB, zone string) ([]Record, error) {
	var records []Record
	err := db.Order("created_at").Where("zone_id = ?", zone).Find(&records).Error
	return records, err
}

// RecordListAll returns a list of all DNS records
func RecordListAll(db *gorm.DB) ([]Record, error) {
	var records []Record
	err := db.Order("created_at").Find(&records).Error
	return records, err
}

// RecordDelete deletes a DNS record from a zone
func RecordDelete(db *gorm.DB, recordID string) (bool, error) {
	// Find the record to get the zone ID
	var r Record
	if err := db.Find(&r, "id = ?", recordID).Error; err != nil {
		return false, err
	}

	// Bump the zone serial
	if err := ZoneIncrementSerial(db, r.ZoneID); err != nil {
		return false, err
	}

	req := db.Where("id = ?", recordID).Delete(&Record{})
	return req.RowsAffected > 0, req.Error
}

// RecordUpdate updates a DNS record
func RecordUpdate(db *gorm.DB, updates *Record) error {
	var currentRecord Record
	if err := db.Find(&currentRecord, "id = ?", updates.ID).Error; err != nil {
		return err
	}

	if err := ZoneIncrementSerial(db, currentRecord.ZoneID); err != nil {
		return err
	}

	return db.Model(&currentRecord).Updates(updates).Error
}
