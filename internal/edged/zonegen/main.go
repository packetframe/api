package zonegen

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/packetframe/api/internal/common/db"
)

// cache of zone FQDN to serial
var cache = make(map[string]uint64)

// writeZoneToFile writes a zone file to disk
func writeZoneToFile(database *gorm.DB, zoneID, zonesDirectory string) error {
	zone, err := db.ZoneFindByID(database, zoneID)
	if err != nil {
		return err
	}

	records, err := db.RecordList(database, zoneID)
	if err != nil {
		return err
	}

	// Serial
	// Refresh, number of seconds after which secondary NSes should query the main to detect zone changes
	// Retry, number of seconds after which secondary NSes should retry serial query from the main if it doesn't respond
	// Expire, number of seconds after which secondary NSes should stop answering if main doesn't respond
	// Negative Cache TTL
	zoneFile := fmt.Sprintf(`@ IN SOA ns1.packetframe.com. info.packetframe.com. %d 7200 3600 1209600 300
@ 86400 IN NS ns1.packetframe.com.
@ 86400 IN NS ns2.packetframe.com.
`, zone.Serial)

	for _, record := range records {
		if record.Type == "SCRIPT" {
			zoneFile += fmt.Sprintf("%s 3600 IN NS script-ns.packetframe.com.\n", record.Label)
		} else if record.Proxy {
			zoneFile += fmt.Sprintf("%s 3600 IN A 66.248.234.7\n", record.Label)
			zoneFile += fmt.Sprintf("%s 3600 IN AAAA 2602:809:3004::7\n", record.Label)
		} else {
			zoneFile += fmt.Sprintf("%s %d IN %s %s\n", record.Label, record.TTL, record.Type, record.Value)
		}
	}

	// Write the zone file to disk
	return os.WriteFile(path.Join(zonesDirectory, "db."+strings.TrimSuffix(zone.Zone, ".")), []byte(zoneFile), 0644)
}

// writeZoneManifest writes the zone configuration file for knot
func writeZoneManifest(database *gorm.DB, knotZonesFile string) error {
	zones, err := db.ZoneList(database)
	if err != nil {
		return err
	}

	manifestContent := fmt.Sprintf("# knot.zones.conf generated at %v\n", time.Now().UTC())
	for _, zone := range zones {
		manifestContent += fmt.Sprintf(`zone:
  - domain: %s
    template: default
`, strings.TrimSuffix(zone.Zone, "."))
	}

	// Write the zone manifest to disk
	return os.WriteFile(knotZonesFile, []byte(manifestContent), 0644)
}

// Update writes all zone files to disk and removes unreferenced ones
func Update(zonesDirectory, knotZonesFile string, database *gorm.DB) error {
	zones, err := db.ZoneList(database)
	if err != nil {
		return err
	}

	// Is a knot reload required?
	reloadRequired := false

	for _, zone := range zones {
		// If zone not in cache or cached serial older than current serial...
		if _, inCache := cache[zone.Zone]; !inCache || cache[zone.Zone] < zone.Serial {
			reloadRequired = true
			cache[zone.Zone] = zone.Serial
			if err := writeZoneToFile(database, zone.ID, zonesDirectory); err != nil {
				log.Warnf("writing zone file (%s): %s", zone.Zone, err)
			}
		}
	}

	// Remove zones that aren't referenced in the cache
	zoneFiles, err := os.ReadDir(zonesDirectory)
	if err != nil {
		return err
	}
	for _, f := range zoneFiles {
		found := false
		for _, zone := range zones {
			if "db."+strings.TrimSuffix(zone.Zone, ".") == f.Name() {
				found = true
				break
			}
		}

		if !found {
			log.Debugf("%s not found, removing", f.Name())
			if err := os.Remove(path.Join(zonesDirectory, f.Name())); err != nil {
				log.Warnf("removing referenced zone file %s: %s", f.Name(), err)
			}
			reloadRequired = true
		}
	}

	if err := writeZoneManifest(database, knotZonesFile); err != nil {
		return err
	}

	if reloadRequired {
		// Reloads the knot daemon to pick up the latest configuration
		if err := exec.Command("/usr/sbin/knotc", "reload").Run(); err != nil {
			return err
		}
	}

	return nil
}
