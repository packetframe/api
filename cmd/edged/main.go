package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/miekg/dns"
	log "github.com/sirupsen/logrus"
	"go.kuoruan.net/v8go-polyfills/fetch"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	v8 "rogchap.com/v8go"

	"github.com/packetframe/api/internal/common/db"
)

var (
	dnsListenAddr   = flag.String("dns-listen", ":5354", "DNS listen address")
	rpcListenAddr   = flag.String("rpc-listen", ":8083", "RPC listen address")
	dbHost          = flag.String("db-host", "localhost", "postgres database host")
	refreshInterval = flag.String("refresh", "30s", "script refresh interval")
	verbose         = flag.Bool("verbose", false, "enable verbose logging")
)

var scriptCache map[string]string

type RR struct {
	Name  string `json:"name"`
	TTL   uint32 `json:"ttl"`
	Type  string `json:"type"`
	Value string `json:"value"`
}

// ToDNSRR converts the RR to a dns.RR
func (rr *RR) ToDNSRR() (dns.RR, error) {
	return dns.NewRR(fmt.Sprintf("%s %d IN %s %s", rr.Name, rr.TTL, rr.Type, rr.Value))
}

type Answer struct {
	RRs           []RR `json:"rrs"`
	Authoritative bool `json:"authoritative"`
}

// ToRRSet converts this answer into a slice of dns.RR
func (a *Answer) ToRRSet() ([]dns.RR, error) {
	var rrSet []dns.RR
	for _, rr := range a.RRs {
		dnsRR, err := rr.ToDNSRR()
		if err != nil {
			return nil, err
		}
		rrSet = append(rrSet, dnsRR)
	}
	return rrSet, nil
}

// dnsQuestionMsgToObject converts a DNS message to a v8 object
func dnsQuestionMsgToObject(iso *v8.Isolate, ctx *v8.Context, m *dns.Msg) (*v8.Object, error) {
	question := dns.Question{
		Name:   "",
		Qtype:  0,
		Qclass: 0,
	}
	if len(m.Question) > 0 {
		question = m.Question[0]
	}

	// Create the response object to pass to the handler function
	resp := v8.NewObjectTemplate(iso)
	if err := resp.Set("name", question.Name); err != nil {
		return nil, err
	}
	if err := resp.Set("type", dns.TypeToString[question.Qtype]); err != nil {
		return nil, err
	}

	// Add extra section
	for _, rr := range m.Extra {
		for _, o := range rr.(*dns.OPT).Option {
			switch o.(type) {
			case *dns.EDNS0_COOKIE:
				if err := resp.Set("cookie", o.String()); err != nil {
					return nil, err
				}
			case *dns.EDNS0_SUBNET:
				if err := resp.Set("subnet", o.String()); err != nil {
					return nil, err
				}
			}
		}
	}

	respInstance, err := resp.NewInstance(ctx)
	if err != nil {
		return nil, err
	}
	return respInstance, nil
}

// newScript creates a new isolate for a script
func newScript(scriptContents, origin string) (*v8.Isolate, *v8.Context, error) {
	iso := v8.NewIsolate()

	//printfn := v8.NewFunctionTemplate(iso, func(info *v8.FunctionCallbackInfo) *v8.Value {
	//	fmt.Printf("%v", info.Args())
	//	return nil
	//})
	//global.Set("print", printfn)

	// Get the global object
	global := v8.NewObjectTemplate(iso)

	// Inject the fetch polyfill into the isolate's global object
	if err := fetch.InjectTo(iso, global); err != nil {
		return nil, nil, err
	}

	ctx := v8.NewContext(iso, global)
	_, err := ctx.RunScript(scriptContents, origin)
	if err != nil {
		return nil, nil, err
	}

	return iso, ctx, nil
}

// loadRecord loads a record into the DNS handler
func loadRecord(label, script string) {
	log.Debugf("Loading zone script %s", label)

	iso, ctx, err := newScript(script, strings.TrimSuffix(label, "."))
	if err != nil {
		log.Fatal(err)
	}

	handleQueryVal, err := ctx.Global().Get("handleQuery")
	if err != nil {
		log.Fatalf("unable to find handleQuery")
	}
	handleQuery, err := handleQueryVal.AsFunction()
	if err != nil {
		log.Fatalf("unable to retreive handleQuery as function")
	}

	dns.HandleRemove(label)
	dns.HandleFunc(label, func(w dns.ResponseWriter, r *dns.Msg) {
		m := new(dns.Msg)
		m.SetReply(r)

		optArg, err := dnsQuestionMsgToObject(iso, ctx, r)
		if err != nil {
			log.Fatalf("unable to convert DNS message to object: %s", err)
		}

		var handlerResultPromise *v8.Promise
		done := make(chan bool, 1)
		go func() {
			handlerResultVal, err := handleQuery.Call(ctx.Global(), optArg)
			if err != nil {
				if strings.Contains(err.Error(), "script execution has been terminated") {
					return
				} else {
					log.Fatalf("calling handleQuery: %s", err)
				}
			}
			handlerResultPromise, err = handlerResultVal.AsPromise()
			if err != nil {
				log.Fatalf("aspromise: %s", err)
			}

			for handlerResultPromise.State() == v8.Pending {
				continue
			}
			done <- true
		}()

		// Timeout after 1 second
		select {
		case <-time.After(1 * time.Second):
			iso.TerminateExecution()
			break
		case <-done:
			// Convert the returned object to an Answer struct
			handlerResultJSONStr, err := v8.JSONStringify(ctx, handlerResultPromise.Result())
			if err != nil {
				log.Warnf("unable to convert handler result to object: %s", err)
				break
			}
			var answer Answer
			if err := json.Unmarshal([]byte(handlerResultJSONStr), &answer); err != nil {
				log.Warnf("unable to unmarhsal JSON: %s (%s)", err, handlerResultJSONStr)
				break
			}

			rrSet, err := answer.ToRRSet()
			if err != nil {
				log.Warnf("unable to convert to RR set: %s", err)
				break
			}

			m.Answer = rrSet
			m.Authoritative = answer.Authoritative
		}

		if err := w.WriteMsg(m); err != nil {
			log.Warnf("dns write message: %s", err)
		}
	})
}

// cached checks if a script is in the cache and updates the cache with the new value
func cached(label, script string) bool {
	if scriptCache == nil {
		scriptCache = map[string]string{}
	}

	cachedScript, isCached := scriptCache[label]

	if !isCached || cachedScript != script {
		// Update the cache and return
		scriptCache[label] = script
		return false
	}

	return true
}

// loadRecordHandlers loads DNS record handlers from the database
func loadRecordHandlers(database *gorm.DB) {
	scriptRecords, err := db.ScriptRecords(database)
	if err != nil {
		log.Fatal(err)
	}

	for label, script := range scriptRecords {
		if !cached(label, script) {
			loadRecord(label, script)
		}
	}
}

func main() {
	flag.Parse()
	if *verbose {
		log.SetLevel(log.DebugLevel)
	}

	log.Println("Connecting to database")
	database, err := gorm.Open(postgres.Open(fmt.Sprintf("host=%s user=api password=api dbname=api port=5432 sslmode=disable", *dbHost)), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	// Update public suffix list on a ticker
	refresh, err := time.ParseDuration(*refreshInterval)
	refreshTicker := time.NewTicker(refresh)
	go func() {
		for range refreshTicker.C {
			log.Debug("Refreshing")
			loadRecordHandlers(database)
		}
	}()

	loadRecordHandlers(database)

	log.Printf("Starting DNS server on %s", *dnsListenAddr)
	go func() {
		srv := &dns.Server{Addr: *dnsListenAddr, Net: "udp"}
		if err := srv.ListenAndServe(); err != nil {
			log.Fatalf("failed to start UDP listener: %s", err)
		}
	}()
	go func() {
		srv := &dns.Server{Addr: *dnsListenAddr, Net: "tcp"}
		if err := srv.ListenAndServe(); err != nil {
			log.Fatalf("failed to start TCP listener: %s", err)
		}
	}()

	http.HandleFunc("/refresh", func(w http.ResponseWriter, r *http.Request) {
		loadRecordHandlers(database)
		fmt.Fprint(w, "Refreshed")
	})

	log.Infof("Starting RPC server on %s", *rpcListenAddr)
	if err := http.ListenAndServe(*rpcListenAddr, nil); err != nil {
		log.Fatal(err)
	}
}
