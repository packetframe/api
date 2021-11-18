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
	return db.Create(record).Error
}

// RecordList returns a list of DNS records for a zone
func RecordList(db *gorm.DB, zone string) ([]Record, error) {
	var records []Record
	err := db.Order("created_at").Where("zone_id = ?", zone).Find(&records).Error
	return records, err
}

// RecordDelete deletes a DNS record from a zone
func RecordDelete(db *gorm.DB, record string) (bool, error) {
	req := db.Where("id = ?", record).Delete(&Record{})
	return req.RowsAffected > 0, req.Error
}

// RecordUpdate updates a DNS record
func RecordUpdate(db *gorm.DB, updates *Record) error {
	var currentRecord Record
	if err := db.Find(&currentRecord, "id = ?", updates.ID).Error; err != nil {
		return err
	}

	return db.Model(&currentRecord).Updates(updates).Error
}
