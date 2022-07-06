package caddy

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"

	"gorm.io/gorm"

	"github.com/packetframe/api/internal/common/db"
	"github.com/packetframe/api/internal/common/util"
)

// writeIfDiff writes contents to filename if the contents aren't already identical. Returns if the file was modified.
func writeIfDiff(filename, contents string) (bool, error) {
	newHash, err := util.SHA256(contents)
	if err != nil {
		return false, err
	}

	fileHash, err := util.SHA256File(filename)
	if err != nil || newHash != fileHash { // If unable to read file or if hashes don't match
		if err := os.WriteFile(filename, []byte(contents), 0644); err != nil {
			return false, err
		}
		return true, nil

	}
	return false, nil
}

// Update writes a new Caddyfile with proxied record configurations
func Update(database *gorm.DB, caddyFilePath, nodeId, certDir string) error {
	var caddyPrefix string

	// Write credentials
	certReloadRequired := false
	credentials, err := db.CredentialList(database)
	if err != nil {
		return err
	}
	for _, credential := range credentials {
		modified, err := writeIfDiff(path.Join(certDir, credential.FQDN+".cert"), credential.Cert)
		if err != nil {
			return err
		}
		if modified {
			certReloadRequired = true
		}

		modified, err = writeIfDiff(path.Join(certDir, credential.FQDN+".key"), credential.Key)
		if err != nil {
			return err
		}
		if modified {
			certReloadRequired = true
		}
	}

	// Write Caddyfile
	zones, err := db.ZoneList(database)
	if err != nil {
		return err
	}

	config := map[string][]string{} // domain:[]upstream IPs

	for _, zone := range zones {
		records, err := db.RecordList(database, zone.ID)
		if err != nil {
			return err
		}

		for _, record := range records {
			if record.Proxy {
				domain := record.Label
				if domain == "@" {
					domain = zone.Zone
				} else if !strings.HasSuffix(domain, zone.Zone) {
					domain += "." + zone.Zone
				}
				domain = strings.TrimSuffix(domain, ".")

				upstreamAddr := record.Value
				if record.Type == "AAAA" {
					upstreamAddr = fmt.Sprintf("[%s]", upstreamAddr)
				}
				upstreamAddr = "https://" + upstreamAddr

				// Add the IP to the config
				if config[domain] == nil {
					config[domain] = []string{upstreamAddr}
				} else {
					config[domain] = append(config[domain], upstreamAddr)
				}
			}
		}
	}

	caddyFile := caddyPrefix
	for domain, ips := range config {
		// Check if we have a TLS certificate for this domain
		tlsDirective := "internal"
		if _, err := os.Stat(path.Join(certDir, domain+".cert")); err == nil {
			tlsDirective = path.Join(certDir, domain+".cert") + " " + path.Join(certDir, domain+".key")
		}

		caddyFile += domain + ` {
    tls ` + tlsDirective + `
    reverse_proxy /.well-known/acme-challenge/* {
        to http://172.16.90.1:8081
    }
    reverse_proxy {
        to ` + strings.Join(ips, " ") + `
        lb_policy round_robin
        header_up X-Packetframe-PoP "` + nodeId + `"
        header_up Host ` + domain + `
        transport http {
            tls
            tls_insecure_skip_verify
            tls_server_name ` + domain + `
            dial_timeout 5s
            response_header_timeout 30s
        }
    }
}
`
	}

	// Write Caddyfile
	caddyfileModified, err := writeIfDiff(caddyFilePath, caddyFile)
	if err != nil {
		return err
	}

	// Reload running caddy config
	if caddyfileModified || certReloadRequired {
		if err := exec.Command("caddy", "reload", "-config", caddyFile).Run(); err != nil {
			return err
		}
	}

	return nil
}
