package db

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/packetframe/api/internal/api/rpc"
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
	if record.Type == "DNSSCRIPT" {
		if err := DNSScriptCompile(record.Value, record.Label); err != nil {
			return fmt.Errorf("dns script compile: %s", err)
		}
	}

	if err := ZoneSetSerial(db, record.ZoneID); err != nil {
		return err
	}

	tx := db.Create(record)
	if tx.Error == nil {
		if err := rpc.Call("update_zone", map[string]string{"id": record.ZoneID}); err != nil {
			log.Warnf("RPC: %v", err)
		}
	}
	return tx.Error
}

// RecordList returns a list of DNS records for a zone
func RecordList(db *gorm.DB, zone string) ([]Record, error) {
	var records []Record
	err := db.Order("created_at").Where("zone_id = ?", zone).Find(&records).Error
	return records, err
}

// RecordListNoDNSScript returns a list of DNS records for a zone excluding DNSSCRIPT records
func RecordListNoDNSScript(db *gorm.DB, zone string) ([]Record, error) {
	var records []Record
	err := db.Order("created_at").Where("zone_id = ? AND type IS NOT 'DNSSCRIPT'", zone).Find(&records).Error
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
	if err := ZoneSetSerial(db, r.ZoneID); err != nil {
		return false, err
	}

	req := db.Where("id = ?", recordID).Delete(&Record{})
	if req.Error == nil {
		if err := rpc.Call("update_zone", map[string]string{"id": r.ZoneID}); err != nil {
			log.Warnf("RPC: %v", err)
		}
	}

	return req.RowsAffected > 0, req.Error
}

// RecordUpdate updates a DNS record
func RecordUpdate(db *gorm.DB, updates *Record) error {
	var currentRecord Record
	if err := db.Find(&currentRecord, "id = ?", updates.ID).Error; err != nil {
		return err
	}

	if err := ZoneSetSerial(db, currentRecord.ZoneID); err != nil {
		return err
	}

	tx := db.Model(&currentRecord).Updates(updates)
	if tx.Error == nil {
		if err := rpc.Call("update_zone", map[string]string{"id": currentRecord.ZoneID}); err != nil {
			log.Warnf("RPC: %v", err)
		}
	}

	return tx.Error
}
