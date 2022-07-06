package db

import (
	"time"

	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Credential struct {
	FQDN      string    `gorm:"primary_key" json:"id"`
	Cert      string    `json:"cert"`
	Key       string    `json:"key"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
}

// CredentialAddOrUpdate adds a new credential to the database
func CredentialAddOrUpdate(db *gorm.DB, fqdn, cert, key string) error {
	log.Debugf("Attempting to add/update credential for %s", fqdn)

	// Attempt to grab the credential
	var cred Credential
	if err := db.First(&cred, "fqdn = ?", fqdn).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			return err
		}
	}
	cred.FQDN = fqdn
	cred.Cert = cert
	cred.Key = key

	return db.Save(&cred).Error
}

// CredentialList gets a list of credentials
func CredentialList(db *gorm.DB) ([]Credential, error) {
	var creds []Credential
	res := db.Find(&creds)
	if res.Error != nil {
		return nil, res.Error
	}

	return creds, nil
}
