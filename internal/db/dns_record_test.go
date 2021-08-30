package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRecordAddListDelete(t *testing.T) {
	db, err := TestSetup()
	assert.Nil(t, err)

	err = ZoneAdd(db, "example1.com")
	assert.Nil(t, err)
	example1, err := ZoneFind(db, "example1.com")
	assert.Nil(t, err)
	assert.NotNil(t, example1)

	err = RecordAdd(db, &Record{
		Type:   "A",
		Label:  "@",
		Value:  "192.168.2.1",
		TTL:    86400,
		ZoneID: example1.ID,
	})
	assert.Nil(t, err)

	records, err := RecordList(db, example1.ID)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(records))

	err = RecordDelete(db, records[0].ID)
	assert.Nil(t, err)
}
