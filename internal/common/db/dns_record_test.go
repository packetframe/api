package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRecordAddListDelete(t *testing.T) {
	db, err := TestSetup()
	assert.Nil(t, err)

	// Add user1
	err = UserAdd(db, "user1@example.com", "password1", "example referrer")
	assert.Nil(t, err)

	err = ZoneAdd(db, "example1.com", "user1@example.com")
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

	deleted, err := RecordDelete(db, records[0].ID)
	assert.Nil(t, err)
	assert.True(t, deleted)
}

func TestRecordUpdate(t *testing.T) {
	db, err := TestSetup()
	assert.Nil(t, err)

	// Add user1
	err = UserAdd(db, "user1@example.com", "password1", "example referrer")
	assert.Nil(t, err)

	// Add example1.com
	err = ZoneAdd(db, "example1.com", "user1@example.com")
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
	assert.Equal(t, "192.168.2.1", records[0].Value)

	err = RecordUpdate(db, &Record{Value: "203.0.113.1", ID: records[0].ID})
	assert.Nil(t, err)

	records, err = RecordList(db, example1.ID)
	assert.Nil(t, err)
	assert.Equal(t, "203.0.113.1", records[0].Value)
}

func TestScriptRecordAddListDelete(t *testing.T) {
	db, err := TestSetup()
	assert.Nil(t, err)

	// Add user1
	err = UserAdd(db, "user1@example.com", "password1", "example referrer")
	assert.Nil(t, err)

	err = ZoneAdd(db, "example1.com", "user1@example.com")
	assert.Nil(t, err)
	example1, err := ZoneFind(db, "example1.com")
	assert.Nil(t, err)
	assert.NotNil(t, example1)

	err = RecordAdd(db, &Record{
		Type:   "SCRIPT",
		Label:  "@",
		Value:  "async function handleQuery(query) {}",
		TTL:    86400,
		ZoneID: example1.ID,
	})
	assert.Nil(t, err)

	records, err := RecordList(db, example1.ID)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(records))

	deleted, err := RecordDelete(db, records[0].ID)
	assert.Nil(t, err)
	assert.True(t, deleted)
}
