package db

import (
	"time"

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
	var cred Credential
	res := db.FirstOrCreate(&cred, "fqdn = ?", fqdn)
	if res.Error != nil {
		return res.Error
	}

	cred.Cert = cert
	cred.Key = key
	res = db.Save(&cred)
	if res.Error != nil {
		return res.Error
	}

	return nil
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
