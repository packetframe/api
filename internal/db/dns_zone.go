package db

import (
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"
	"github.com/miekg/dns"
	"gorm.io/gorm"
)

var (
	ErrUserExistingZoneMember = errors.New("user is already a member of this zone")
	ErrUserNotFound           = errors.New("user not found")
)

// Zone stores a DNS zone
type Zone struct {
	ID        string         `gorm:"primaryKey,type:uuid;default:uuid_generate_v4()" json:"id"`
	Zone      string         `gorm:"uniqueIndex" json:"zone"`
	Serial    uint64         `json:"-"`
	DNSSEC    DNSSECKey      `gorm:"embedded" json:"-"`
	Users     pq.StringArray `gorm:"type:text[]" json:"users"`
	CreatedAt time.Time      `json:"-"`
	UpdatedAt time.Time      `json:"-"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// DNSSECKey stores a DNSSEC signing key
type DNSSECKey struct {
	Base           string // Base key filename prefix
	Key            string // DNSKEY
	Private        string // Private key
	DSKeyTag       int    // DS key tag
	DSAlgo         int    // DS algorithm
	DSDigestType   int    // DS digest type
	DSDigest       string // DS digest
	DSRecordString string // Full DS record in zone file format
}

// NewKey generates a new DNSSEC signing key for a zone
func NewKey(zone string) (*DNSSECKey, error) {
	key := &dns.DNSKEY{
		Hdr: dns.RR_Header{
			Name:   dns.Fqdn(zone),
			Class:  dns.ClassINET,
			Ttl:    3600,
			Rrtype: dns.TypeDNSKEY,
		},
		Algorithm: dns.ECDSAP256SHA256, Flags: 257, Protocol: 3,
	}

	private, err := key.Generate(256)
	if err != nil {
		return nil, err
	}

	ds := key.ToDS(dns.SHA256)

	return &DNSSECKey{
		Base:           fmt.Sprintf("K%s+%03d+%05d", key.Header().Name, key.Algorithm, key.KeyTag()),
		Key:            key.String(),
		Private:        key.PrivateKeyString(private),
		DSKeyTag:       int(ds.KeyTag),
		DSAlgo:         int(ds.Algorithm),
		DSDigestType:   int(ds.DigestType),
		DSDigest:       ds.Digest,
		DSRecordString: ds.String(),
	}, nil // nil error
}

// ZoneSetSerial sets a zone's SOA serial
func ZoneSetSerial(db *gorm.DB, uuid string) error {
	var zone Zone
	if err := db.First(&zone, "id = ?", uuid).Error; err != nil {
		return err
	}
	zone.Serial = uint64(time.Now().Unix())
	return db.Save(&zone).Error
}

// ZoneAdd adds a DNS zone by zone name and user email
func ZoneAdd(db *gorm.DB, zone string, user string) error {
	u, err := UserFindByEmail(db, user)
	if err != nil {
		return err
	}
	if u == nil {
		return ErrUserNotFound
	}

	zone = dns.Fqdn(zone)
	dnssecKey, err := NewKey(zone)
	if err != nil {
		return err
	}
	return db.Create(&Zone{
		Zone:   zone,
		Serial: uint64(time.Now().Unix()),
		DNSSEC: *dnssecKey,
		Users:  []string{u.ID},
	}).Error
}

// ZoneList gets a list of all zones
func ZoneList(db *gorm.DB) ([]Zone, error) {
	var zones []Zone
	res := db.Find(&zones)
	if res.Error != nil {
		return nil, res.Error
	}
	return zones, nil
}

// ZoneFind finds a user by zone and returns nil if no zone exists
func ZoneFind(db *gorm.DB, zone string) (*Zone, error) {
	zone = dns.Fqdn(zone)
	var z Zone
	res := db.First(&z, "zone = ?", zone)
	if errors.Is(res.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if res.Error != nil {
		return nil, res.Error
	}

	return &z, nil
}

// ZoneDelete deletes a DNS zone
func ZoneDelete(db *gorm.DB, zone string) error {
	return db.Delete(&Zone{}, "id = ?", zone).Error
}

// ZoneRotateDNSSECKey rotates a zone's DNSSEC key
func ZoneRotateDNSSECKey(db *gorm.DB, uuid string) error {
	var zone Zone
	if err := db.First(&zone, "id = ?", uuid).Error; err != nil {
		return err
	}
	dnssecKey, err := NewKey(zone.Zone)
	if err != nil {
		return err
	}
	zone.DNSSEC = *dnssecKey
	return db.Save(&zone).Error
}

// ZoneUserAdd adds a user to a zone
func ZoneUserAdd(db *gorm.DB, zone string, user string) error {
	var z Zone
	if err := db.First(&z, "zone = ?", dns.Fqdn(zone)).Error; err != nil {
		return err
	}

	// Make sure user exists before adding it to the zone
	u, err := UserFindByEmail(db, user)
	if err != nil {
		return err
	}
	if u == nil {
		return ErrUserNotFound
	}

	// Check if user is already added to this zone
	for _, existingUserID := range z.Users {
		if existingUserID == u.ID {
			return ErrUserExistingZoneMember
		}
	}

	z.Users = append(z.Users, u.ID)
	return db.Save(&z).Error
}

// ZoneUserDelete deletes a user from a zone
func ZoneUserDelete(db *gorm.DB, zone string, user string) error {
	var z Zone
	if err := db.First(&z, "id = ?", zone).Error; err != nil {
		return err
	}

	for i, u := range z.Users {
		if u == user {
			z.Users = append(z.Users[:i], z.Users[i+1:]...)
		}
	}

	return db.Save(&z).Error
}

// ZoneGet gets a zone by UUID
func ZoneGet(db *gorm.DB, zone string) (*Zone, error) {
	var z Zone
	if err := db.First(&z, "id = ?", zone).Error; err != nil {
		return nil, err
	}
	return &z, nil
}

// ZoneUserGetZones gets all zones a user is a member of
func ZoneUserGetZones(db *gorm.DB, user string) ([]Zone, error) {
	var zones []Zone
	res := db.Model(&Zone{}).Where("? = ANY(users)", user).Find(&zones)
	if res.Error != nil {
		return nil, res.Error
	}
	return zones, nil
}

// ZoneUserAuthorized checks if a user is authorized for a zone
func ZoneUserAuthorized(db *gorm.DB, zone string, user string) (bool, error) {
	var z Zone
	res := db.Model(&Zone{}).Where("id = ? AND ? = ANY(users)", zone, user).Find(&z)
	if errors.Is(res.Error, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if res.Error != nil {
		return false, res.Error
	}

	return z.Zone != "", nil
}
