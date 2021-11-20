package orchestrator

import (
	"fmt"
	"github.com/packetframe/api/internal/db"
)

// ZoneFile creates a DNS zone file from the database
func ZoneFile(zoneID string) (string, error) {
	zone, err := db.ZoneFindByID(nil, zoneID)
	if err != nil {
		return "", err
	}

	records, err := db.RecordList(nil, zoneID)
	if err != nil {
		return "", err
	}

	zoneFile := fmt.Sprintf(`// Packetframe zone file
@ IN SOA ns1.packetframe.com. info.packetframe.com. (
   %d        ; Serial
   7200      ; Refresh, number of seconds after which secondary NSes should query the main to detect zone changes
   3600      ; Retry, number of seconds after which secondary NSes should retry serial query from the main if it doesn't respond
   1209600   ; Expire, number of seconds after which secondary NSes should stop answering if main doesn't respond
   300 )     ; Negative Cache TTL 
`, zone.Serial)

	for _, record := range records {
		zoneFile += fmt.Sprintf("%s %d IN %s %s", record.Label, record.TTL, record.Type, record.Value)
	}

	return zoneFile, nil
}
